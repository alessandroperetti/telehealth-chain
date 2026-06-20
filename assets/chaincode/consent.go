package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// ConsentContract for patient consent management
type ConsentContract struct {
	contractapi.Contract
}

// Consent represents a patient's consent grant
type Consent struct {
	ConsentID   string    `json:"consentId"`
	PatientID   string    `json:"patientId"`
	GrantedTo   string    `json:"grantedTo"`
	DataScope   string    `json:"dataScope"`   // ALL, RECORDS, PAYMENTS, DEMOGRAPHICS
	Purpose     string    `json:"purpose"`
	GrantedAt   time.Time `json:"grantedAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
	RevokedAt   time.Time `json:"revokedAt,omitempty"`
	Status      string    `json:"status"`      // ACTIVE, REVOKED, EXPIRED
}

// Consent Functions ==============================================

// GrantConsent creates a new consent record for a patient
func (cc *ConsentContract) GrantConsent(ctx contractapi.TransactionContextInterface,
	consentID string, patientID string, grantedTo string, dataScope string, purpose string, durationDays int) error {

	// Verify the caller is the patient (or authorized representative)
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	// Check role - only patients can grant consent
	role, found, err := ctx.GetClientIdentity().GetAttributeValue("role")
	if err != nil {
		return fmt.Errorf("failed to get role attribute: %v", err)
	}
	if found && role != "patient" && role != "admin" {
		return fmt.Errorf("only patients or admins can grant consent, caller role: %s", role)
	}

	// Check if consent already exists
	existingJSON, err := ctx.GetStub().GetState("Consent_" + consentID)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if existingJSON != nil {
		return fmt.Errorf("consent %s already exists", consentID)
	}

	// Validate data scope
	validScopes := map[string]bool{"ALL": true, "RECORDS": true, "PAYMENTS": true, "DEMOGRAPHICS": true}
	if !validScopes[dataScope] {
		return fmt.Errorf("invalid data scope: %s. Must be ALL, RECORDS, PAYMENTS, or DEMOGRAPHICS", dataScope)
	}

	now := time.Now()
	consent := Consent{
		ConsentID: consentID,
		PatientID: patientID,
		GrantedTo: grantedTo,
		DataScope: dataScope,
		Purpose:   purpose,
		GrantedAt: now,
		ExpiresAt: now.AddDate(0, 0, durationDays),
		Status:    "ACTIVE",
	}

	consentJSON, err := json.Marshal(consent)
	if err != nil {
		return err
	}

	// Log the consent grant as an access event on the patient record
	_ = logAccessEvent(ctx, patientID, clientID, "Consent granted to "+grantedTo)

	return ctx.GetStub().PutState("Consent_"+consentID, consentJSON)
}

// RevokeConsent revokes an existing consent
func (cc *ConsentContract) RevokeConsent(ctx contractapi.TransactionContextInterface, consentID string) error {

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	consent, err := cc.GetConsent(ctx, consentID)
	if err != nil {
		return err
	}

	if consent.Status != "ACTIVE" {
		return fmt.Errorf("consent %s is not active, current status: %s", consentID, consent.Status)
	}

	consent.Status = "REVOKED"
	consent.RevokedAt = time.Now()

	consentJSON, err := json.Marshal(consent)
	if err != nil {
		return err
	}

	// Log the revocation
	_ = logAccessEvent(ctx, consent.PatientID, clientID, "Consent revoked for "+consent.GrantedTo)

	return ctx.GetStub().PutState("Consent_"+consentID, consentJSON)
}

// CheckConsent verifies if a specific entity has active consent for a patient
func (cc *ConsentContract) CheckConsent(ctx contractapi.TransactionContextInterface,
	patientID string, entityID string, dataScope string) (bool, error) {

	// Query all consents using a partial composite key approach
	// For simplicity, we iterate through known consent IDs
	// In production, use CouchDB rich queries or composite keys
	resultsIterator, err := ctx.GetStub().GetStateByRange("Consent_", "Consent_~")
	if err != nil {
		return false, fmt.Errorf("failed to query consents: %v", err)
	}
	defer resultsIterator.Close()

	now := time.Now()

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return false, err
		}

		var consent Consent
		err = json.Unmarshal(queryResponse.Value, &consent)
		if err != nil {
			continue
		}

		// Check if this consent matches the query
		if consent.PatientID == patientID &&
			consent.GrantedTo == entityID &&
			consent.Status == "ACTIVE" &&
			now.Before(consent.ExpiresAt) &&
			(consent.DataScope == "ALL" || consent.DataScope == dataScope) {
			return true, nil
		}
	}

	return false, nil
}

// GetConsent retrieves a specific consent record
func (cc *ConsentContract) GetConsent(ctx contractapi.TransactionContextInterface, consentID string) (*Consent, error) {
	consentJSON, err := ctx.GetStub().GetState("Consent_" + consentID)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if consentJSON == nil {
		return nil, fmt.Errorf("consent %s does not exist", consentID)
	}

	var consent Consent
	err = json.Unmarshal(consentJSON, &consent)
	if err != nil {
		return nil, err
	}

	return &consent, nil
}

// GetPatientConsents retrieves all consent records for a patient
func (cc *ConsentContract) GetPatientConsents(ctx contractapi.TransactionContextInterface, patientID string) ([]*Consent, error) {

	resultsIterator, err := ctx.GetStub().GetStateByRange("Consent_", "Consent_~")
	if err != nil {
		return nil, fmt.Errorf("failed to query consents: %v", err)
	}
	defer resultsIterator.Close()

	var consents []*Consent

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var consent Consent
		err = json.Unmarshal(queryResponse.Value, &consent)
		if err != nil {
			continue
		}

		if consent.PatientID == patientID {
			consents = append(consents, &consent)
		}
	}

	return consents, nil
}

// logAccessEvent is a helper that appends to a patient's access log
func logAccessEvent(ctx contractapi.TransactionContextInterface, patientID string, entityID string, purpose string) error {
	patientJSON, err := ctx.GetStub().GetState(patientID)
	if err != nil || patientJSON == nil {
		return err
	}

	var patient Patient
	err = json.Unmarshal(patientJSON, &patient)
	if err != nil {
		return err
	}

	patient.AccessLog = append(patient.AccessLog, Access{
		Timestamp: time.Now(),
		EntityID:  entityID,
		Purpose:   purpose,
	})

	updatedJSON, err := json.Marshal(patient)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(patientID, updatedJSON)
}

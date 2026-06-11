package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// EHRContract for medical records
type EHRContract struct {
	contractapi.Contract
}

// PaymentContract for financial transactions
type PaymentContract struct {
	contractapi.Contract
}

// ── Data Structures (Alessandro's originals, preserved exactly) ───────────────

// Patient represents EHR structure
type Patient struct {
	AccessLog      []Access `json:"accessLog"`
	DOB            string   `json:"dob"`
	ID             string   `json:"id"`
	MedicalHistory []Record `json:"history"`
	Name           string   `json:"name"`
}

// Record represents medical entry
type Record struct {
	Date      time.Time `json:"date"`
	Diagnosis string    `json:"diagnosis"`
	DoctorID  string    `json:"doctorId"`
	Treatment string    `json:"treatment"`
}

// Access tracks data access
type Access struct {
	EntityID  string    `json:"entityId"`
	Purpose   string    `json:"purpose"`
	Timestamp time.Time `json:"timestamp"`
}

// Payment represents financial transaction
type Payment struct {
	Amount      float64   `json:"amount"`
	ID          string    `json:"id"`
	PatientID   string    `json:"patientId"`
	ProviderID  string    `json:"providerId"`
	ServiceDate time.Time `json:"serviceDate"`
	Status      string    `json:"status"` // PENDING, PAID, DENIED
}

// ── Added: Private data structure for sensitive patient fields ─────────────────

// PatientPrivateData holds sensitive data stored in private collections (Org1 only)
type PatientPrivateData struct {
	PatientID        string `json:"patientId"`
	SSN              string `json:"ssn"`
	InsuranceID      string `json:"insuranceId"`
	BloodType        string `json:"bloodType"`
	Allergies        string `json:"allergies"`
	EmergencyContact string `json:"emergencyContact"`
}

// ── Added: Insurance claim structure ─────────────────────────────────────────

// InsuranceClaim represents a claim submitted by an insurer against a payment
type InsuranceClaim struct {
	ClaimID     string    `json:"claimId"`
	PaymentID   string    `json:"paymentId"`
	PatientID   string    `json:"patientId"`
	InsurerID   string    `json:"insurerId"`
	ClaimAmount float64   `json:"claimAmount"`
	Status      string    `json:"status"` // SUBMITTED, APPROVED, REJECTED
	SubmittedAt time.Time `json:"submittedAt"`
	ProcessedAt time.Time `json:"processedAt,omitempty"`
	Notes       string    `json:"notes,omitempty"`
}

// ── Added: Transaction history structure ──────────────────────────────────────

// HistoryEntry represents a single entry in a key's transaction history
type HistoryEntry struct {
	TxID      string          `json:"txId"`
	Timestamp time.Time       `json:"timestamp"`
	IsDelete  bool            `json:"isDelete"`
	Value     json.RawMessage `json:"value,omitempty"`
}

// ── EHR InitLedger (Alessandro's original, preserved exactly) ─────────────────

func (s *EHRContract) InitLedgerEHR(ctx contractapi.TransactionContextInterface) error {
	patients := []Patient{
		{ID: "patient1", Name: "John Doe", DOB: "1990-01-01", MedicalHistory: []Record{}, AccessLog: []Access{}},
		{ID: "patient2", Name: "Jane Smith", DOB: "1985-05-15", MedicalHistory: []Record{}, AccessLog: []Access{}},
	}

	for _, patient := range patients {
		patientJSON, err := json.Marshal(patient)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(patient.ID, patientJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state. %v", err)
		}
	}

	return nil
}

// ── Payment InitLedger (Alessandro's original, preserved exactly) ─────────────

func (s *PaymentContract) InitLedgerPayment(ctx contractapi.TransactionContextInterface) error {
	payments := []Payment{
		{ID: "payment1", Amount: 100.0, PatientID: "patient1", ProviderID: "provider1", ServiceDate: time.Now(), Status: "PENDING"},
		{ID: "payment2", Amount: 150.0, PatientID: "patient2", ProviderID: "provider2", ServiceDate: time.Now(), Status: "PENDING"},
	}

	for _, payment := range payments {
		paymentJSON, err := json.Marshal(payment)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(payment.ID, paymentJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state. %v", err)
		}
	}

	return nil
}

// ── EHR Functions (Alessandro's originals + RBAC added) ──────────────────────

func (ec *EHRContract) AddPatient(ctx contractapi.TransactionContextInterface, id string, name string, dob string) error {
	// Added: RBAC check
	if err := CheckPermission(ctx, "AddPatient"); err != nil {
		return err
	}

	exists, err := ec.PatientExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("patient %s already exists", id)
	}

	patient := Patient{
		ID:             id,
		Name:           name,
		DOB:            dob,
		MedicalHistory: []Record{},
		AccessLog:      []Access{},
	}

	patientJSON, err := json.Marshal(patient)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, patientJSON)
}

func (ec *EHRContract) AddMedicalRecord(ctx contractapi.TransactionContextInterface, patientID string, diagnosis string, treatment string) error {
	// Added: RBAC check
	if err := CheckPermission(ctx, "AddMedicalRecord"); err != nil {
		return err
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	// Uses getPatientInternal to avoid double RBAC check
	patient, err := ec.getPatientInternal(ctx, patientID)
	if err != nil {
		return err
	}

	record := Record{
		Date:      time.Now(),
		Diagnosis: diagnosis,
		Treatment: treatment,
		DoctorID:  clientID,
	}

	patient.MedicalHistory = append(patient.MedicalHistory, record)
	patient.AccessLog = append(patient.AccessLog, Access{
		Timestamp: time.Now(),
		EntityID:  clientID,
		Purpose:   "Record addition",
	})

	patientJSON, err := json.Marshal(patient)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(patientID, patientJSON)
}

// ── Added: Private data functions ─────────────────────────────────────────────

// SavePatientPrivateData stores sensitive patient data in Org1-only private collection
func (ec *EHRContract) SavePatientPrivateData(ctx contractapi.TransactionContextInterface,
	patientID string, ssn string, insuranceID string, bloodType string, allergies string, emergencyContact string) error {

	if err := CheckPermission(ctx, "AddPatient"); err != nil {
		return err
	}

	exists, err := ec.PatientExists(ctx, patientID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("patient %s does not exist, create patient first", patientID)
	}

	privateData := PatientPrivateData{
		PatientID:        patientID,
		SSN:              ssn,
		InsuranceID:      insuranceID,
		BloodType:        bloodType,
		Allergies:        allergies,
		EmergencyContact: emergencyContact,
	}

	privateDataJSON, err := json.Marshal(privateData)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutPrivateData("patientPrivateData", patientID, privateDataJSON)
}

// GetPatientPrivateData retrieves sensitive patient data from Org1-only private collection
func (ec *EHRContract) GetPatientPrivateData(ctx contractapi.TransactionContextInterface, patientID string) (*PatientPrivateData, error) {
	privateDataJSON, err := ctx.GetStub().GetPrivateData("patientPrivateData", patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to read private data: %v", err)
	}
	if privateDataJSON == nil {
		return nil, fmt.Errorf("private data for patient %s does not exist", patientID)
	}

	var privateData PatientPrivateData
	err = json.Unmarshal(privateDataJSON, &privateData)
	if err != nil {
		return nil, err
	}

	clientID, _ := GetCallerID(ctx)
	_ = logAccessEvent(ctx, patientID, clientID, "Private data accessed")

	return &privateData, nil
}

// ── Added: Transaction history ─────────────────────────────────────────────────

// GetTransactionHistory returns the full Fabric ledger history for any key,
// providing the immutable audit trail described in the paper.
func (ec *EHRContract) GetTransactionHistory(ctx contractapi.TransactionContextInterface, key string) ([]*HistoryEntry, error) {
	if err := CheckPermission(ctx, "GetPatient"); err != nil {
		return nil, err
	}

	resultsIterator, err := ctx.GetStub().GetHistoryForKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get history for key %s: %v", key, err)
	}
	defer resultsIterator.Close()

	var history []*HistoryEntry

	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		entry := &HistoryEntry{
			TxID:     response.TxId,
			IsDelete: response.IsDelete,
		}

		if response.Timestamp != nil {
			entry.Timestamp = time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos))
		}

		if !response.IsDelete && response.Value != nil {
			entry.Value = json.RawMessage(response.Value)
		}

		history = append(history, entry)
	}

	return history, nil
}

// ── Payment Functions (Alessandro's originals + RBAC added) ──────────────────

func (pc *PaymentContract) CreatePayment(ctx contractapi.TransactionContextInterface,
	paymentID string, amount float64, patientID string, providerID string) error {

	// Added: RBAC check
	if err := CheckPermission(ctx, "CreatePayment"); err != nil {
		return err
	}

	payment := Payment{
		ID:          paymentID,
		Amount:      amount,
		PatientID:   patientID,
		ProviderID:  providerID,
		ServiceDate: time.Now(),
		Status:      "PENDING",
	}

	paymentJSON, err := json.Marshal(payment)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(paymentID, paymentJSON)
}

func (pc *PaymentContract) UpdatePaymentStatus(ctx contractapi.TransactionContextInterface,
	paymentID string, newStatus string) error {

	// Added: RBAC check
	if err := CheckPermission(ctx, "UpdatePaymentStatus"); err != nil {
		return err
	}

	payment, err := pc.GetPayment(ctx, paymentID)
	if err != nil {
		return err
	}

	// Validate status transition (Alessandro's original logic, preserved)
	if (payment.Status == "PENDING" && newStatus == "PAID") ||
		(payment.Status == "PENDING" && newStatus == "DENIED") {
		payment.Status = newStatus
	} else {
		return fmt.Errorf("invalid status transition")
	}

	paymentJSON, err := json.Marshal(payment)
	if err != nil {
		return err
	}

	// Added: access log entry
	clientID, _ := GetCallerID(ctx)
	_ = logAccessEvent(ctx, payment.PatientID, clientID,
		fmt.Sprintf("Payment %s status changed to %s", paymentID, newStatus))

	return ctx.GetStub().PutState(paymentID, paymentJSON)
}

// ── Added: Insurance claim functions ─────────────────────────────────────────

// SubmitClaim allows an insurer to submit a claim against a PAID payment.
// Claims at or below the AUTO_APPROVE_THRESHOLD (500.0) are auto-approved;
// above it they remain SUBMITTED pending manual ProcessClaim.
func (pc *PaymentContract) SubmitClaim(ctx contractapi.TransactionContextInterface,
	claimID string, paymentID string, claimAmount float64, notes string) error {

	if err := CheckPermission(ctx, "SubmitClaim"); err != nil {
		return err
	}

	payment, err := pc.GetPayment(ctx, paymentID)
	if err != nil {
		return fmt.Errorf("payment not found: %v", err)
	}
	if payment.Status != "PAID" {
		return fmt.Errorf("claims can only be submitted against PAID payments, current status: %s", payment.Status)
	}
	if claimAmount > payment.Amount {
		return fmt.Errorf("claim amount %.2f exceeds payment amount %.2f", claimAmount, payment.Amount)
	}

	insurerID, err := GetCallerID(ctx)
	if err != nil {
		return err
	}

	const autoApproveThreshold = 500.0
	claimStatus := "SUBMITTED"
	if claimAmount <= autoApproveThreshold {
		claimStatus = "APPROVED"
	}

	claim := InsuranceClaim{
		ClaimID:     claimID,
		PaymentID:   paymentID,
		PatientID:   payment.PatientID,
		InsurerID:   insurerID,
		ClaimAmount: claimAmount,
		Status:      claimStatus,
		SubmittedAt: time.Now(),
		Notes:       notes,
	}

	claimJSON, err := json.Marshal(claim)
	if err != nil {
		return err
	}

	_ = logAccessEvent(ctx, payment.PatientID, insurerID,
		fmt.Sprintf("Insurance claim %s submitted (status: %s)", claimID, claimStatus))

	return ctx.GetStub().PutState("Claim_"+claimID, claimJSON)
}

// ProcessClaim allows manual approve/reject of SUBMITTED claims above the threshold
func (pc *PaymentContract) ProcessClaim(ctx contractapi.TransactionContextInterface,
	claimID string, decision string, notes string) error {

	if err := CheckPermission(ctx, "ProcessClaim"); err != nil {
		return err
	}

	if decision != "APPROVED" && decision != "REJECTED" {
		return fmt.Errorf("decision must be APPROVED or REJECTED, got: %s", decision)
	}

	claimJSON, err := ctx.GetStub().GetState("Claim_" + claimID)
	if err != nil {
		return fmt.Errorf("failed to read claim: %v", err)
	}
	if claimJSON == nil {
		return fmt.Errorf("claim %s does not exist", claimID)
	}

	var claim InsuranceClaim
	if err := json.Unmarshal(claimJSON, &claim); err != nil {
		return err
	}

	if claim.Status != "SUBMITTED" {
		return fmt.Errorf("only SUBMITTED claims can be processed, current status: %s", claim.Status)
	}

	claim.Status = decision
	claim.ProcessedAt = time.Now()
	if notes != "" {
		claim.Notes = notes
	}

	updatedJSON, err := json.Marshal(claim)
	if err != nil {
		return err
	}

	processorID, _ := GetCallerID(ctx)
	_ = logAccessEvent(ctx, claim.PatientID, processorID,
		fmt.Sprintf("Insurance claim %s %s", claimID, decision))

	return ctx.GetStub().PutState("Claim_"+claimID, updatedJSON)
}

// ── Common Utilities (Alessandro's originals, preserved exactly) ──────────────

func (ec *EHRContract) PatientExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	patientJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	return patientJSON != nil, nil
}

// GetPatient — public facing, with RBAC check and access logging added
func (ec *EHRContract) GetPatient(ctx contractapi.TransactionContextInterface, id string) (*Patient, error) {
	// Added: RBAC check
	if err := CheckPermission(ctx, "GetPatient"); err != nil {
		return nil, err
	}

	patient, err := ec.getPatientInternal(ctx, id)
	if err != nil {
		return nil, err
	}

	// Added: access log entry
	clientID, _ := GetCallerID(ctx)
	patient.AccessLog = append(patient.AccessLog, Access{
		Timestamp: time.Now(),
		EntityID:  clientID,
		Purpose:   "Patient record accessed",
	})

	updatedJSON, err := json.Marshal(patient)
	if err == nil {
		_ = ctx.GetStub().PutState(id, updatedJSON)
	}

	return patient, nil
}

// getPatientInternal — reads patient without RBAC, for internal use by other functions
func (ec *EHRContract) getPatientInternal(ctx contractapi.TransactionContextInterface, id string) (*Patient, error) {
	patientJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if patientJSON == nil {
		return nil, fmt.Errorf("patient %s does not exist", id)
	}

	var patient Patient
	err = json.Unmarshal(patientJSON, &patient)
	if err != nil {
		return nil, err
	}

	return &patient, nil
}

// GetPayment — Alessandro's original, preserved exactly (uses "Payment_" prefix)
func (pc *PaymentContract) GetPayment(ctx contractapi.TransactionContextInterface, id string) (*Payment, error) {
	paymentJSON, err := ctx.GetStub().GetState("Payment_" + id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if paymentJSON == nil {
		return nil, fmt.Errorf("payment %s does not exist", id)
	}

	var payment Payment
	err = json.Unmarshal(paymentJSON, &payment)
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

// GetPatientPayments retrieves all payments for a given patient
func (pc *PaymentContract) GetPatientPayments(ctx contractapi.TransactionContextInterface, patientID string) ([]*Payment, error) {
	if err := CheckPermission(ctx, "GetPatientPayments"); err != nil {
		return nil, err
	}

	// Note: uses Payment_ prefix namespace consistent with GetPayment
	resultsIterator, err := ctx.GetStub().GetStateByRange("Payment_", "Payment_~")
	if err != nil {
		return nil, fmt.Errorf("failed to query payments: %v", err)
	}
	defer resultsIterator.Close()

	var payments []*Payment
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var payment Payment
		err = json.Unmarshal(queryResponse.Value, &payment)
		if err != nil {
			continue
		}
		if payment.PatientID == patientID {
			payments = append(payments, &payment)
		}
	}

	return payments, nil
}

// ── Added: logAccessEvent helper ──────────────────────────────────────────────

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

// ── main — registers all three contracts ──────────────────────────────────────

func main() {
	chaincode, err := contractapi.NewChaincode(
		&EHRContract{},
		&PaymentContract{},
		&ConsentContract{}, // Added
	)

	if err != nil {
		log.Panicf("Error creating chaincode: %v", err)
	}

	if err := chaincode.Start(); err != nil {
		log.Panicf("Error starting chaincode: %v", err)
	}
}

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

// EHR InitLedger function
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

// Payment InitLedger function
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

// EHR Functions ==============================================

func (ec *EHRContract) AddPatient(ctx contractapi.TransactionContextInterface, id string, name string, dob string) error {
	exists, err := ec.PatientExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("patient %s already exists", id)
	}

	patient := Patient{
		ID:   id,
		Name: name,
		DOB:  dob,
	}

	patientJSON, err := json.Marshal(patient)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, patientJSON)
}

func (ec *EHRContract) AddMedicalRecord(ctx contractapi.TransactionContextInterface, patientID string, diagnosis string, treatment string) error {
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	patient, err := ec.GetPatient(ctx, patientID)
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

// Payment Functions ==========================================

func (pc *PaymentContract) CreatePayment(ctx contractapi.TransactionContextInterface,
	paymentID string, amount float64, patientID string, providerID string) error {

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

	payment, err := pc.GetPayment(ctx, paymentID)
	if err != nil {
		return err
	}

	// Validate status transition
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

	return ctx.GetStub().PutState(paymentID, paymentJSON)
}

// Common Utilities ===========================================

func (ec *EHRContract) PatientExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	patientJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	return patientJSON != nil, nil
}

func (ec *EHRContract) GetPatient(ctx contractapi.TransactionContextInterface, id string) (*Patient, error) {
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

func main() {
	chaincode, err := contractapi.NewChaincode(
		&EHRContract{},
		&PaymentContract{},
	)

	if err != nil {
		log.Panicf("Error creating chaincode: %v", err)
	}

	if err := chaincode.Start(); err != nil {
		log.Panicf("Error starting chaincode: %v", err)
	}
}

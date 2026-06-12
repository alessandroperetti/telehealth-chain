package main

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/hash"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"github.com/hyperledger/fabric-protos-go-apiv2/gateway"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

const (
	mspID            = "Org1MSP"
	cryptoPath       = "../fabric-samples/test-network/organizations/peerOrganizations/org1.example.com"
	certPath         = cryptoPath + "/users/User1@org1.example.com/msp/signcerts"
	keyPath          = cryptoPath + "/users/User1@org1.example.com/msp/keystore"
	tlsCertPath      = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
	peerEndpoint     = "dns:///localhost:7051"
	gatewayPeer      = "peer0.org1.example.com"
	peerHostOverride = "peer0.org1.example.com"
)

func main() {
	// Alessandro's original connection setup — preserved exactly
	clientConnection := newGrpcConnection()
	defer clientConnection.Close()

	id := newIdentity()
	sign := newSign()

	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithHash(hash.SHA256),
		client.WithClientConnection(clientConnection),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	chaincodeName := "ehr-payment"
	if ccname := os.Getenv("CHAINCODE_NAME"); ccname != "" {
		chaincodeName = ccname
	}

	channelName := "mychannel"
	if cname := os.Getenv("CHANNEL_NAME"); cname != "" {
		channelName = cname
	}

	network := gw.GetNetwork(channelName)

	// Alessandro's original used single contract; extended to three
	ehrContract     := network.GetContractWithName(chaincodeName, "EHRContract")
	paymentContract := network.GetContractWithName(chaincodeName, "PaymentContract")
	consentContract := network.GetContractWithName(chaincodeName, "ConsentContract")

	fmt.Println("============================================================")
	fmt.Println("  Telehealth Chain - Demonstration")
	fmt.Println("============================================================")

	// Step 1: Initialize ledger (Alessandro's original flow)
	fmt.Println("\n--- Step 1: Initialize Ledger ---")
	initLedger(ehrContract)

	// Step 2: Query existing patient (Alessandro's original)
	fmt.Println("\n--- Step 2: Query Patient ---")
	getPatient(ehrContract, "patient1")

	// Step 3: Add a new patient
	fmt.Println("\n--- Step 3: Add New Patient ---")
	addPatient(ehrContract, "patient3", "Alice Johnson", "1995-03-20")

	// Step 4: Add a medical record
	fmt.Println("\n--- Step 4: Add Medical Record ---")
	addMedicalRecord(ehrContract, "patient1", "Hypertension", "Prescribed Lisinopril 10mg")

	// Step 5: Verify updated record
	fmt.Println("\n--- Step 5: Verify Updated Patient Record ---")
	getPatient(ehrContract, "patient1")

	// Step 6: Grant consent (patient → insurer, PAYMENTS scope, 365 days)
	fmt.Println("\n--- Step 6: Grant Consent ---")
	grantConsent(consentContract, "consent1", "patient1", "org2-insurer", "PAYMENTS", "Insurance claim processing", 365)

	// Step 7: Verify consent is active
	fmt.Println("\n--- Step 7: Check Consent ---")
	checkConsent(consentContract, "patient1", "org2-insurer", "PAYMENTS")

	// Step 8: Get full consent record
	fmt.Println("\n--- Step 8: Get Consent Details ---")
	getConsent(consentContract, "consent1")

	// Step 9: Create a payment
	fmt.Println("\n--- Step 9: Create Payment ---")
	createPayment(paymentContract, "payment3", 250.0, "patient1", "provider1")

	// Step 10: Process payment
	fmt.Println("\n--- Step 10: Process Payment ---")
	updatePaymentStatus(paymentContract, "payment3", "PAID")

	// Step 11: Submit insurance claim (auto-approved — amount below €500 threshold)
	fmt.Println("\n--- Step 11: Submit Insurance Claim ---")
	submitClaim(paymentContract, "claim1", "payment3", 200.0, "Routine teleconsultation reimbursement")

	// Step 12: Get transaction history for patient1 (immutable audit trail)
	fmt.Println("\n--- Step 12: Get Transaction History ---")
	getTransactionHistory(ehrContract, "patient1")

	// Step 13: Revoke consent
	fmt.Println("\n--- Step 13: Revoke Consent ---")
	revokeConsent(consentContract, "consent1")

	// Step 14: Verify consent is revoked
	fmt.Println("\n--- Step 14: Verify Consent Revoked ---")
	checkConsent(consentContract, "patient1", "org2-insurer", "PAYMENTS")

	fmt.Println("\n============================================================")
	fmt.Println("  Demonstration Complete — 14 Steps")
	fmt.Println("============================================================")
}

// ── EHR Functions ─────────────────────────────────────────────────────────────

// initLedger — Alessandro's original function, preserved exactly
func initLedger(contract *client.Contract) {
	fmt.Printf("\n--> Submit Transaction: InitLedger, function creates the initial set of assets on the ledger \n")
	_, err := contract.SubmitTransaction("InitLedgerEHR")
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}
	fmt.Printf("*** Transaction committed successfully\n")
}

// getPatient — Alessandro's original function, preserved exactly
func getPatient(contract *client.Contract, patientId string) {
	fmt.Println("\n--> Evaluate Transaction: Get specific patient, function returns a specific asset on the ledger")
	evaluateResult, err := contract.EvaluateTransaction("GetPatient", patientId)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)
	fmt.Printf("*** Result:%s\n", result)
}

func addPatient(contract *client.Contract, id string, name string, dob string) {
	fmt.Printf("--> Submit Transaction: AddPatient(%s, %s, %s)\n", id, name, dob)
	_, err := contract.SubmitTransaction("AddPatient", id, name, dob)
	if err != nil {
		handleError(err)
		return
	}
	fmt.Printf("*** Patient %s added successfully\n", id)
}

func addMedicalRecord(contract *client.Contract, patientID string, diagnosis string, treatment string) {
	fmt.Printf("--> Submit Transaction: AddMedicalRecord(%s)\n", patientID)
	_, err := contract.SubmitTransaction("AddMedicalRecord", patientID, diagnosis, treatment)
	if err != nil {
		handleError(err)
		return
	}
	fmt.Println("*** Medical record added successfully")
}

// ── Consent Functions ─────────────────────────────────────────────────────────

func grantConsent(contract *client.Contract, consentID string, patientID string, grantedTo string, dataScope string, purpose string, days int) {
	fmt.Printf("--> Submit Transaction: GrantConsent(%s -> %s, scope=%s)\n", patientID, grantedTo, dataScope)
	_, err := contract.SubmitTransaction("GrantConsent", consentID, patientID, grantedTo, dataScope, purpose, fmt.Sprintf("%d", days))
	if err != nil {
		handleError(err)
		return
	}
	fmt.Printf("*** Consent %s granted successfully\n", consentID)
}

func revokeConsent(contract *client.Contract, consentID string) {
	fmt.Printf("--> Submit Transaction: RevokeConsent(%s)\n", consentID)
	_, err := contract.SubmitTransaction("RevokeConsent", consentID)
	if err != nil {
		handleError(err)
		return
	}
	fmt.Printf("*** Consent %s revoked successfully\n", consentID)
}

func checkConsent(contract *client.Contract, patientID string, entityID string, dataScope string) {
	fmt.Printf("--> Evaluate Transaction: CheckConsent(%s, %s, %s)\n", patientID, entityID, dataScope)
	result, err := contract.EvaluateTransaction("CheckConsent", patientID, entityID, dataScope)
	if err != nil {
		handleError(err)
		return
	}
	fmt.Printf("*** Consent active: %s\n", string(result))
}

func getConsent(contract *client.Contract, consentID string) {
	fmt.Printf("--> Evaluate Transaction: GetConsent(%s)\n", consentID)
	result, err := contract.EvaluateTransaction("GetConsent", consentID)
	if err != nil {
		handleError(err)
		return
	}
	fmt.Printf("*** Result: %s\n", formatJSON(result))
}

// ── Payment Functions ─────────────────────────────────────────────────────────

func createPayment(contract *client.Contract, paymentID string, amount float64, patientID string, providerID string) {
	fmt.Printf("--> Submit Transaction: CreatePayment(%s, %.2f)\n", paymentID, amount)
	_, err := contract.SubmitTransaction("CreatePayment", paymentID, fmt.Sprintf("%f", amount), patientID, providerID)
	if err != nil {
		handleError(err)
		return
	}
	fmt.Printf("*** Payment %s created successfully\n", paymentID)
}

func updatePaymentStatus(contract *client.Contract, paymentID string, newStatus string) {
	fmt.Printf("--> Submit Transaction: UpdatePaymentStatus(%s -> %s)\n", paymentID, newStatus)
	_, err := contract.SubmitTransaction("UpdatePaymentStatus", paymentID, newStatus)
	if err != nil {
		handleError(err)
		return
	}
	fmt.Printf("*** Payment %s status updated to %s\n", paymentID, newStatus)
}

// ── Insurance Claim Functions ─────────────────────────────────────────────────

func submitClaim(contract *client.Contract, claimID string, paymentID string, amount float64, notes string) {
	fmt.Printf("--> Submit Transaction: SubmitClaim(%s against %s, amount=%.2f)\n", claimID, paymentID, amount)
	_, err := contract.SubmitTransaction("SubmitClaim", claimID, paymentID, fmt.Sprintf("%f", amount), notes)
	if err != nil {
		handleError(err)
		return
	}
	fmt.Printf("*** Claim %s submitted successfully\n", claimID)
}

func processClaim(contract *client.Contract, claimID string, decision string, notes string) {
	fmt.Printf("--> Submit Transaction: ProcessClaim(%s -> %s)\n", claimID, decision)
	_, err := contract.SubmitTransaction("ProcessClaim", claimID, decision, notes)
	if err != nil {
		handleError(err)
		return
	}
	fmt.Printf("*** Claim %s processed: %s\n", claimID, decision)
}

// ── Transaction History ───────────────────────────────────────────────────────

func getTransactionHistory(contract *client.Contract, key string) {
	fmt.Printf("--> Evaluate Transaction: GetTransactionHistory(%s)\n", key)
	result, err := contract.EvaluateTransaction("GetTransactionHistory", key)
	if err != nil {
		handleError(err)
		return
	}
	fmt.Printf("*** Audit Trail: %s\n", formatJSON(result))
}

// ── Error Handling — Alessandro's original implementation, preserved exactly ──

func handleError(err error) {
	fmt.Println("*** Successfully caught the error:")

	var endorseErr *client.EndorseError
	var submitErr *client.SubmitError
	var commitStatusErr *client.CommitStatusError
	var commitErr *client.CommitError

	if errors.As(err, &endorseErr) {
		fmt.Printf("Endorse error for transaction %s with gRPC status %v: %s\n", endorseErr.TransactionID, status.Code(endorseErr), endorseErr)
	} else if errors.As(err, &submitErr) {
		fmt.Printf("Submit error for transaction %s with gRPC status %v: %s\n", submitErr.TransactionID, status.Code(submitErr), submitErr)
	} else if errors.As(err, &commitStatusErr) {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Printf("Timeout waiting for transaction %s commit status: %s", commitStatusErr.TransactionID, commitStatusErr)
		} else {
			fmt.Printf("Error obtaining commit status for transaction %s with gRPC status %v: %s\n", commitStatusErr.TransactionID, status.Code(commitStatusErr), commitStatusErr)
		}
	} else if errors.As(err, &commitErr) {
		fmt.Printf("Transaction %s failed to commit with status %d: %s\n", commitErr.TransactionID, int32(commitErr.Code), err)
	} else {
		panic(fmt.Errorf("unexpected error type %T: %w", err, err))
	}

	// Alessandro's original: extract peer/orderer error details from gRPC status
	statusErr := status.Convert(err)
	details := statusErr.Details()
	if len(details) > 0 {
		fmt.Println("Error Details:")
		for _, detail := range details {
			switch detail := detail.(type) {
			case *gateway.ErrorDetail:
				fmt.Printf("- address: %s; mspId: %s; message: %s\n", detail.Address, detail.MspId, detail.Message)
			}
		}
	}
}

// ── Connection Utilities — Alessandro's originals, preserved exactly ──────────

func newGrpcConnection() *grpc.ClientConn {
	certificatePEM, err := os.ReadFile(tlsCertPath)
	if err != nil {
		panic(fmt.Errorf("failed to read TLS certificate file: %w", err))
	}
	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		panic(err)
	}
	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, peerHostOverride)
	connection, err := grpc.NewClient(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		panic(fmt.Errorf("failed to create gRPC connection: %w", err))
	}
	return connection
}

func newIdentity() *identity.X509Identity {
	certificatePEM, err := readFirstFile(certPath)
	if err != nil {
		panic(fmt.Errorf("failed to read certificate file: %w", err))
	}
	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		panic(err)
	}
	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		panic(err)
	}
	return id
}

func newSign() identity.Sign {
	privateKeyPEM, err := readFirstFile(keyPath)
	if err != nil {
		panic(fmt.Errorf("failed to read private key file: %w", err))
	}
	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		panic(err)
	}
	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		panic(err)
	}
	return sign
}

// readFirstFile — Alessandro's original, preserved exactly
func readFirstFile(dirPath string) ([]byte, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}
	fileNames, err := dir.Readdirnames(1)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path.Join(dirPath, fileNames[0]))
}

// formatJSON — Alessandro's original, preserved exactly
func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return prettyJSON.String()
}

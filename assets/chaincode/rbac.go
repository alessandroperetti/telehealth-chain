package main

import (
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Role constants for the Telehealth Chain network
const (
	RolePatient  = "patient"
	RoleDoctor   = "doctor"
	RoleInsurer  = "insurer"
	RoleAdmin    = "admin"
)

// Permission defines what each role can do
// Format: role -> list of allowed actions
var rolePermissions = map[string][]string{
	RolePatient: {
		"GetPatient",
		"GrantConsent",
		"RevokeConsent",
		"GetConsent",
		"GetPatientConsents",
		"CheckConsent",
		"GetPayment",
		"GetPatientPayments",
	},
	RoleDoctor: {
		"GetPatient",
		"AddMedicalRecord",
		"CheckConsent",
		"GetConsent",
		"CreatePayment",
		"GetPayment",
	},
	RoleInsurer: {
		"GetPayment",
		"GetPatientPayments",
		"UpdatePaymentStatus",
		"SubmitClaim",
		"ProcessClaim",
		"CheckConsent",
		"GetTransactionHistory",
	},
	RoleAdmin: {
		"AddPatient",
		"GetPatient",
		"GetTransactionHistory",
		"InitLedgerEHR",
		"InitLedgerPayment",
		"GrantConsent",
		"RevokeConsent",
		"GetConsent",
		"GetPatientConsents",
		"CheckConsent",
		"CreatePayment",
		"GetPayment",
		"GetPatientPayments",
		"UpdatePaymentStatus",
		"AddMedicalRecord",
		"SubmitClaim",
		"ProcessClaim",
	},
}

// GetClientRole extracts the role attribute from the caller's certificate
func GetClientRole(ctx contractapi.TransactionContextInterface) (string, error) {
	role, found, err := ctx.GetClientIdentity().GetAttributeValue("role")
	if err != nil {
		return "", fmt.Errorf("failed to get role attribute: %v", err)
	}

	// If no role attribute is set, default to admin for backward compatibility
	// with existing certificates that don't have role attributes
	if !found || role == "" {
		return RoleAdmin, nil
	}

	// Validate that the role is recognized
	validRoles := map[string]bool{
		RolePatient: true,
		RoleDoctor:  true,
		RoleInsurer: true,
		RoleAdmin:   true,
	}
	if !validRoles[role] {
		return "", fmt.Errorf("unrecognized role: %s", role)
	}

	return role, nil
}

// CheckPermission verifies if the caller has permission to execute a given action
func CheckPermission(ctx contractapi.TransactionContextInterface, action string) error {
	role, err := GetClientRole(ctx)
	if err != nil {
		return err
	}

	permissions, exists := rolePermissions[role]
	if !exists {
		return fmt.Errorf("no permissions defined for role: %s", role)
	}

	for _, perm := range permissions {
		if perm == action {
			return nil
		}
	}

	return fmt.Errorf("access denied: role '%s' does not have permission for action '%s'", role, action)
}

// GetCallerID is a convenience wrapper to get the caller's identity
func GetCallerID(ctx contractapi.TransactionContextInterface) (string, error) {
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("failed to get client identity: %v", err)
	}
	return clientID, nil
}

// GetCallerMSPID returns the MSP ID of the calling organization
func GetCallerMSPID(ctx contractapi.TransactionContextInterface) (string, error) {
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", fmt.Errorf("failed to get MSP ID: %v", err)
	}
	return mspID, nil
}

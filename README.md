# HealthLedger — Hyperledger Fabric Implementation

A blockchain-based Electronic Health Record (EHR) and payment management system built on Hyperledger Fabric. Implements patient consent workflows, role-based access control, private data collections, and automated payment processing.

## Repository Structure

```
HealthLedger-chain/
├── assets/
│   └── chaincode/
│       ├── ehr-payment.go          # EHR + Payment + Claim smart contracts (core)
│       ├── consent.go              # Patient consent management contract
│       ├── rbac.go                 # Role-based access control
│       ├── collections_config.json # Private data collection definitions
│       ├── go.mod
│       └── go.sum
├── applications/
│   ├── ehr.go                      # Gateway client — full 14-step demo workflow
│   ├── go.mod
│   └── go.sum
├── network/
│   ├── docker-compose.yaml         # 2-org Fabric network (Orderer + Org1 + Org2)
│   ├── connection-profile-org1.json # Org1 (Hospital) connection profile
│   └── connection-profile-org2.json # Org2 (Insurer) connection profile
├── Makefile                        # Network lifecycle commands
└── README.md
```

## Smart Contracts

### EHRContract
Manages patient records on-chain with full access logging.
- `InitLedgerEHR` — seed ledger with sample patients
- `AddPatient` — register a new patient (admin only)
- `AddMedicalRecord` — append diagnosis/treatment to a patient (doctors + admins)
- `GetPatient` — retrieve patient record with RBAC check and access log entry
- `PatientExists` — existence check
- `GetTransactionHistory` — retrieve full Fabric ledger history for any key, providing an immutable audit trail of all modifications
- `SavePatientPrivateData` — store sensitive fields (SSN, insurance ID, blood type) in private collection
- `GetPatientPrivateData` — retrieve from private collection (Org1 only)

### PaymentContract
Handles payment lifecycle and insurance claim processing with validated status transitions.
- `InitLedgerPayment` — seed ledger with sample payments
- `CreatePayment` — create a new PENDING payment
- `UpdatePaymentStatus` — transition PENDING → PAID or PENDING → DENIED
- `GetPayment` — retrieve payment by ID
- `GetPatientPayments` — retrieve all payments for a patient
- `SubmitClaim` — insurer submits a claim against a PAID payment; auto-approves if amount ≤ €500 threshold
- `ProcessClaim` — manual approve/reject of SUBMITTED claims above threshold

### ConsentContract
Fine-grained, expirable patient consent management.
- `GrantConsent` — patient grants scoped consent to an entity (ALL / RECORDS / PAYMENTS / DEMOGRAPHICS)
- `RevokeConsent` — patient revokes an active consent
- `CheckConsent` — verify active consent at runtime (used internally by other contracts)
- `GetConsent` — retrieve a specific consent record
- `GetPatientConsents` — retrieve all consent records for a patient

## Role-Based Access Control

Roles are encoded in the caller's X.509 certificate as a `role` attribute.

| Role    | Permitted Actions |
|---------|-------------------|
| patient | GetPatient, GrantConsent, RevokeConsent, GetConsent, GetPatientConsents, CheckConsent, GetPayment, GetPatientPayments |
| doctor  | GetPatient, AddMedicalRecord, CheckConsent, GetConsent, CreatePayment, GetPayment |
| insurer | GetPayment, GetPatientPayments, UpdatePaymentStatus, SubmitClaim, ProcessClaim, CheckConsent |
| admin   | All operations |

## Private Data Collections

| Collection | Accessible by | Contents |
|---|---|---|
| `patientPrivateData` | Org1 only | SSN, insurance ID, blood type, allergies, emergency contact |
| `paymentPrivateData` | Org1 + Org2 | Sensitive payment details |
| `consentRecords` | Org1 + Org2 | Consent grant/revoke events |

## Prerequisites

- Docker & Docker Compose
- Go 1.20+
- Hyperledger Fabric binaries (`fabric-samples` in parent directory)

```bash
# Install Fabric samples and binaries
curl -sSL https://bit.ly/2ysbOFE | bash -s
```

## Deployment

```bash
# Start the test network, deploy chaincode, run demo
make all

# Individual steps
make download     # Pull Fabric Docker images
make start        # Start 2-org test network
make deploy       # Package, install, approve, commit chaincode
make invoke       # Run sample transactions
make query        # Query patient1

# Teardown
make clean
```

## Running the Application

```bash
cd applications
go run ehr.go
```

The demo application walks through a 12-step workflow:
1. Initialize ledger
2. Query existing patient
3. Add new patient
4. Add medical record
5. Verify updated record
6. Grant consent (patient → insurer, PAYMENTS scope, 365 days)
7. Check consent (returns true)
8. Get consent details
9. Create payment
10. Process payment (PENDING → PAID)
11. Revoke consent
12. Verify consent revoked (returns false)

## Known Limitations

- `CheckConsent` uses a linear range scan over all consent keys — suitable for demonstration; production deployments should use CouchDB rich queries with composite keys.
- FHIR resource alignment is planned as a future extension; current data structures are custom flat schemas.
- Payment automation is status-based (CRUD); conditional trigger logic (e.g., auto-approve below threshold) is a future extension.

## Authors

- Alessandro Peretti — initial implementation, network configuration
- Syed Sarosh Mahdi — consent management, RBAC, private data collections, application layer

## License

Apache 2.0


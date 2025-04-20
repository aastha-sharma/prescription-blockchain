package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
)

// PrescriptionContract provides functions for managing prescriptions
type PrescriptionContract struct {
}

// Prescription represents a medication prescription
type Prescription struct {
	ID             string   `json:"id"`
	PatientID      string   `json:"patientId"`
	DoctorID       string   `json:"doctorId"`
	MedicationName string   `json:"medicationName"`
	Dosage         string   `json:"dosage"`
	Quantity       int      `json:"quantity"`
	RefillsAllowed int      `json:"refillsAllowed"`
	RefillsUsed    int      `json:"refillsUsed"`
	IssueDate      string   `json:"issueDate"`
	ExpiryDate     string   `json:"expiryDate"`
	Status         string   `json:"status"`
	DocumentHash   string   `json:"documentHash"`
	MedHistory     []string `json:"medHistory"`
}

// Init initializes chaincode
func (c *PrescriptionContract) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

// Invoke is called per transaction
func (c *PrescriptionContract) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	function, args := stub.GetFunctionAndParameters()

	switch function {
	case "createPrescription":
		return c.createPrescription(stub, args)
	case "getPrescription":
		return c.getPrescription(stub, args)
	case "requestRefill":
		return c.requestRefill(stub, args)
	case "approveRefill":
		return c.approveRefill(stub, args)
	case "getPatientHistory":
		return c.getPatientHistory(stub, args)
	case "updatePrescription":
		return c.updatePrescription(stub, args)
	default:
		return shim.Error("Invalid function name")
	}
}

// createPrescription issues a new prescription
func (c *PrescriptionContract) createPrescription(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 8 {
		return shim.Error("Incorrect number of arguments. Expecting 8")
	}

	id := args[0]
	patientID := args[1]
	doctorID := args[2]
	medicationName := args[3]
	dosage := args[4]
	quantity, _ := strconv.Atoi(args[5])
	refillsAllowed, _ := strconv.Atoi(args[6])
	documentHash := args[7]

	// Check if prescription already exists
	prescriptionAsBytes, err := stub.GetState(id)
	if err != nil {
		return shim.Error("Failed to get prescription: " + err.Error())
	} else if prescriptionAsBytes != nil {
		return shim.Error("This prescription already exists: " + id)
	}

	// Create prescription object
	issueDate := time.Now().Format(time.RFC3339)
	expiryDate := time.Now().AddDate(0, 6, 0).Format(time.RFC3339) // 6 months validity

	prescription := Prescription{
		ID:             id,
		PatientID:      patientID,
		DoctorID:       doctorID,
		MedicationName: medicationName,
		Dosage:         dosage,
		Quantity:       quantity,
		RefillsAllowed: refillsAllowed,
		RefillsUsed:    0,
		IssueDate:      issueDate,
		ExpiryDate:     expiryDate,
		Status:         "ACTIVE",
		DocumentHash:   documentHash,
		MedHistory:     []string{"Prescription created on " + issueDate},
	}

	// Convert to JSON
	prescriptionJSON, err := json.Marshal(prescription)
	if err != nil {
		return shim.Error("Failed to marshal prescription: " + err.Error())
	}

	// Put in state
	err = stub.PutState(id, prescriptionJSON)
	if err != nil {
		return shim.Error(err.Error())
	}

	// Create a composite key for patient history lookup
	patientPrescKey, err := stub.CreateCompositeKey("patient~prescription", []string{patientID, id})
	if err != nil {
		return shim.Error(err.Error())
	}

	// Store the composite key
	err = stub.PutState(patientPrescKey, []byte{0})
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// getPrescription returns prescription with given ID
func (c *PrescriptionContract) getPrescription(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	prescriptionAsBytes, err := stub.GetState(args[0])
	if err != nil {
		return shim.Error("Failed to get prescription: " + err.Error())
	} else if prescriptionAsBytes == nil {
		return shim.Error("Prescription not found: " + args[0])
	}

	return shim.Success(prescriptionAsBytes)
}

// requestRefill allows a patient to request a refill
func (c *PrescriptionContract) requestRefill(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	id := args[0]

	prescriptionAsBytes, err := stub.GetState(id)
	if err != nil {
		return shim.Error("Failed to get prescription: " + err.Error())
	} else if prescriptionAsBytes == nil {
		return shim.Error("Prescription not found: " + id)
	}

	var prescription Prescription
	err = json.Unmarshal(prescriptionAsBytes, &prescription)
	if err != nil {
		return shim.Error("Failed to unmarshal prescription: " + err.Error())
	}

	// Check if refills are available
	if prescription.RefillsUsed >= prescription.RefillsAllowed {
		return shim.Error("No refills remaining for prescription: " + id)
	}

	// Check if prescription is active
	if prescription.Status != "ACTIVE" {
		return shim.Error("Prescription is not active: " + id)
	}

	// Check if prescription is expired
	expiryDate, _ := time.Parse(time.RFC3339, prescription.ExpiryDate)
	if time.Now().After(expiryDate) {
		return shim.Error("Prescription has expired: " + id)
	}

	// Update status to REFILL_REQUESTED
	prescription.Status = "REFILL_REQUESTED"

	// Update history
	requestDate := time.Now().Format(time.RFC3339)
	prescription.MedHistory = append(
		prescription.MedHistory,
		"Refill requested on "+requestDate,
	)

	prescriptionJSON, err := json.Marshal(prescription)
	if err != nil {
		return shim.Error("Failed to marshal prescription: " + err.Error())
	}

	err = stub.PutState(id, prescriptionJSON)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// approveRefill allows a pharmacy to approve a refill
func (c *PrescriptionContract) approveRefill(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2: prescriptionID and pharmacyID")
	}

	id := args[0]
	pharmacyID := args[1]

	prescriptionAsBytes, err := stub.GetState(id)
	if err != nil {
		return shim.Error("Failed to get prescription: " + err.Error())
	} else if prescriptionAsBytes == nil {
		return shim.Error("Prescription not found: " + id)
	}

	var prescription Prescription
	err = json.Unmarshal(prescriptionAsBytes, &prescription)
	if err != nil {
		return shim.Error("Failed to unmarshal prescription: " + err.Error())
	}

	// Check if refill was requested
	if prescription.Status != "REFILL_REQUESTED" {
		return shim.Error("Refill was not requested for prescription: " + id)
	}

	// Update refill count and status
	prescription.RefillsUsed++
	prescription.Status = "ACTIVE"

	// Update history
	approveDate := time.Now().Format(time.RFC3339)
	prescription.MedHistory = append(
		prescription.MedHistory,
		"Refill approved by pharmacy "+pharmacyID+" on "+approveDate,
	)

	// If no more refills, mark as COMPLETED
	if prescription.RefillsUsed >= prescription.RefillsAllowed {
		prescription.Status = "COMPLETED"
		prescription.MedHistory = append(
			prescription.MedHistory,
			"Prescription marked as completed on "+approveDate+" - no more refills",
		)
	}

	prescriptionJSON, err := json.Marshal(prescription)
	if err != nil {
		return shim.Error("Failed to marshal prescription: " + err.Error())
	}

	err = stub.PutState(id, prescriptionJSON)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// getPatientHistory returns all prescriptions for a patient
func (c *PrescriptionContract) getPatientHistory(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1: patientID")
	}

	patientID := args[0]

	// Get all prescriptions for this patient using composite key
	resultsIterator, err := stub.GetStateByPartialCompositeKey("patient~prescription", []string{patientID})
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	// Buffer to store prescription IDs
	var buffer []string

	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		// Get the prescription ID from composite key
		_, compositeKeyParts, err := stub.SplitCompositeKey(response.Key)
		if err != nil {
			return shim.Error(err.Error())
		}

		prescriptionID := compositeKeyParts[1]
		buffer = append(buffer, prescriptionID)
	}

	// Convert to JSON
	bufferJSON, err := json.Marshal(buffer)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(bufferJSON)
}

// updatePrescription updates an existing prescription
func (c *PrescriptionContract) updatePrescription(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 5 {
		return shim.Error("Incorrect number of arguments. Expecting 5: id, doctorID, medicationName, dosage, comment")
	}

	id := args[0]
	doctorID := args[1]
	medicationName := args[2]
	dosage := args[3]
	comment := args[4]

	prescriptionAsBytes, err := stub.GetState(id)
	if err != nil {
		return shim.Error("Failed to get prescription: " + err.Error())
	} else if prescriptionAsBytes == nil {
		return shim.Error("Prescription not found: " + id)
	}

	var prescription Prescription
	err = json.Unmarshal(prescriptionAsBytes, &prescription)
	if err != nil {
		return shim.Error("Failed to unmarshal prescription: " + err.Error())
	}

	// Verify doctor has permission to update
	if prescription.DoctorID != doctorID {
		return shim.Error("Doctor " + doctorID + " is not authorized to update this prescription")
	}

	// Only allow updates to active prescriptions
	if prescription.Status != "ACTIVE" {
		return shim.Error("Cannot update prescription with status: " + prescription.Status)
	}

	// Update prescription
	updateDate := time.Now().Format(time.RFC3339)

	// Record previous values
	historyEntry := "Updated on " + updateDate + ": "

	if medicationName != "" && medicationName != prescription.MedicationName {
		historyEntry += "Medication changed from " + prescription.MedicationName + " to " + medicationName + ". "
		prescription.MedicationName = medicationName
	}

	if dosage != "" && dosage != prescription.Dosage {
		historyEntry += "Dosage changed from " + prescription.Dosage + " to " + dosage + ". "
		prescription.Dosage = dosage
	}

	if comment != "" {
		historyEntry += "Comment: " + comment
	}

	prescription.MedHistory = append(prescription.MedHistory, historyEntry)

	prescriptionJSON, err := json.Marshal(prescription)
	if err != nil {
		return shim.Error("Failed to marshal prescription: " + err.Error())
	}

	err = stub.PutState(id, prescriptionJSON)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// Main function starts the chaincode
func main() {
	err := shim.Start(new(PrescriptionContract))
	if err != nil {
		fmt.Printf("Error starting prescription chaincode: %s", err)
	}
}

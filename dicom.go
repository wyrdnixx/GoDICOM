package main

import (
	"fmt"
	"log"

	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
)

func getDicomData(filename string) (string, string, error) {
	log.Println(filename)
	//fmt.Println("FileName: ", filename)
	// Open the DICOM file
	//fmt.Println("Processing file:", filename)

	// Open and parse the DICOM file
	dcm, err := dicom.ParseFile(filename, nil)
	if err != nil {
		//log.Printf("Error parsing DICOM file - no valid DICOM %s: %v\n", filename, err)
		return "", "", err
	}

	// Extract the PatientName tag
	PatientName := ""
	PatientID := ""

	ePatientName, _ := dcm.FindElementByTag(tag.PatientName)
	ePatientID, _ := dcm.FindElementByTag(tag.PatientID)
	//patientName, err := dcm.FindElementByTag(tag.PatientName)

	if ePatientName != nil {
		PatientName = fmt.Sprintf("%v", ePatientName.Value.GetValue())
		//PatientName = ePatientName.Value.GetValue().([]string)

	} else {
		//log.Printf("Patient Name tag not found in file %s\n", filename)
		PatientName = "tag.PatientName not found"
	}

	if ePatientID != nil {
		//PatientID = ePatientID.Value.GetValue().(string)
		PatientID = fmt.Sprintf("%v", ePatientID.Value.GetValue())

	} else {
		//log.Printf("Patient Name tag not found in file %s\n", filename)
		PatientID = "tag.PatientID not found"
	}

	// return fmt.Sprintf("%v", ePatientName.Value.GetValue()), nil
	return PatientName, PatientID, nil
}

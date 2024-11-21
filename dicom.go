package main

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
	//"github.com/gradienthealth/dicom"
)

// ToDo : Insitution Name
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

const (
	dicomHost = "127.0.0.1" // DICOM server address
	dicomPort = "11112"     // DICOM server port
	aet       = "MYAET"     // Calling AET (Application Entity Title)
	calledAet = "ANY-SCP"   // Called AET (server's AE Title)
	dicomFile = "test.dcm"  // Path to the DICOM file to send
)

// SendDicomFile sends a DICOM file to a remote DICOM SCP using storescu (DCMTK)
func SendDicomFile(aet, remoteAet, remoteHost, dicomFile string, remotePort int) (string, error) {
	log.Printf("sending")
	// Read DICOM file with go-dicom (graymeta package)
	/*
		_, err := dicom.ParseFile(dicomFile, nil)
		if err != nil {
			return fmt.Errorf("error reading DICOM file: %v", err)
		}
	*/

	//storescu -aet "test" term2022 11125 -aec "remote" '/home/ulewu/Projects/Golang/GoDICOM/TestDaten/Braun Albert 220010273/DICOM/0000E0F0/AA42A9F6/AA477D28/0000D070/EE55BACD'

	// Use storescu command to send the file
	cmd := exec.Command("storescu", "-aet", aet, remoteHost, fmt.Sprintf("%d", remotePort), dicomFile)

	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("error executing storescu: %v, output: %s", err, output)
	}

	fmt.Printf("DICOM file sent successfully, output: %s\n", output)
	return fmt.Sprintf(string(output)), nil
}

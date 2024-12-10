package main

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"runtime"

	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
	//"github.com/gradienthealth/dicom"
)

func getDicomData_old(filename string) (string, string, string, error) {
	log.Println(filename)
	//fmt.Println("FileName: ", filename)
	// Open the DICOM file
	//fmt.Println("Processing file:", filename)

	// Manual GarbageCollector - without memory gets overflowed
	defer runtime.GC() // Run garbage collection

	// Open and parse the DICOM file
	dcm, err := dicom.ParseFile(filename, nil)

	if err != nil {
		//log.Printf("Error parsing DICOM file - no valid DICOM %s: %v\n", filename, err)
		return "", "", "", err
	}

	// Extract the PatientName tag
	PatientName := ""
	PatientID := ""
	InstitutionName := ""

	ePatientName, _ := dcm.FindElementByTag(tag.PatientName)
	ePatientID, _ := dcm.FindElementByTag(tag.PatientID)
	eInstitutionName, _ := dcm.FindElementByTag(tag.InstitutionName)
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

	if eInstitutionName != nil {
		//PatientID = ePatientID.Value.GetValue().(string)
		InstitutionName = fmt.Sprintf("%v", eInstitutionName.Value.GetValue())

	} else {
		//log.Printf("Patient Name tag not found in file %s\n", filename)
		InstitutionName = "tag.InstitutionName not found"
	}

	// return fmt.Sprintf("%v", ePatientName.Value.GetValue()), nil
	return PatientName, PatientID, InstitutionName, nil
}

func getDicomData(dicomFile string) (string, string, string, error) {
	// +P 0010,0010 - name
	// +P 0010,0020  - patientID
	// +P 0008,0080 - Institution
	cmdPatName := exec.Command("dcmdump.exe", "+P", "0010,0010", dicomFile)
	cmdPatId := exec.Command("dcmdump.exe", "+P", "0010,0020", dicomFile)
	cmdInstitute := exec.Command("dcmdump.exe", "+P", "0008,0080", dicomFile)

	outPatName, errPatName := cmdPatName.CombinedOutput()
	if errPatName != nil {
		log.Printf("error getting patName %v, output: %s", errPatName, outPatName)
	}

	outPatId, errPatId := cmdPatId.CombinedOutput()
	if errPatName != nil {
		log.Printf("error getting cmdPatId %v, output: %s", errPatId, outPatId)
	}

	outInstitute, errInstitute := cmdInstitute.CombinedOutput()
	if errInstitute != nil {
		log.Printf("error getting patName %v, output: %s", errInstitute, outInstitute)
	}

	//fmt.Printf("DICOM output: %s\n", output)
	//fmt.Printf("result: %s ", extractTextBetweenBrackets(string(output)))
	patName := extractTextBetweenBrackets(string(outPatName))
	patID := extractTextBetweenBrackets(string(outPatId))
	Institute := extractTextBetweenBrackets(string(outInstitute))
	// Print each match on a new line

	if errPatName != nil || errPatId != nil {
		log.Printf("Non valid DICOM file: %s", dicomFile)
		return "", "", "", errPatName
	} else if len(Institute) == 0 || errInstitute != nil {
		fmt.Printf("name: %s\n", patName[0])
		fmt.Printf("ID: %s\n", patID[0])
		return patName[0], patID[0], "no institution name", nil
	} else {
		fmt.Printf("name: %s\n", patName[0])
		fmt.Printf("ID: %s\n", patID[0])
		fmt.Printf("Inst: %s\n", Institute[0])
		return patName[0], patID[0], Institute[0], nil
	}

}

func extractTextBetweenBrackets(input string) []string {
	// Define the regex pattern to match text within square brackets, including the brackets.
	re := regexp.MustCompile(`\[[^\[\]]*\]`)

	// Find all matches in the input string
	matches := re.FindAllString(input, -1)

	return matches
}

// SendDicomFile sends a DICOM file to a remote DICOM SCP using storescu (DCMTK)
func SendDicomFile(aet string, remoteAet string, remoteHost string, remotePort string, dicomFile string) (string, error) {
	// Test deactive sending
	//return "", nil
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
	cmd := exec.Command("storescu", "--propose-lossless", "-aet", aet, remoteHost, remotePort, dicomFile)

	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("error executing storescu: %v, output: %s", err, output)
	}

	fmt.Printf("DICOM file sent successfully, output: %s\n", output)
	return fmt.Sprintf(string(output)), nil
}

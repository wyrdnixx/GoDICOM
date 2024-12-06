package main

import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func startFileRunner() {

	log.Printf("'startFileRunner' starting filerunner")
	fileRunnerRunning = true
	var wg sync.WaitGroup

	sem := make(chan struct{}, Config.MaxGoroutines) // Semaphore to limit to 50 concurrent Goroutines

	// Start a Goroutine to monitor the number of active Goroutines
	go func() {
		for {
			//fmt.Printf("%s : Currently running Goroutines: %d\n", time.Now().Format("02.01.2006 15:04:05"), atomic.LoadInt32(&activeGoroutines))
			log.Printf("'startFileRunner' Currently running Goroutines: %d", atomic.LoadInt32(&activeGoroutines))

			time.Sleep(2000 * time.Millisecond) // Adjust the interval for monitoring
		}
	}()

	err := walkDir(Config.RootDirectory, &wg, sem, &activeGoroutines)
	if err != nil {
		fmt.Printf("Error walking the directory %s : %v\n", Config.RootDirectory, err)
		return
	}

	// Wait for all Goroutines to finish
	wg.Wait()
	fmt.Println("\nAll files processed.")
	fileRunnerRunning = false
}

func walkDir(root string, wg *sync.WaitGroup, sem chan struct{}, activeGoroutines *int32) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			wg.Add(1)
			sem <- struct{}{}                    // Acquire a semaphore slot
			atomic.AddInt32(activeGoroutines, 1) // Increment the counter for a new Goroutine
			go fileProcessor(path, wg, sem, activeGoroutines)
		}
		return nil
	})
}

func fileProcessor(path string, wg *sync.WaitGroup, sem chan struct{}, activeGoroutines *int32) {
	defer wg.Done()
	defer atomic.AddInt32(activeGoroutines, -1) // Decrement the counter when done
	defer func() { <-sem }()                    // Release the semaphore slot when done

	exists, err := checkFileInDB(db, path) //First check if file already in DB

	if err != nil {
		log.Printf("processFile: error checking for file entry in DB: %s", err)
		return
	}

	if exists { // skip file that is already in database
		cFilesSkippedAlreadyProcessed++
	} else { // Process new file

		if strings.HasSuffix(path, ".tar") {
			processTarFile(path)
			return
		} else {
			processNonTarFile(path)
		}

	}
}

func processTarFile(path string) {
	log.Printf("File %s is a TAR file... extracting and processing sub-files", path)
	tarFileName := filepath.Base(path)
	tempFolder := Config.TempDir + "\\" + tarFileName
	extractedFiles, err := ExtractTarFile(path, tempFolder)
	if err != nil {
		log.Printf("Eror extracting tar file: %s : %s", path, err)
		InsertFilenameToDB(db, path, 0, "", "", "", "0", fmt.Sprintf("tar extraction error: %s", err)) //error on tar file
	} else {
		InsertFilenameToDB(db, path, 0, "", "", "", "0", "tar archive file") //error on tar file
		// run each extracted files
		for _, file := range extractedFiles {
			log.Println("processing Extracted file:", file)
			processNonTarFile(file) // not extra go routine to make shure all files processed an cleanup tempdir after
		}
	}
	log.Printf("delete temp folder: %s", tempFolder)
	// Use os.RemoveAll to delete the root directory and its contents
	err = os.RemoveAll(tempFolder)
	if err != nil {
		log.Println("Error deleting root directory:", err)
	} else {
		log.Println("Root directory deleted successfully")
	}
}

func processNonTarFile(path string) {
	PatientName, PatientID, InstitutionName, err := getDicomData(path)
	if err != nil {
		log.Printf("non dicom file: %s", path)
		InsertFilenameToDB(db, path, 0, PatientName, PatientID, "", "0", "0") //non DICOM file
		cFilesImportedNoDCMToDB++
	} else {
		log.Printf("valid dicom file: %s", path)
		processDicomFile(path, PatientName, PatientID, InstitutionName)
	}
}

func processDicomFile(path string, PatientName string, PatientID string, InstitutionName string) {

	if checkDicomInstitute(InstitutionName) {
		log.Printf("valid Institution: %s for file: %s", InstitutionName, path)
		// send dicom file
		res, err := SendDicomFile(Config.DicomServerLocalAET, Config.DicomServerRemoteAET, Config.DicomServer, Config.DicomServerPort, path)
		if err != nil {
			log.Printf("error sending dicom file: %s", err)
			err := InsertFilenameToDB(db, path, 1, PatientName, PatientID, InstitutionName, "0", fmt.Sprintf("%s", err)) // Valid DICOM file
			if err != nil {
				log.Printf("DB insert error: %s", err)
			}

		} else {
			err := InsertFilenameToDB(db, path, 1, PatientName, PatientID, InstitutionName, "1", res) // Valid DICOM file was send to dicom store
			if err != nil {
				log.Printf("DB insert error: %s", err)
			}
		}
	} else {
		log.Printf("non valid Institution: %s for file: %s", InstitutionName, path)
		err := InsertFilenameToDB(db, path, 1, PatientName, PatientID, InstitutionName, "0", "invalid institutionName") // Valid DICOM file was send to dicom store
		if err != nil {
			log.Printf("DB insert error: %s", err)
		}
	}
}

func checkDicomInstitute(InstitutionName string) bool {
	splitFilters := strings.Split(Config.DicomInstituteFilter, "|")

	// Flag to check if any split part contains the entire DicomInstituteFilter
	validInstitute := false

	// Iterate over the split string and check if any part contains the original filter
	for _, filter := range splitFilters {
		if strings.Contains(InstitutionName, filter) {
			validInstitute = true
			break
		}
	}
	return validInstitute
}

// ExtractTarFile extracts a tar file and returns the path of the extracted files
func ExtractTarFile(tarFilePath, destDir string) ([]string, error) {
	// Check if the file has a .tar extension
	if !strings.HasSuffix(tarFilePath, ".tar") {
		return nil, fmt.Errorf("the file does not have a .tar extension")
	}

	// Open the tar file
	tarFile, err := os.Open(tarFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open tar file: %w", err)
	}
	defer tarFile.Close()

	// Create a new tar reader
	tarReader := tar.NewReader(tarFile)

	var extractedFiles []string

	// Extract each file from the tar header *archive/tar.Header nil
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			// End of tar file
			return extractedFiles, nil
			//break
		}
		if err != nil {
			return extractedFiles, nil
		}
		// Invalid tar header, we just skip the current entry and move to the next
		/*if err != nil {
			if strings.Contains(err.Error(), "invalid tar header") {
				// Skip this entry and continue reading the next entry
				fmt.Printf("Warning: Invalid tar header, skipping entry...\n")
				// Skip the invalid entry by calling Next again
				break
			}
			return nil, fmt.Errorf("failed to read tar entry: %w", err)
		}*/

		// Determine the full path for the extracted file, including subdirectories
		extractedFilePath := filepath.Join(destDir, header.Name)

		// Handle directory entries
		if header.Typeflag == tar.TypeDir {
			// Ensure the directory exists
			if err := os.MkdirAll(extractedFilePath, os.ModePerm); err != nil {
				return nil, fmt.Errorf("failed to create directory: %w", err)
			}
		} else {
			// Ensure the parent directory exists for files
			dir := filepath.Dir(extractedFilePath)
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				return nil, fmt.Errorf("failed to create directory for file: %w", err)
			}

			// Create the file
			extractedFile, err := os.Create(extractedFilePath)
			if err != nil {
				return nil, fmt.Errorf("failed to create file: %w", err)
			}

			// Copy the file contents from the tar
			if _, err := io.Copy(extractedFile, tarReader); err != nil {
				return nil, fmt.Errorf("failed to copy file data: %w", err)
			}
			extractedFile.Close()
		}

		// Add the extracted file path to the list
		log.Printf("extracted: %s ", extractedFilePath)
		extractedFiles = append(extractedFiles, extractedFilePath)
	}

	return extractedFiles, nil
}

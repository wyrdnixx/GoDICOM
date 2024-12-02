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

// processFile is the function that will be run in a new Goroutine for each file
func processFile(path string, wg *sync.WaitGroup, sem chan struct{}, activeGoroutines *int32) {
	defer wg.Done()
	defer atomic.AddInt32(activeGoroutines, -1) // Decrement the counter when done
	defer func() { <-sem }()                    // Release the semaphore slot when done

	exists, err := checkFileInDB(db, path) //First check if file already in DB

	if err != nil {
		log.Printf("processFile: error checking for file entry in DB: %s", err)
		return
	}

	if !exists { // if file not already present in DB

		PatientName, PatientID, InstitutionName, err := getDicomData(path)
		if err != nil {
			InsertFilenameToDB(db, path, 0, PatientName, PatientID, "", "0", "0") //non DICOM file
			cFilesImportedNoDCMToDB++
		} else {

			// check for valid institution

			// Split the string by "|"
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

			if !validInstitute {
				err := InsertFilenameToDB(db, path, 1, PatientName, PatientID, InstitutionName, "0", "Not valid Institute") // Valid DICOM file
				if err != nil {
					log.Printf("DB insert error: %s", err)
				}
			} else {

				// send dicom file
				res, err := SendDicomFile(Config.DicomServerLocalAET, Config.DicomServerRemoteAET, Config.DicomServer, Config.DicomServerPort, path)
				if err != nil {
					log.Printf("error sending dicom file: %s", err)
					err := InsertFilenameToDB(db, path, 1, PatientName, PatientID, InstitutionName, "0", fmt.Sprintf("%s", err)) // Valid DICOM file
					log.Printf("DB insert error: %s", err)
				} else {
					log.Printf("result: %s", res)
					err := InsertFilenameToDB(db, path, 1, PatientName, PatientID, InstitutionName, "1", res) // Valid DICOM file
					log.Printf("DB insert error: %s", err)
				}
			}

			cFilesImportedDCMToDB++

		}

	} else {
		cFilesSkippedAlreadyProcessed++
	}
	//// Simulate file processing with sleep (replace with actual file processing)
	//time.Sleep(20 * time.Second)
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
			go processFile(path, wg, sem, activeGoroutines)
		}
		return nil
	})
}

func startFileRunner() {

	log.Printf("'startFileRunner' starting filerunner")
	fileRunnerRunning = true
	var wg sync.WaitGroup

	//root := "/home/ulewu/Projects/Golang/GoDICOM/TestDaten" // Replace with your directory path

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

// ExtractTarFile extracts a tar file and returns the path of the extracted files
func ExtractTarFile(tarFilePath, destDir string) ([]string, error) {
	// Open the tar file
	tarFile, err := os.Open(tarFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open tar file: %w", err)
	}
	defer tarFile.Close()

	// Create a new tar reader
	tarReader := tar.NewReader(tarFile)

	var extractedFiles []string

	// Extract each file from the tar
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			// End of tar file
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Determine the destination file path
		extractedFilePath := filepath.Join(destDir, header.Name)

		// Create directories if necessary
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(extractedFilePath, os.ModePerm); err != nil {
				return nil, fmt.Errorf("failed to create directory: %w", err)
			}
		} else {
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
		extractedFiles = append(extractedFiles, extractedFilePath)
	}

	return extractedFiles, nil
}

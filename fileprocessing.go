package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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

		PatientName, PatientID, err := getDicomData(path)
		if err != nil {
			InsertFilenameToDB(db, path, 0, PatientName, PatientID, "", "0", "0") //non DICOM file
			cFilesImportedNoDCMToDB++
		} else {
			//log.Printf(patname)

			// send dicom file
			res, err := SendDicomFile(Config.DicomServerLocalAET, Config.DicomServerRemoteAET, Config.DicomServer, Config.DicomServerPort, path)
			if err != nil {
				log.Printf("error sending dicom file: %s", err)
				err := InsertFilenameToDB(db, path, 1, PatientName, PatientID, "institute", "0", fmt.Sprintf("%s", err)) // Valid DICOM file
				log.Printf("DB insert error: %s", err)
			} else {
				log.Printf("result: %s", res)
				err := InsertFilenameToDB(db, path, 1, PatientName, PatientID, "institute", "1", res) // Valid DICOM file
				log.Printf("DB insert error: %s", err)
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

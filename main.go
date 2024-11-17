package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Config struct to hold the configuration values
type Configguration struct {
	MaxGoroutines int    `json:"max_goroutines"`
	RootDirectory string `json:"root_directory"`
	ConnString    string `json:"connString"`
}

var Config Configguration
var db *sql.DB
var activeGoroutines int32 // To track the number of active Goroutines
var startTime time.Time
var cFilesSkippedAlreadyProcessed int32
var cFilesImportedDCMToDB int32
var cFilesImportedNoDCMToDB int32

// loadConfig loads the configuration from a JSON file
func loadConfig(configFile string) (Configguration, error) {

	file, err := os.ReadFile(configFile)
	if err != nil {
		return Config, err
	}
	err = json.Unmarshal(file, &Config)
	return Config, err
}

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
			InsertFilenameToDB(db, path, 0, PatientName, PatientID) //non DICOM file
			cFilesImportedNoDCMToDB++
		} else {
			//log.Printf(patname)
			InsertFilenameToDB(db, path, 1, PatientName, PatientID) // Valid DICOM file
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
		fmt.Printf("Error walking the directory: %v\n", err)
		return
	}

	// Wait for all Goroutines to finish
	wg.Wait()
	fmt.Println("\nAll files processed.")
}

func exitGracefully() {

	log.Printf("waiting for running routines to finish...")
	// Todo: log the programm running times to database

	// no sense to wait - all files will be finished
	/* for atomic.LoadInt32(&activeGoroutines) > 0 {
		log.Printf("waiting for %d tasks... ", atomic.LoadInt32(&activeGoroutines))
		time.Sleep(2000 * time.Millisecond) // Adjust the interval for monitoring
	} */

	exitProgramm()
}

func exitProgramm() {

	db.Close()

	// Get the current time at the end
	end := time.Now()
	// Calculate the duration
	duration := end.Sub(startTime)
	// Print the duration
	log.Printf("Program execution time: %v\n", duration)
	log.Printf("Processed: %d files Skipped already present, %d non DICOM File imported, %d DICOM Files imported",
		cFilesSkippedAlreadyProcessed, cFilesImportedNoDCMToDB, cFilesImportedDCMToDB)
	os.Exit(0)
}

func main() {

	startTime = time.Now()

	//// handle strg+c signal
	// Create a channel to receive OS signals.
	sigChan := make(chan os.Signal, 1)

	// Notify the channel when an interrupt signal (SIGINT) or termination signal (SIGTERM) is received.
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	// Wait for the signal in a separate goroutine
	go func() {
		<-sigChan
		log.Println("\n'main' Signal received, initiating graceful shutdown...")
		exitGracefully() // Cancel the context, signaling goroutines to stop.
	}()

	// Load the configuration from the config.json file
	Config, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("'main' Error loading configuration: %s ", err)
		return // Exit Programm
	}

	log.Printf("'main' config loaded - Scann directory: %s", Config.ConnString)
	// Connect to the database
	db, err = sql.Open("sqlserver", Config.ConnString)
	if err != nil {
		//log.Fatalf("Error creating connection pool: %s", err.Error())
		log.Fatalf("'main' Error creating connection pool %s", err)
		return // Exit Programm
	}
	defer db.Close()

	log.Printf("'main' Connected to the database!")
	//log.Fatalf("fatal %d ", 32)
	initDB(db)

	go startWebservice() // Start the webservice in an extra go routine

	startFileRunner()

	exitProgramm()
}

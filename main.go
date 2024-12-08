package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Config struct to hold the configuration values
type Configguration struct {
	MaxGoroutines        int    `json:"MaxGoroutines"`
	RootDirectory        string `json:"RootDirectory"`
	ConnString           string `json:"ConnString"`
	DicomServer          string `json:"DicomServer"`
	DicomServerPort      string `json:"DicomServerPort"`
	DicomServerLocalAET  string `json:"DicomServerLocalAET"`
	DicomServerRemoteAET string `json:"DicomServerRemoteAET"`
	DicomInstituteFilter string `json:"DicomInstituteFilter"`
	TempDir              string `json:"TempDir"`
}

var Config Configguration
var db *sql.DB
var activeGoroutines int32 // To track the number of active Goroutines
var startTime time.Time
var filerunnerFinishedTime time.Time
var cFilesSkippedAlreadyProcessed int32
var cFilesImportedDCMToDB int32
var cFilesSkippedWrongInstitute int32
var cFilesImportedNoDCMToDB int32
var cFilesTarProcessed int32
var fileRunnerRunning bool

// loadConfig loads the configuration from a JSON file
func loadConfig(configFile string) (Configguration, error) {

	file, err := os.ReadFile(configFile)
	if err != nil {
		return Config, err
	}
	err = json.Unmarshal(file, &Config)
	return Config, err
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

	// Debug Memory using ' go tool pprof http://localhost:6060/debug/pprof/heap
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

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

	log.Printf("'main' config loaded - Scann directory: %s", Config.RootDirectory)
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

	go startFileRunner()
	startWebservice() // Start the webservice

	//exitProgramm()
}

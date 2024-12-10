package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	//vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	//	fmt.Fprintf(w, "Main Page\n", vars["category"])

	runningTime := time.Now()
	// Calculate the difference between the two dates
	diff := runningTime.Sub(startTime)

	// Convert the difference to days, hours, minutes, and seconds
	days := int(diff.Hours()) / 24
	hours := int(diff.Hours()) % 24
	minutes := int(diff.Minutes()) % 60
	seconds := int(diff.Seconds()) % 60

	// Print the difference in various formats

	fmt.Fprintf(w, "Main Page\n")
	fmt.Fprintf(w, "Server start time: %s\n", startTime)
	fmt.Fprintf(w, "Processing folder: %s\n", Config.RootDirectory)
	fmt.Fprintf(w, "Programm running since: days, hours, minutes, seconds : %d D, %d H, %d M, %d S\n", days, hours, minutes, seconds)
	fmt.Fprintf(w, "Filerunner running status: %t\n", fileRunnerRunning)

	if !fileRunnerRunning {
		// Calculate the difference between the two dates
		frdiff := filerunnerFinishedTime.Sub(startTime)
		// Convert the difference to days, hours, minutes, and seconds
		frdays := int(frdiff.Hours()) / 24
		frhours := int(frdiff.Hours()) % 24
		frminutes := int(frdiff.Minutes()) % 60
		frseconds := int(frdiff.Seconds()) % 60
		fmt.Fprintf(w, "Filerunner finished at: %s\n", filerunnerFinishedTime)
		//fmt.Fprintf(w, "Filerunner was running for (days, hours, minutes, seconds) : %v D, %v H, %v M, %v S", diff.Hours()/24, diff.Hours(), diff.Minutes(), diff.Seconds())
		fmt.Fprintf(w, "Filerunner was running for (days, hours, minutes, seconds) : %d D, %d H, %d M, %d S\n", frdays, frhours, frminutes, frseconds)
	}

	fmt.Fprintf(w, "activeGoroutines: %d\n", activeGoroutines)
	fmt.Fprintf(w, "cFilesSkippedAlreadyProcessed: %d\n", cFilesSkippedAlreadyProcessed)
	fmt.Fprintf(w, "cFilesTarProcessed: %d\n", cFilesTarProcessed)
	fmt.Fprintf(w, "cFilesImportedDCMToDB: %d\n", cFilesImportedDCMToDB)
	fmt.Fprintf(w, "cFilesSkippedWrongInstitute: %d\n", cFilesSkippedWrongInstitute)
	fmt.Fprintf(w, "cFilesImportedNoDCMToDB: %d\n", cFilesImportedNoDCMToDB)

}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Hello Status")
}

func startWebservice() {

	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/status", StatusHandler)

	http.Handle("/", r)

	srv := &http.Server{
		Handler: r,
		Addr:    "127.0.0.1:8000",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("webservice: Stating webservice at: %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())

}

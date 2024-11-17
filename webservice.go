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
	fmt.Fprintf(w, "Main Page\n")
	fmt.Fprintf(w, "Server start time: %s\n", startTime)
	fmt.Fprintf(w, "activeGoroutines: %d\n", activeGoroutines)
	fmt.Fprintf(w, "cFilesSkippedAlreadyProcessed: %d\n", cFilesSkippedAlreadyProcessed)
	fmt.Fprintf(w, "cFilesImportedDCMToDB: %d\n", cFilesImportedDCMToDB)
	fmt.Fprintf(w, "cFilesImportedNoDCMToDB: %d\n", cFilesImportedNoDCMToDB)

}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Hello Status")
}

func startWebservice() {
	log.Printf("webservice: Stating webservice")
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

	log.Fatal(srv.ListenAndServe())

}

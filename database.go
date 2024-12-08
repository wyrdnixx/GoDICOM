package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	_ "github.com/denisenkom/go-mssqldb" // MSSQL driver import
)

func initDB(db *sql.DB) error {

	log.Printf("'initdb' Initialising the database!")
	createTableSQL := `IF NOT EXISTS (
    	SELECT * 
    	FROM sys.tables 
    	WHERE name = 'importfiles'
	)
	BEGIN
    CREATE TABLE importfiles (
        id INT IDENTITY(1,1) PRIMARY KEY, -- Auto-increment primary key
        filename VARCHAR(255) NOT NULL,       -- File name or path 
		archiveFile VARCHAR(255) NOT NULL,       -- File name or path 
        isDICOM INT ,             -- 1 for DICOM, 0 for non-DICOM
		PatientName VARCHAR(255), 		-- patients name from dicom fields
		PatientID VARCHAR(255), 		-- patients ID from dicom fields
		Institute VARCHAR(255), 		-- institute from dicom fields
        StoreStatus VARCHAR(10),			-- file was successfuly stored
		StoreMessage VARCHAR(512)			-- file was successfuly stored
    );
	END;`

	// Execute the query
	_, err := db.Exec(createTableSQL)
	if err != nil {
		//log.Fatalf("error creating table: %v ", err)
		log.Fatalf("'initdb' error creating table: "+err.Error(), 3)
		return err
	}

	log.Printf("'initdb' Table 'importfiles' is ready.")
	return nil

}

/* Check for existing file entry in database */
func checkFileInDB(db *sql.DB, filename string) (bool, error) {

	// Check if the filename exists
	var exists bool
	query := `SELECT COUNT(*) FROM importfiles WHERE filename ='` + filename + `';`
	err := db.QueryRow(query, filename).Scan(&exists)
	if err != nil {
		log.Fatalf("'checkFileInDB' Error checking filename existence:%s Query: %s Error: %v", filename, query, err)
		return false, err
	}

	if !exists {
		return false, nil
	} else {
		return true, nil
	}

}

// InsertFilenameToDB checks if the filename exists, and inserts it if it does not
func InsertFilenameToDB(db *sql.DB, filename string, archiveFile string, isDICOM int, PatientName string, PatientID string, Institute string, storeStatus string, storeMessage string) error {

	// Create the SQL query with placeholders
	query := fmt.Sprintf("INSERT INTO importfiles (filename, archiveFile, isDICOM, PatientName, PatientID, Institute, storeStatus, StoreMessage) VALUES ('%s','%s','%s','%s','%s','%s','%s','%s')",
		filename,
		archiveFile,
		strconv.Itoa(isDICOM),
		PatientName,
		PatientID,
		Institute,
		storeStatus,
		strings.Replace(storeMessage, "'", "''", -1)) // replace ' by '' -> for SQL Insert text containing ' must be escaped

	// Prepare the statement to avoid SQL injection
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Printf("error query prepare")
		return err
	}
	defer stmt.Close()

	// Execute the query with the provided values
	_, err = stmt.Exec()
	if err != nil {
		log.Printf("error query execute: %s", query)
		return err
	}

	/*
		_, err := db.Exec(insertQuery, filename)
		if err != nil {
			log.Fatalf("'InsertFilenameIfNotExists' Error inserting filename: %v", err)
			return err
		}
		log.Printf("'InsertFilenameIfNotExists' Filename '%s' inserted successfully!\n", filename)

	*/
	return nil
}

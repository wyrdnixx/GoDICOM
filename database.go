package main

import (
	"database/sql"
	"log"
	"strconv"

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
        filename VARCHAR(255) NOT NULL,       -- File name or path (adjust size as needed)
        isDICOM INT ,             -- 1 for DICOM, 0 for non-DICOM
		PatientName VARCHAR(255), 		-- patients name from dicom fields
		PatientID VARCHAR(255), 		-- patients name from dicom fields
        imported DATETIME         -- Timestamp when the file was imported
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
		log.Fatalf("'InsertFilenameIfNotExists' Error checking filename existence: %v", err)
		return false, err
	}

	if !exists {
		return false, nil
	} else {
		return true, nil
	}

}

// InsertFilenameToDB checks if the filename exists, and inserts it if it does not
func InsertFilenameToDB(db *sql.DB, filename string, isDICOM int, PatientName string, PatientID string) error {
	insertQuery := `INSERT INTO importfiles (filename, isDICOM, PatientName, PatientID) VALUES ('` + filename + `' ,` + strconv.Itoa(isDICOM) + `,'` + PatientName + `' , '` + PatientID + `');`
	_, err := db.Exec(insertQuery, filename)
	if err != nil {
		log.Fatalf("'InsertFilenameIfNotExists' Error inserting filename: %v", err)
		return err
	}
	log.Printf("'InsertFilenameIfNotExists' Filename '%s' inserted successfully!\n", filename)
	return nil
}

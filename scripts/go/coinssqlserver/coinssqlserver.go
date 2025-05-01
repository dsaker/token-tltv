package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

// Token represents the structure of each token object in the JSON file.
type Token struct {
	UploadUsed bool      `json:"UploadUsed"`
	TimesUsed  int       `json:"TimesUsed"`
	Created    time.Time `json:"Created"`
	Hash       string    `json:"Hash"`
}

func main() {
	// Update the connection string to match your environment
	// Possible format for SQL Server:
	// "server=localhost;database=YourDatabase;user id=YourUsername;password=YourPassword;encrypt=disable"
	connString := os.Getenv("MY_CONNECTION_STRING")

	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		log.Fatalf("Error opening DB connection: %v", err)
	}
	defer db.Close()

	// Verify if the connection is valid
	if err = db.Ping(); err != nil {
		log.Fatalf("Error pinging DB: %v", err)
	}

	filePath := os.Getenv("FILE_PATH")
	// Read the JSON file into memory.
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	// Parse the JSON data into a slice of Token structs.
	var tokens []Token
	if err := json.Unmarshal(fileBytes, &tokens); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	// Insert tokens into the table
	for _, token := range tokens {
		err = insertToken(db, token)
		if err != nil {
			log.Printf("Failed to insert token %s: %v\n", token.Hash, err)
		} else {
			fmt.Printf("Successfully inserted token: %s\n", token.Hash)
		}
	}
}

// insertToken inserts a single token into the tokens table
func insertToken(db *sql.DB, token Token) error {
	query := "INSERT INTO Tokens (Hash, Created, Used) VALUES (@Hash, @Created, @Used)"
	_, err := db.Exec(query,
		sql.Named("Hash", token.Hash),
		sql.Named("Created", token.Created),
		sql.Named("Used", token.TimesUsed))
	return err
}

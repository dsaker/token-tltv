/*
TOKEN GENERATOR
--------------
This tool generates secure tokens and stores them in a Firestore database.

Usage:

	token-generator -p PROJECT_ID -n NUM_TOKENS [-c COLLECTION]

Required Parameters:

	-p string    Google Cloud project ID where the tokens will be stored in Firestore
	-n int       Number of tokens to generate (must be greater than 0)

Optional Parameters:

	-c string    Firestore collection name where tokens will be stored (default: "tokens")

Examples:

	# Generate 5 tokens in the default collection
	token-generator -p my-project-id -n 5

	# Generate 10 tokens in a custom collection
	token-generator -p my-project-id -n 10 -c custom-tokens

Notes:
  - This tool requires proper Google Cloud authentication
  - Set GOOGLE_APPLICATION_CREDENTIALS environment variable to your service account JSON file
  - Each token consists of a random plaintext value (displayed after generation)
    and a secure hash that's stored in Firestore
  - Only the hash is stored in Firestore for security reasons
  - The plaintext values are displayed once after generation and should be stored securely
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"talkliketv.com/tltv/internal/services/tokens"
)

// go run . -p=token-tltv-test -n=1
func main() {
	var projectId string
	var collection string
	var numTokens int

	flag.StringVar(&projectId, "p", "", "project_id where the tokens will be stored")
	flag.StringVar(&collection, "c", "tokens", "collection where the tokens will be stored")
	flag.IntVar(&numTokens, "n", 0, "the number of tokens to generate")
	flag.Parse()

	// Display help message if required parameters are missing
	if projectId == "" || numTokens <= 0 {
		printUsage()
		return
	}

	plaintexts, err := tokens.CreateTokens(numTokens, collection, projectId)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Successfully generated and stored", numTokens, "tokens.")
	fmt.Println("Token plaintext values (keep these secure):")
	for i, plaintext := range plaintexts {
		fmt.Printf("  %d: %s\n", i+1, plaintext)
	}
}

func printUsage() {
	fmt.Println(`TOKEN GENERATOR
--------------
This tool generates secure tokens and stores them in a Firestore database.

Usage:
  token-generator -p PROJECT_ID -n NUM_TOKENS [-c COLLECTION]

Required Parameters:
  -p string    Google Cloud project ID where the tokens will be stored in Firestore
  -n int       Number of tokens to generate (must be greater than 0)

Optional Parameters:
  -c string    Firestore collection name where tokens will be stored (default: "tokens")

Examples:
  # Generate 5 tokens in the default collection
  token-generator -p my-project-id -n 5

  # Generate 10 tokens in a custom collection
  token-generator -p my-project-id -n 10 -c custom-tokens

Notes:
  - This tool requires proper Google Cloud authentication
  - Set GOOGLE_APPLICATION_CREDENTIALS environment variable to your service account JSON file
  - Each token consists of a random plaintext value (displayed after generation)
    and a secure hash that's stored in Firestore
  - Only the hash is stored in Firestore for security reasons
  - The plaintext values are displayed once after generation and should be stored securely`)
	os.Exit(1)
}

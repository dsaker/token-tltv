/*
coinsfirestore uploads a jsonfile of tokens to the default firestore database

Usage:

	go run coinsfirestore.go [flags]

The flags are:

	-f
	    The file path to the json file to upload
	-p
	    The project id to upload it to
	-c
		The collection to upload it to

	coinsfirestore % go run . -f ../../../internal/models/jsonmodels/tokens-20250131102018.json -p token-tltv -c token-tltv-test
*/
package main

import (
	"context"
	"encoding/json"
	firebase "firebase.google.com/go"
	"flag"
	"fmt"
	"log"
	"os"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/test"
)

func main() {
	filePath := flag.String("f", "", "filepath is required")
	projectID := flag.String("p", test.TestProject, "project is required")
	collection := flag.String("c", test.FirestoreTestCollection, "collection is required. ")
	flag.Parse()

	if *filePath == "" {
		fmt.Println("Error: -f filepath flag is required")
		os.Exit(1)
	}

	if *projectID == "" {
		fmt.Println("Error: -p project id flag is required")
		os.Exit(1)
	}

	if *collection == "" {
		fmt.Println("Error: -c collection flag is required")
		os.Exit(1)
	}

	// Use the application default credentials
	ctx := context.Background()
	conf := &firebase.Config{ProjectID: *projectID}
	app, err := firebase.NewApp(ctx, conf)
	if err != nil {
		log.Fatalln(err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	defer client.Close()

	// Open the JSON file
	file, err := os.Open(*filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Decode the JSON data
	var tokens []models.Token
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&tokens)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	// get the tokens collection from the database
	tokensColl := client.Collection(*collection)
	for _, token := range tokens {
		_, err = tokensColl.Doc(token.Hash).Set(ctx, models.FirestoreToken{
			UploadUsed: token.UploadUsed,
			TimesUsed:  token.TimesUsed,
			Created:    token.Created,
		})
		//_, _, err = tokensColl.Add(ctx, token)
		if err != nil {
			log.Fatalf("Failed adding token: %v", err)
		}
	}
}

package models

import (
	"cloud.google.com/go/firestore"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"talkliketv.click/tltv/internal/util"
	"time"
)

type Token struct {
	UploadUsed bool
	TimesUsed  int
	Created    time.Time
	Hash       string
}

type FirestoreToken struct {
	UploadUsed bool
	TimesUsed  int
	Created    time.Time
}

type Tokens struct {
	Coll *firestore.CollectionRef
}

type TokensX interface {
	CheckToken(c context.Context, token string) error
	UpdateField(c context.Context, value any, token string, path string) error
}

var (
	UsedTokenError = errors.New("token already used")
)

func (t *Tokens) CheckToken(ctx context.Context, token string) error {
	tokenHash := sha256.Sum256([]byte(token))
	hashString := hex.EncodeToString(tokenHash[:])
	tokenDoc, err := t.Coll.Doc(hashString).Get(ctx)
	if err != nil {
		return fmt.Errorf("get token check failed: %w", err)
	}
	data := tokenDoc.Data()
	for d := range data {
		log.Printf("type: %v, value: %v", d)
	}
	log.Printf("token: %v", tokenDoc.Data())
	var tStruct Token
	err = tokenDoc.DataTo(&tStruct)
	if err != nil {
		return fmt.Errorf("token data to struct failed: %w", err)
	}
	if tStruct.UploadUsed {
		return UsedTokenError
	}
	return err
}

func (t *Tokens) AddToken(ctx context.Context, token Token) error {
	_, err := t.Coll.Doc(token.Hash).Set(ctx, FirestoreToken{
		UploadUsed: token.UploadUsed,
		TimesUsed:  token.TimesUsed,
		Created:    token.Created,
	})
	if err != nil {
		return fmt.Errorf("failed adding token: %v", err)
	}
	return err
}

func (t *Tokens) UpdateField(ctx context.Context, value any, token, path string) error {
	tokenHash := sha256.Sum256([]byte(token))
	hashString := hex.EncodeToString(tokenHash[:])
	tokenDoc := t.Coll.Doc(hashString)
	_, err := tokenDoc.Update(ctx, []firestore.Update{
		{
			Path:  path,
			Value: value,
		},
	})
	if err != nil {
		return fmt.Errorf("token update failed: %w", err)
	}
	return err
}

func CreateTokensFile(filePath string, filename string, numTokens int) ([]string, error) {
	var tokens []*Token
	var plaintexts []string
	for i := 0; i < numTokens; i++ {
		token, plaintext, err := GenerateToken()
		if err != nil {
			log.Fatal(err)
		}
		tokens = append(tokens, token)
		plaintexts = append(plaintexts, plaintext)
	}

	// Marshal the data to JSON
	jsonData, err := json.Marshal(tokens)
	if err != nil {
		log.Fatal(err)
	}

	// create a file path if it does not exist
	exists, err := util.PathExists(filePath)
	if err != nil {
		log.Fatal(err)
	}
	if !exists {
		err = os.MkdirAll(filePath, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}
	// Open the file for writing
	file, err := os.Create(filePath + filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Write the JSON data to the file
	_, err = file.Write(jsonData)
	if err != nil {
		log.Fatal(err)
	}
	return plaintexts, nil
}

func GenerateToken() (*Token, string, error) {
	// Initialize a zero-valued byte slice with a length of 16 bytes.
	randomBytes := make([]byte, 16)

	// Use the Read() function from the crypto/rand package to fill the byte slice with
	// random bytes from your operating system's CSPRNG. This will return an error if
	// the CSPRNG fails to function correctly.
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, "", err
	}

	token := &Token{
		Created: time.Now(),
	}
	plaintext := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	// Generate a SHA-256 hash of the plaintext token string. This will be the value
	// that we store in the `hash` field of our database table.
	// Create the hash
	hash := sha256.Sum256([]byte(plaintext))

	// Convert the hash to a hexadecimal string
	hashString := hex.EncodeToString(hash[:])
	token.Hash = hashString

	return token, plaintext, nil
}

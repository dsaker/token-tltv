package tokens

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"talkliketv.com/tltv/internal/interfaces"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
)

func CreateTokens(numTokens int, collection, project_id string) ([]string, error) {
	tokens := make([]*interfaces.Token, 0, numTokens)
	plaintexts := make([]string, 0, numTokens)

	for i := 0; i < numTokens; i++ {
		token, plaintext, err := GenerateToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate token: %w", err)
		}
		tokens = append(tokens, token)
		plaintexts = append(plaintexts, plaintext)
	}

	// Initialize Firestore client
	ctx := context.Background()
	firestoreClient, err := initializeFirestore(ctx, project_id)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firestore: %w", err)
	}
	defer firestoreClient.Close()

	// Store tokens in Firestore
	err = storeTokensInFirestore(ctx, firestoreClient, tokens, collection)
	if err != nil {
		return nil, fmt.Errorf("failed to store tokens in Firestore: %w", err)
	}

	return plaintexts, nil
}

func GenerateToken() (*interfaces.Token, string, error) {
	// Initialize a zero-valued byte slice with a length of 16 bytes.
	randomBytes := make([]byte, 16)

	// Use the Read() function from the crypto/rand package to fill the byte slice with
	// random bytes from your operating system's CSPRNG. This will return an error if
	// the CSPRNG fails to function correctly.
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, "", err
	}

	token := &interfaces.Token{
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

func initializeFirestore(ctx context.Context, projectID string) (*firestore.Client, error) {
	// Use the application default credentials
	conf := &firebase.Config{ProjectID: projectID}
	app, err := firebase.NewApp(ctx, conf)
	if err != nil {
		return nil, fmt.Errorf("failed to create firebase app: %w", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create firestore client: %w", err)
	}

	return client, nil
}

func storeTokensInFirestore(ctx context.Context, client *firestore.Client, tokens []*interfaces.Token, collectionName string) error {
	bulkWriter := client.BulkWriter(ctx)

	for _, token := range tokens {
		firestoreToken := interfaces.FirestoreToken{
			UploadUsed: token.UploadUsed,
			TimesUsed:  token.TimesUsed,
			Created:    token.Created,
		}

		// Use the token hash as the document ID
		_, err := client.Collection("tokens").Doc(token.Hash).Set(ctx, firestoreToken)
		if err != nil {
			return fmt.Errorf("failed to store token in Firestore: %w", err)
		}
	}

	// Wait for all operations to complete and check for errors
	bulkWriter.End()
	return nil
}

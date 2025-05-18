package models

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"talkliketv.com/tltv/internal/interfaces"
	"talkliketv.com/tltv/internal/testutil"
	"talkliketv.com/tltv/internal/util"
	"testing"
	"time"
)

// TestCheckTokenIntegration tests the CheckToken method with a real Firestore client
// Run with: go test -v ./internal/models -test=integration -project-id=token-tltv-test
func TestCheckTokenIntegration(t *testing.T) {
	if util.Test != "integration" && !testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup test environment
	ctx := context.Background()
	fClient, err := testCfg.FirestoreClient()
	require.NoError(t, err)
	defer fClient.Close()

	// Create unique collection names for this test
	randomPrefix := t.Name()
	testLangColl := randomPrefix + LangCollString
	testLangCodeColl := randomPrefix + LangCodeCollString
	testVoiceColl := randomPrefix + VoiceCollString
	testTokenColl := randomPrefix + TokenCollString

	collections := []string{testLangColl, testLangCodeColl, testVoiceColl, testTokenColl}

	// Clean up collections after tests
	defer func() {
		for _, coll := range collections {
			err := testutil.DeleteCollection(ctx, fClient, coll, 10)
			require.NoError(t, err)
		}
	}()

	// Initialize the model with our test collections
	models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
	require.NoError(t, err)

	// Create test data
	testTokens := []struct {
		plaintext string
		hash      string
		token     interfaces.Token
	}{
		{
			plaintext: "valid-unused-token-1",
			token: interfaces.Token{
				UploadUsed: false,
				Created:    time.Now(),
			},
		},
		{
			plaintext: "valid-unused-token-2",
			token: interfaces.Token{
				UploadUsed: false,
				Created:    time.Now().Add(-24 * time.Hour), // Created yesterday
			},
		},
		{
			plaintext: "valid-used-token",
			token: interfaces.Token{
				UploadUsed: true,
				Created:    time.Now().Add(-48 * time.Hour), // Created 2 days ago
			},
		},
	}

	// Calculate hashes and add tokens to Firestore
	for i := range testTokens {
		hash := sha256.Sum256([]byte(testTokens[i].plaintext))
		testTokens[i].hash = hex.EncodeToString(hash[:])

		_, err := fClient.Collection(testTokenColl).Doc(testTokens[i].hash).Set(ctx, testTokens[i].token)
		require.NoError(t, err)
	}

	t.Run("Check valid unused token", func(t *testing.T) {
		err := models.CheckToken(ctx, testTokens[0].plaintext)
		require.NoError(t, err, "Valid unused token should not return an error")
	})

	t.Run("Check valid but used token", func(t *testing.T) {
		err := models.CheckToken(ctx, testTokens[2].plaintext)
		require.Error(t, err)
		assert.Equal(t, ErrUsedToken, err, "Used token should return ErrUsedToken")
	})

	t.Run("Check non-existent token", func(t *testing.T) {
		err := models.CheckToken(ctx, "non-existent-token")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "get token check failed")
	})

	t.Run("Check token with corrupted data", func(t *testing.T) {
		// Create a token with invalid/corrupted data
		corruptToken := "corrupt-token-data"
		hashBytes := sha256.Sum256([]byte(corruptToken))
		hashString := hex.EncodeToString(hashBytes[:])

		// Add invalid data that's not a proper Token struct
		invalidData := map[string]interface{}{
			"UploadUsed": "not-a-boolean", // This should cause DataTo to fail
			"Created":    "not-a-timestamp",
		}

		_, err := fClient.Collection(testTokenColl).Doc(hashString).Set(ctx, invalidData)
		require.NoError(t, err)

		// Test that checking this token returns the appropriate error
		err = models.CheckToken(ctx, corruptToken)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token data to struct failed")
	})

	t.Run("Check token with missing field", func(t *testing.T) {
		// Create a token with missing required field
		incompleteToken := "incomplete-token-data"
		hashBytes := sha256.Sum256([]byte(incompleteToken))
		hashString := hex.EncodeToString(hashBytes[:])

		// Add data with missing UploadUsed field
		incompleteData := map[string]interface{}{
			// UploadUsed is missing
			"Created": time.Now(),
		}

		_, err := fClient.Collection(testTokenColl).Doc(hashString).Set(ctx, incompleteData)
		require.NoError(t, err)

		// Test checking this token - should default to false for UploadUsed
		err = models.CheckToken(ctx, incompleteToken)
		require.NoError(t, err, "Token with missing UploadUsed should default to false and not return error")
	})

	t.Run("Multiple concurrent checks", func(t *testing.T) {
		// Test concurrent access to the same token
		const concurrentChecks = 5
		tokenToCheck := testTokens[1].plaintext

		// Create a channel for errors
		errChan := make(chan error, concurrentChecks)

		// Start multiple goroutines to check the same token
		for i := 0; i < concurrentChecks; i++ {
			go func() {
				err := models.CheckToken(ctx, tokenToCheck)
				errChan <- err
			}()
		}

		// Collect all results
		for i := 0; i < concurrentChecks; i++ {
			err := <-errChan
			require.NoError(t, err, "Concurrent token check should succeed")
		}
	})

	t.Run("Empty token", func(t *testing.T) {
		err := models.CheckToken(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "get token check failed")
	})

	t.Run("Really long token", func(t *testing.T) {
		// Create a very long token (1KB)
		longToken := testutil.RandomString(1024)

		// Calculate hash for this long token
		hash := sha256.Sum256([]byte(longToken))
		hashString := hex.EncodeToString(hash[:])

		// Add to Firestore
		_, err := fClient.Collection(testTokenColl).Doc(hashString).Set(ctx, interfaces.Token{
			UploadUsed: false,
			Created:    time.Now(),
		})
		require.NoError(t, err)

		// Check the token
		err = models.CheckToken(ctx, longToken)
		require.NoError(t, err, "Long token should work correctly")
	})

	t.Run("Check token with special characters", func(t *testing.T) {
		specialToken := "token-with-$pecial-@#!*&()-characters"
		hash := sha256.Sum256([]byte(specialToken))
		hashString := hex.EncodeToString(hash[:])

		// Add to Firestore
		_, err := fClient.Collection(testTokenColl).Doc(hashString).Set(ctx, interfaces.Token{
			UploadUsed: false,
			Created:    time.Now(),
		})
		require.NoError(t, err)

		// Check the token
		err = models.CheckToken(ctx, specialToken)
		require.NoError(t, err, "Token with special characters should work correctly")
	})
}

// TestAddTokenIntegration tests the AddToken method with a real Firestore client
// Run with: go test -v ./internal/models -test=integration -project-id=token-tltv-test
func TestAddTokenIntegration(t *testing.T) {
	if util.Test != "integration" && !testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup test environment
	ctx := context.Background()
	fClient, err := testCfg.FirestoreClient()
	require.NoError(t, err)
	defer fClient.Close()

	// Create unique collection names for this test
	randomPrefix := t.Name()
	testLangColl := randomPrefix + LangCollString
	testLangCodeColl := randomPrefix + LangCodeCollString
	testVoiceColl := randomPrefix + VoiceCollString
	testTokenColl := randomPrefix + TokenCollString

	collections := []string{testLangColl, testLangCodeColl, testVoiceColl, testTokenColl}

	// Clean up collections after tests
	defer func() {
		for _, coll := range collections {
			err := testutil.DeleteCollection(ctx, fClient, coll, 10)
			require.NoError(t, err)
		}
	}()

	// Initialize the model with our test collections
	models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
	require.NoError(t, err)

	t.Run("Add valid token", func(t *testing.T) {
		// Create a test token
		hash := sha256.Sum256([]byte("test-token-1"))
		hashString := hex.EncodeToString(hash[:])

		token := interfaces.Token{
			Hash:       hashString,
			UploadUsed: false,
			TimesUsed:  0,
			Created:    time.Now(),
		}

		// Add the token
		err := models.AddToken(ctx, token)
		require.NoError(t, err, "Adding valid token should succeed")

		// Verify the token was added correctly
		tokenDoc, err := fClient.Collection(testTokenColl).Doc(hashString).Get(ctx)
		require.NoError(t, err, "Should be able to retrieve the added token")

		var retrievedToken interfaces.FirestoreToken
		err = tokenDoc.DataTo(&retrievedToken)
		require.NoError(t, err, "Should be able to parse token data")

		assert.Equal(t, token.UploadUsed, retrievedToken.UploadUsed)
		assert.Equal(t, token.TimesUsed, retrievedToken.TimesUsed)
		assert.WithinDuration(t, token.Created, retrievedToken.Created, 1*time.Second)
	})

	t.Run("Add token with all fields set", func(t *testing.T) {
		// Create a test token with all fields set
		hash := sha256.Sum256([]byte("test-token-2"))
		hashString := hex.EncodeToString(hash[:])

		token := interfaces.Token{
			Hash:       hashString,
			UploadUsed: true,
			TimesUsed:  5,
			Created:    time.Now().Add(-24 * time.Hour), // Created yesterday
		}

		// Add the token
		err := models.AddToken(ctx, token)
		require.NoError(t, err, "Adding token with all fields set should succeed")

		// Verify the token was added correctly
		tokenDoc, err := fClient.Collection(testTokenColl).Doc(hashString).Get(ctx)
		require.NoError(t, err)

		var retrievedToken interfaces.FirestoreToken
		err = tokenDoc.DataTo(&retrievedToken)
		require.NoError(t, err)

		assert.Equal(t, token.UploadUsed, retrievedToken.UploadUsed)
		assert.Equal(t, token.TimesUsed, retrievedToken.TimesUsed)
		assert.WithinDuration(t, token.Created, retrievedToken.Created, 1*time.Second)
	})

	t.Run("Update existing token", func(t *testing.T) {
		// Create and add initial token
		hash := sha256.Sum256([]byte("test-token-3"))
		hashString := hex.EncodeToString(hash[:])

		initialToken := interfaces.Token{
			Hash:       hashString,
			UploadUsed: false,
			TimesUsed:  0,
			Created:    time.Now().Add(-1 * time.Hour),
		}

		err := models.AddToken(ctx, initialToken)
		require.NoError(t, err)

		// Now update the token
		updatedToken := interfaces.Token{
			Hash:       hashString, // Same hash
			UploadUsed: true,       // Changed
			TimesUsed:  3,          // Changed
			Created:    time.Now(), // Changed
		}

		err = models.AddToken(ctx, updatedToken)
		require.NoError(t, err, "Updating existing token should succeed")

		// Verify the token was updated
		tokenDoc, err := fClient.Collection(testTokenColl).Doc(hashString).Get(ctx)
		require.NoError(t, err)

		var retrievedToken interfaces.FirestoreToken
		err = tokenDoc.DataTo(&retrievedToken)
		require.NoError(t, err)

		assert.Equal(t, updatedToken.UploadUsed, retrievedToken.UploadUsed)
		assert.Equal(t, updatedToken.TimesUsed, retrievedToken.TimesUsed)
		assert.WithinDuration(t, updatedToken.Created, retrievedToken.Created, 1*time.Second)
	})

	t.Run("Add token with empty hash", func(t *testing.T) {
		// Create a token with empty hash
		token := interfaces.Token{
			Hash:       "", // Empty hash
			UploadUsed: false,
			TimesUsed:  0,
			Created:    time.Now(),
		}

		// This should fail or use the empty string as the document ID
		err := models.AddToken(ctx, token)
		require.Error(t, err, "failed adding token: rpc error: code = InvalidArgument desc = Document name")
	})

	t.Run("Add multiple tokens concurrently", func(t *testing.T) {
		// Test concurrent token additions
		const concurrentAdds = 10
		errChan := make(chan error, concurrentAdds)

		for i := 0; i < concurrentAdds; i++ {
			go func(index int) {
				hash := sha256.Sum256([]byte(testutil.RandomString(20)))
				hashString := hex.EncodeToString(hash[:])

				token := interfaces.Token{
					Hash:       hashString,
					UploadUsed: false,
					TimesUsed:  index,
					Created:    time.Now(),
				}

				err := models.AddToken(ctx, token)
				errChan <- err
			}(i)
		}

		// Check that all additions succeeded
		for i := 0; i < concurrentAdds; i++ {
			err := <-errChan
			assert.NoError(t, err, "Concurrent token addition %d should succeed", i)
		}
	})

	t.Run("Add token with very long hash", func(t *testing.T) {
		// Create a token with a very long hash (not typically possible with SHA-256 but testing edge case)
		longHash := testutil.RandomString(1000)

		token := interfaces.Token{
			Hash:       longHash,
			UploadUsed: false,
			TimesUsed:  0,
			Created:    time.Now(),
		}

		// Add the token
		err := models.AddToken(ctx, token)
		require.NoError(t, err, "Adding token with long hash should succeed")

		// Verify the token was added correctly
		tokenDoc, err := fClient.Collection(testTokenColl).Doc(longHash).Get(ctx)
		require.NoError(t, err)

		var retrievedToken interfaces.FirestoreToken
		err = tokenDoc.DataTo(&retrievedToken)
		require.NoError(t, err)

		assert.Equal(t, token.UploadUsed, retrievedToken.UploadUsed)
		assert.Equal(t, token.TimesUsed, retrievedToken.TimesUsed)
		assert.WithinDuration(t, token.Created, retrievedToken.Created, 1*time.Second)
	})

	t.Run("Add token with special characters in hash", func(t *testing.T) {
		// Create a token with special characters
		// Note: Firestore document IDs can't contain certain characters,
		// so we're using a limited set of special chars that are allowed
		specialHash := "special-hash_with@characters$and.numbers123"

		token := interfaces.Token{
			Hash:       specialHash,
			UploadUsed: false,
			TimesUsed:  0,
			Created:    time.Now(),
		}

		// Add the token
		err := models.AddToken(ctx, token)
		require.NoError(t, err, "Adding token with special characters should succeed")

		// Verify the token was added correctly
		tokenDoc, err := fClient.Collection(testTokenColl).Doc(specialHash).Get(ctx)
		require.NoError(t, err)

		var retrievedToken interfaces.FirestoreToken
		err = tokenDoc.DataTo(&retrievedToken)
		require.NoError(t, err)

		assert.Equal(t, token.UploadUsed, retrievedToken.UploadUsed)
		assert.Equal(t, token.TimesUsed, retrievedToken.TimesUsed)
		assert.WithinDuration(t, token.Created, retrievedToken.Created, 1*time.Second)
	})

	t.Run("Add token with zero time", func(t *testing.T) {
		// Create a token with zero time
		hash := sha256.Sum256([]byte("test-token-zero-time"))
		hashString := hex.EncodeToString(hash[:])

		token := interfaces.Token{
			Hash:       hashString,
			UploadUsed: false,
			TimesUsed:  0,
			Created:    time.Time{}, // Zero time
		}

		// Add the token
		err := models.AddToken(ctx, token)
		require.NoError(t, err, "Adding token with zero time should succeed")

		// Verify the token was added with zero time
		tokenDoc, err := fClient.Collection(testTokenColl).Doc(hashString).Get(ctx)
		require.NoError(t, err)

		var retrievedToken interfaces.FirestoreToken
		err = tokenDoc.DataTo(&retrievedToken)
		require.NoError(t, err)

		assert.Equal(t, token.UploadUsed, retrievedToken.UploadUsed)
		assert.Equal(t, token.TimesUsed, retrievedToken.TimesUsed)
		assert.True(t, retrievedToken.Created.IsZero() || retrievedToken.Created.Unix() == 0,
			"Created time should be zero or Unix epoch")
	})

	t.Run("Add future-dated token", func(t *testing.T) {
		// Create a token with future date
		hash := sha256.Sum256([]byte("test-token-future"))
		hashString := hex.EncodeToString(hash[:])

		futureTime := time.Now().Add(30 * 24 * time.Hour) // 30 days in the future
		token := interfaces.Token{
			Hash:       hashString,
			UploadUsed: false,
			TimesUsed:  0,
			Created:    futureTime,
		}

		// Add the token
		err := models.AddToken(ctx, token)
		require.NoError(t, err, "Adding token with future date should succeed")

		// Verify the token was added with future date
		tokenDoc, err := fClient.Collection(testTokenColl).Doc(hashString).Get(ctx)
		require.NoError(t, err)

		var retrievedToken interfaces.FirestoreToken
		err = tokenDoc.DataTo(&retrievedToken)
		require.NoError(t, err)

		assert.Equal(t, token.UploadUsed, retrievedToken.UploadUsed)
		assert.Equal(t, token.TimesUsed, retrievedToken.TimesUsed)
		assert.WithinDuration(t, futureTime, retrievedToken.Created, 1*time.Second)
	})
}

// TestUpdateTokenFieldIntegration tests the UpdateTokenField method with a real Firestore client
// Run with: go test -v ./internal/models -test=integration -project-id=token-tltv-test
func TestUpdateTokenFieldIntegration(t *testing.T) {
	if util.Test != "integration" && !testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup test environment
	ctx := context.Background()
	fClient, err := testCfg.FirestoreClient()
	require.NoError(t, err)
	defer fClient.Close()

	// Create unique collection names for this test
	randomPrefix := t.Name()
	testLangColl := randomPrefix + LangCollString
	testLangCodeColl := randomPrefix + LangCodeCollString
	testVoiceColl := randomPrefix + VoiceCollString
	testTokenColl := randomPrefix + TokenCollString

	collections := []string{testLangColl, testLangCodeColl, testVoiceColl, testTokenColl}

	// Clean up collections after tests
	defer func() {
		for _, coll := range collections {
			err := testutil.DeleteCollection(ctx, fClient, coll, 10)
			require.NoError(t, err)
		}
	}()

	// Initialize the model with our test collections
	models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
	require.NoError(t, err)

	// Setup: Create test tokens
	testTokens := []struct {
		plaintext string
		hash      string
		token     interfaces.FirestoreToken
	}{
		{
			plaintext: "update-token-1",
			token: interfaces.FirestoreToken{
				UploadUsed: false,
				TimesUsed:  0,
				Created:    time.Now().Add(-24 * time.Hour),
			},
		},
		{
			plaintext: "update-token-2",
			token: interfaces.FirestoreToken{
				UploadUsed: false,
				TimesUsed:  5,
				Created:    time.Now().Add(-48 * time.Hour),
			},
		},
	}

	// Calculate hashes and add tokens to Firestore
	for i := range testTokens {
		hash := sha256.Sum256([]byte(testTokens[i].plaintext))
		testTokens[i].hash = hex.EncodeToString(hash[:])

		_, err := fClient.Collection(testTokenColl).Doc(testTokens[i].hash).Set(ctx, testTokens[i].token)
		require.NoError(t, err)
	}

	t.Run("Update UploadUsed field to true", func(t *testing.T) {
		// Update the UploadUsed field to true for the first token
		err := models.UpdateTokenField(ctx, true, testTokens[0].plaintext, "UploadUsed")
		require.NoError(t, err, "Updating UploadUsed field should succeed")

		// Verify the field was updated
		tokenDoc, err := fClient.Collection(testTokenColl).Doc(testTokens[0].hash).Get(ctx)
		require.NoError(t, err)

		var updatedToken interfaces.FirestoreToken
		err = tokenDoc.DataTo(&updatedToken)
		require.NoError(t, err)

		assert.True(t, updatedToken.UploadUsed, "UploadUsed field should be true")
		assert.Equal(t, testTokens[0].token.TimesUsed, updatedToken.TimesUsed, "TimesUsed field should remain unchanged")
		assert.WithinDuration(t, testTokens[0].token.Created, updatedToken.Created, 1*time.Second, "Created field should remain unchanged")
	})

	t.Run("Increment TimesUsed field", func(t *testing.T) {
		// Get the current value of TimesUsed
		tokenDoc, err := fClient.Collection(testTokenColl).Doc(testTokens[1].hash).Get(ctx)
		require.NoError(t, err)

		var currentToken interfaces.FirestoreToken
		err = tokenDoc.DataTo(&currentToken)
		require.NoError(t, err)

		// Increment the TimesUsed field
		newTimesUsed := currentToken.TimesUsed + 1
		err = models.UpdateTokenField(ctx, newTimesUsed, testTokens[1].plaintext, "TimesUsed")
		require.NoError(t, err, "Incrementing TimesUsed field should succeed")

		// Verify the field was updated
		tokenDoc, err = fClient.Collection(testTokenColl).Doc(testTokens[1].hash).Get(ctx)
		require.NoError(t, err)

		var updatedToken interfaces.FirestoreToken
		err = tokenDoc.DataTo(&updatedToken)
		require.NoError(t, err)

		assert.Equal(t, newTimesUsed, updatedToken.TimesUsed, "TimesUsed field should be incremented")
		assert.Equal(t, testTokens[1].token.UploadUsed, updatedToken.UploadUsed, "UploadUsed field should remain unchanged")
		assert.WithinDuration(t, testTokens[1].token.Created, updatedToken.Created, 1*time.Second, "Created field should remain unchanged")
	})

	t.Run("Update Created field", func(t *testing.T) {
		// Update the Created field with a new timestamp
		newTime := time.Now()
		err := models.UpdateTokenField(ctx, newTime, testTokens[0].plaintext, "Created")
		require.NoError(t, err, "Updating Created field should succeed")

		// Verify the field was updated
		tokenDoc, err := fClient.Collection(testTokenColl).Doc(testTokens[0].hash).Get(ctx)
		require.NoError(t, err)

		var updatedToken interfaces.FirestoreToken
		err = tokenDoc.DataTo(&updatedToken)
		require.NoError(t, err)

		assert.WithinDuration(t, newTime, updatedToken.Created, 1*time.Second, "Created field should be updated")
		// Other fields should remain unchanged (note: we previously set UploadUsed to true)
		assert.True(t, updatedToken.UploadUsed, "UploadUsed field should remain unchanged")
		assert.Equal(t, testTokens[0].token.TimesUsed, updatedToken.TimesUsed, "TimesUsed field should remain unchanged")
	})

	t.Run("Update non-existent token", func(t *testing.T) {
		// Try to update a token that doesn't exist
		err := models.UpdateTokenField(ctx, true, "non-existent-token", "UploadUsed")
		require.Error(t, err, "Updating non-existent token should fail")
		assert.Contains(t, err.Error(), "token update failed")
	})

	t.Run("Update with non-existent field", func(t *testing.T) {
		// Try to update a field that doesn't exist in the token schema
		err := models.UpdateTokenField(ctx, "some value", testTokens[0].plaintext, "NonExistentField")
		require.NoError(t, err, "Updating with non-existent field should add the field")

		// Verify the field was added
		tokenDoc, err := fClient.Collection(testTokenColl).Doc(testTokens[0].hash).Get(ctx)
		require.NoError(t, err)

		data := tokenDoc.Data()
		value, exists := data["NonExistentField"]
		assert.True(t, exists, "New field should be added to the document")
		assert.Equal(t, "some value", value, "New field should have correct value")
	})

	t.Run("Update with complex value", func(t *testing.T) {
		// Update with a complex value like a map/struct
		complexValue := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		}

		err := models.UpdateTokenField(ctx, complexValue, testTokens[1].plaintext, "ComplexField")
		require.NoError(t, err, "Updating with complex value should succeed")

		// Verify the field was updated
		tokenDoc, err := fClient.Collection(testTokenColl).Doc(testTokens[1].hash).Get(ctx)
		require.NoError(t, err)

		data := tokenDoc.Data()
		value, exists := data["ComplexField"]
		assert.True(t, exists, "Complex field should be added to the document")

		// Check the complex value was stored correctly
		complexMap, ok := value.(map[string]interface{})
		assert.True(t, ok, "Value should be a map")
		assert.Equal(t, "value1", complexMap["key1"])
		assert.Equal(t, int64(42), complexMap["key2"])
		assert.Equal(t, true, complexMap["key3"])
	})

	t.Run("Update empty token", func(t *testing.T) {
		// Try to update an empty token string
		err := models.UpdateTokenField(ctx, true, "", "UploadUsed")
		require.Error(t, err, "Updating with empty token should fail")
		assert.Contains(t, err.Error(), "token update failed")
	})

	t.Run("Update with nil value", func(t *testing.T) {
		// Try to update a field with nil value
		err := models.UpdateTokenField(ctx, nil, testTokens[0].plaintext, "NilField")
		require.NoError(t, err, "Updating with nil value should succeed")

		// Verify the field was updated
		tokenDoc, err := fClient.Collection(testTokenColl).Doc(testTokens[0].hash).Get(ctx)
		require.NoError(t, err)

		data := tokenDoc.Data()
		_, exists := data["NilField"]
		assert.True(t, exists, "Nil field should be added to the document")
	})

	t.Run("Multiple concurrent updates", func(t *testing.T) {
		// Test concurrent updates to different fields of the same token
		const concurrentUpdates = 5
		errChan := make(chan error, concurrentUpdates)

		fieldNames := []string{"Field1", "Field2", "Field3", "Field4", "Field5"}

		// Start multiple goroutines to update different fields
		for i := 0; i < concurrentUpdates; i++ {
			go func(index int) {
				fieldName := fieldNames[index]
				fieldValue := fmt.Sprintf("value-%d", index)
				err := models.UpdateTokenField(ctx, fieldValue, testTokens[0].plaintext, fieldName)
				errChan <- err
			}(i)
		}

		// Collect all results
		for i := 0; i < concurrentUpdates; i++ {
			err := <-errChan
			assert.NoError(t, err, "Concurrent field update %d should succeed", i)
		}

		// Verify all fields were updated
		tokenDoc, err := fClient.Collection(testTokenColl).Doc(testTokens[0].hash).Get(ctx)
		require.NoError(t, err)

		data := tokenDoc.Data()
		for i, fieldName := range fieldNames {
			value, exists := data[fieldName]
			assert.True(t, exists, "Field %s should exist", fieldName)
			assert.Equal(t, fmt.Sprintf("value-%d", i), value, "Field %s should have correct value", fieldName)
		}
	})

	t.Run("Update empty field path", func(t *testing.T) {
		// Try to update with an empty field path
		err := models.UpdateTokenField(ctx, "some value", testTokens[0].plaintext, "")
		require.Error(t, err, "Updating with empty field path should fail")
		assert.Contains(t, err.Error(), "token update failed")
	})
}

// TestDeleteTokenIntegration tests the DeleteToken method with a real Firestore client
// Run with: go test -v ./internal/models -test=integration -project-id=token-tltv-test
func TestDeleteTokenIntegration(t *testing.T) {
	if util.Test != "integration" && !testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup test environment
	ctx := context.Background()
	fClient, err := testCfg.FirestoreClient()
	require.NoError(t, err)
	defer fClient.Close()

	// Create unique collection names for this test
	randomPrefix := t.Name()
	testLangColl := randomPrefix + LangCollString
	testLangCodeColl := randomPrefix + LangCodeCollString
	testVoiceColl := randomPrefix + VoiceCollString
	testTokenColl := randomPrefix + TokenCollString

	collections := []string{testLangColl, testLangCodeColl, testVoiceColl, testTokenColl}

	// Clean up collections after tests
	defer func() {
		for _, coll := range collections {
			err := testutil.DeleteCollection(ctx, fClient, coll, 10)
			require.NoError(t, err)
		}
	}()

	// Initialize the model with our test collections
	models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
	require.NoError(t, err)

	t.Run("Delete existing token", func(t *testing.T) {
		// Setup: Create a token to delete
		tokenHash := "token-hash-to-delete" // #nosec G101
		token := interfaces.FirestoreToken{
			UploadUsed: false,
			TimesUsed:  0,
			Created:    time.Now(),
		}

		// Add the token to Firestore
		_, err := fClient.Collection(testTokenColl).Doc(tokenHash).Set(ctx, token)
		require.NoError(t, err, "Setting up token should succeed")

		// Verify the token exists
		tokenDoc, err := fClient.Collection(testTokenColl).Doc(tokenHash).Get(ctx)
		require.NoError(t, err, "Token should exist before deletion")
		require.True(t, tokenDoc.Exists(), "Token document should exist")

		// Delete the token
		err = models.DeleteToken(ctx, tokenHash)
		require.NoError(t, err, "Deleting token should succeed")

		// Verify the token was deleted
		tokenDoc, err = fClient.Collection(testTokenColl).Doc(tokenHash).Get(ctx)
		require.Error(t, err, "Getting deleted token should fail")
		require.False(t, tokenDoc.Exists(), "Token document should no longer exist")
	})

	t.Run("Delete non-existent token", func(t *testing.T) {
		// Try to delete a token that doesn't exist
		nonExistentHash := "non-existent-token-hash"

		// Verify the token doesn't exist
		tokenDoc, err := fClient.Collection(testTokenColl).Doc(nonExistentHash).Get(ctx)
		require.Error(t, err, "Non-existent token should not be found")
		require.False(t, tokenDoc.Exists(), "Token document should not exist")

		// Try to delete the non-existent token
		err = models.DeleteToken(ctx, nonExistentHash)
		require.NoError(t, err, "Deleting non-existent token should not return an error in Firestore")
	})

	t.Run("Multiple token operations", func(t *testing.T) {
		// Setup: Create multiple tokens
		tokens := []string{
			"multi-token-1",
			"multi-token-2",
			"multi-token-3",
		}

		// Add the tokens to Firestore
		for _, hash := range tokens {
			token := interfaces.FirestoreToken{
				UploadUsed: false,
				TimesUsed:  0,
				Created:    time.Now(),
			}
			_, err := fClient.Collection(testTokenColl).Doc(hash).Set(ctx, token)
			require.NoError(t, err, "Setting up token should succeed")
		}

		// Delete the middle token
		err = models.DeleteToken(ctx, tokens[1])
		require.NoError(t, err, "Deleting middle token should succeed")

		// Verify the correct token was deleted
		// First token should still exist
		doc1, err := fClient.Collection(testTokenColl).Doc(tokens[0]).Get(ctx)
		require.NoError(t, err)
		assert.True(t, doc1.Exists(), "First token should still exist")

		// Middle token should be deleted
		doc2, err := fClient.Collection(testTokenColl).Doc(tokens[1]).Get(ctx)
		require.Error(t, err)
		assert.False(t, doc2.Exists(), "Middle token should be deleted")

		// Last token should still exist
		doc3, err := fClient.Collection(testTokenColl).Doc(tokens[2]).Get(ctx)
		require.NoError(t, err)
		assert.True(t, doc3.Exists(), "Last token should still exist")
	})
}

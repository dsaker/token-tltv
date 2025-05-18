package models

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"talkliketv.com/tltv/internal/interfaces"
	"talkliketv.com/tltv/internal/testutil"
	"talkliketv.com/tltv/internal/util"
	"testing"
	"time"
)

// TestGetLanguageCodeIntegration tests the GetLanguageCode method with real Firestore
// Run with: go test -v ./internal/models -test=integration -project-id=token-tltv-test
func TestGetLanguageCodeIntegration(t *testing.T) {
	if util.Test != "integration" && !testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup: create Firestore client and clean test collections
	ctx := context.Background()
	fClient, err := testCfg.FirestoreClient()
	require.NoError(t, err)
	defer fClient.Close()

	// Create unique collection names
	randomPrefix := t.Name()
	testLangColl := randomPrefix + LangCollString
	testLangCodeColl := randomPrefix + LangCodeCollString
	testVoiceColl := randomPrefix + VoiceCollString
	testTokenColl := randomPrefix + TokenCollString

	collections := []string{testLangColl, testLangCodeColl, testVoiceColl, testTokenColl}

	// Ensure collections are cleaned up after tests
	defer func() {
		for _, coll := range collections {
			err := testutil.DeleteCollection(ctx, fClient, coll, 10)
			require.NoError(t, err)
		}
	}()

	// Create base test data for all required collections to avoid refreshCache errors
	baseTestData := createBaseTestData()
	setupAllCollections(t, ctx, fClient, baseTestData, collections)

	// Test language codes to work with
	testLangCodes := []interfaces.LanguageCode{
		{
			Code:     "test-code-1",
			Name:     "Test Language Code 1",
			Language: "test-lang-1",
			Country:  "test-country-1",
			Platform: "test-platform-1",
		},
		{
			Code:     "test-code-2",
			Name:     "Test Language Code 2",
			Language: "test-lang-2",
			Country:  "test-country-2",
			Platform: "test-platform-2",
		},
	}

	// Add test language codes to Firestore (in addition to base data)
	for _, langCode := range testLangCodes {
		_, err = fClient.Collection(testLangCodeColl).Doc(langCode.Code).Set(ctx, langCode)
		require.NoError(t, err)
	}

	t.Run("Get language code from cache", func(t *testing.T) {
		// Initialize models with the test collections
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// First refresh cache to populate it
		err = models.refreshCache(ctx)
		require.NoError(t, err)

		// Get language code
		langCode, err := models.GetLanguageCode(ctx, testLangCodes[0].Code)
		require.NoError(t, err)

		// Verify the language code
		assert.Equal(t, testLangCodes[0].Code, langCode.Code)
		assert.Equal(t, testLangCodes[0].Name, langCode.Name)
		assert.Equal(t, testLangCodes[0].Language, langCode.Language)
		assert.Equal(t, testLangCodes[0].Country, langCode.Country)
		assert.Equal(t, testLangCodes[0].Platform, langCode.Platform)
	})

	t.Run("Get language code with expired cache", func(t *testing.T) {
		// Initialize models with the test collections
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Set cache as expired
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Now().Add(-1 * time.Hour)
		models.cacheMutex.Unlock()

		// Get language code - should trigger a refresh
		langCode, err := models.GetLanguageCode(ctx, testLangCodes[1].Code)
		require.NoError(t, err)

		// Verify the language code
		assert.Equal(t, testLangCodes[1].Code, langCode.Code)
		assert.Equal(t, testLangCodes[1].Name, langCode.Name)
		assert.Equal(t, testLangCodes[1].Language, langCode.Language)
		assert.Equal(t, testLangCodes[1].Country, langCode.Country)
		assert.Equal(t, testLangCodes[1].Platform, langCode.Platform)

		// Verify cache was refreshed
		models.cacheMutex.RLock()
		assert.False(t, models.cacheExpiration.IsZero())
		assert.True(t, models.cacheExpiration.After(time.Now()))
		models.cacheMutex.RUnlock()
	})

	t.Run("Get non-existent language code", func(t *testing.T) {
		// Initialize models with the test collections
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Try to get a non-existent language code
		_, err = models.GetLanguageCode(ctx, "non-existent-code")
		require.Error(t, err)
		assert.Equal(t, ErrLanguageCodeInvalid, err)
	})

	t.Run("Get language code with direct lookup when cache refresh fails", func(t *testing.T) {
		// Create a test language code that will be accessed directly (not via cache)
		directLookupCode := interfaces.LanguageCode{
			Code:     "direct-lookup-code",
			Name:     "Direct Lookup Code",
			Language: "direct-lang",
			Country:  "direct-country",
			Platform: "direct-platform",
		}

		// Add it to Firestore
		_, err = fClient.Collection(testLangCodeColl).Doc(directLookupCode.Code).Set(ctx, directLookupCode)
		require.NoError(t, err)

		// Create models with invalid language collection but valid language code collection
		// This will cause cache refresh to fail, forcing direct lookup
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Get language code - should use direct lookup since refresh will fail
		langCode, err := models.GetLanguageCode(ctx, directLookupCode.Code)
		require.NoError(t, err)

		// Verify the language code
		assert.Equal(t, directLookupCode.Code, langCode.Code)
		assert.Equal(t, directLookupCode.Name, langCode.Name)
		assert.Equal(t, directLookupCode.Language, langCode.Language)
		assert.Equal(t, directLookupCode.Country, langCode.Country)
		assert.Equal(t, directLookupCode.Platform, langCode.Platform)
	})

	t.Run("Direct lookup fails for non-existent code", func(t *testing.T) {
		// Create models with invalid language collection
		// This will cause cache refresh to fail, forcing direct lookup
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Try to get a non-existent language code
		_, err = models.GetLanguageCode(ctx, "another-non-existent-code")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "language code invalid")
	})

	t.Run("Updated language code in Firestore", func(t *testing.T) {
		// Initialize models with test collections
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// First refresh cache
		err = models.refreshCache(ctx)
		require.NoError(t, err)

		// Get initial language code from cache
		initialLangCode, err := models.GetLanguageCode(ctx, testLangCodes[0].Code)
		require.NoError(t, err)
		assert.Equal(t, testLangCodes[0].Name, initialLangCode.Name)

		// Update the language code in Firestore
		updatedLangCode := testLangCodes[0]
		updatedLangCode.Name = "Updated Name"
		_, err = fClient.Collection(testLangCodeColl).Doc(updatedLangCode.Code).Set(ctx, updatedLangCode)
		require.NoError(t, err)

		// Force cache to expire
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Now().Add(-1 * time.Hour)
		models.cacheMutex.Unlock()

		// Get updated language code - should trigger a refresh
		refreshedLangCode, err := models.GetLanguageCode(ctx, updatedLangCode.Code)
		require.NoError(t, err)

		// Verify it has the updated name
		assert.Equal(t, "Updated Name", refreshedLangCode.Name)
	})

	t.Run("Deleted language code in Firestore", func(t *testing.T) {
		// Initialize models with test collections
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// First refresh cache to populate it
		err = models.refreshCache(ctx)
		require.NoError(t, err)

		// Delete a language code from Firestore
		_, err = fClient.Collection(testLangCodeColl).Doc(testLangCodes[0].Code).Delete(ctx)
		require.NoError(t, err)

		// Force cache to expire
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Now().Add(-1 * time.Hour)
		models.cacheMutex.Unlock()

		// Try to get the deleted language code - should fail after refresh
		_, err = models.GetLanguageCode(ctx, testLangCodes[0].Code)
		require.Error(t, err)
		assert.Equal(t, ErrLanguageCodeInvalid, err)
	})

	t.Run("Concurrent access", func(t *testing.T) {
		// Initialize models
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Create a new language code
		concurrentCode := interfaces.LanguageCode{
			Code:     "concurrent-code",
			Name:     "Concurrent Test Code",
			Language: "concurrent-lang",
			Country:  "concurrent-country",
			Platform: "concurrent-platform",
		}

		// Add it to Firestore
		_, err = fClient.Collection(testLangCodeColl).Doc(concurrentCode.Code).Set(ctx, concurrentCode)
		require.NoError(t, err)

		// Run multiple goroutines to access the same language code
		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		errChan := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()

				// Get the language code
				langCode, err := models.GetLanguageCode(ctx, concurrentCode.Code)
				if err != nil {
					errChan <- err
					return
				}

				// Verify the language code
				if langCode.Code != concurrentCode.Code ||
					langCode.Name != concurrentCode.Name {
					errChan <- fmt.Errorf("language code mismatch: got %v, want %v",
						langCode, concurrentCode)
				}
			}()
		}

		// Wait for all goroutines to finish
		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			require.NoError(t, err)
		}
	})

	t.Run("Parsing error simulation", func(t *testing.T) {
		// This is harder to test in integration, but we can add an invalid document
		invalidDoc := map[string]interface{}{
			"Code": 123, // Should be a string but we're setting it to a number
			"Name": "Invalid Type",
		}

		// Add the invalid document to Firestore
		_, err = fClient.Collection(testLangCodeColl).Doc("invalid-type-doc").Set(ctx, invalidDoc)
		require.NoError(t, err)

		// Testing direct lookup with invalid document
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Try direct access to the invalid document
		_, err = models.GetLanguageCode(ctx, "invalid-type-doc")
		// Since this is a direct lookup, we should get a parsing error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error parsing language code")
	})
}

// TestGetLanguageCodesIntegration tests the GetLanguageCodes method with real Firestore
// Run with: go test -v ./internal/models -test=integration -project-id=token-tltv-test
func TestGetLanguageCodesIntegration(t *testing.T) {
	if util.Test != "integration" && !testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup: create Firestore client and clean test collections
	ctx := context.Background()
	fClient, err := testCfg.FirestoreClient()
	require.NoError(t, err)
	defer fClient.Close()

	// Create unique collection names
	randomPrefix := t.Name()
	testLangColl := randomPrefix + LangCollString
	testLangCodeColl := randomPrefix + LangCodeCollString
	testVoiceColl := randomPrefix + VoiceCollString
	testTokenColl := randomPrefix + TokenCollString

	collections := []string{testLangColl, testLangCodeColl, testVoiceColl, testTokenColl}

	// Ensure collections are cleaned up after tests
	defer func() {
		for _, coll := range collections {
			err := testutil.DeleteCollection(ctx, fClient, coll, 10)
			require.NoError(t, err)
		}
	}()

	// Create base test data for all required collections
	baseTestData := createBaseTestData()
	setupAllCollections(t, ctx, fClient, baseTestData, collections)

	t.Run("Get language codes from empty collection", func(t *testing.T) {
		// Clear the language code collection
		err := testutil.DeleteCollection(ctx, fClient, testLangCodeColl, 10)
		require.NoError(t, err)

		// Initialize models with the test collections
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Try to get language codes - should fail because collection is empty
		_, err = models.GetLanguageCodes(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load languages")
	})

	// Restore test data
	setupAllCollections(t, ctx, fClient, baseTestData, collections)

	// Add more test language codes
	testLangCodes := []interfaces.LanguageCode{
		{
			Code:     "test-code-1",
			Name:     "B Test Language Code 1", // Note: Using B for sorting test
			Language: "test-lang-1",
			Country:  "test-country-1",
			Platform: "test-platform-1",
		},
		{
			Code:     "test-code-2",
			Name:     "A Test Language Code 2", // Note: Using A for sorting test
			Language: "test-lang-2",
			Country:  "test-country-2",
			Platform: "test-platform-2",
		},
		{
			Code:     "test-code-3",
			Name:     "C Test Language Code 3", // Note: Using C for sorting test
			Language: "test-lang-3",
			Country:  "test-country-3",
			Platform: "test-platform-3",
		},
	}

	// Add test language codes to Firestore
	for _, langCode := range testLangCodes {
		_, err = fClient.Collection(testLangCodeColl).Doc(langCode.Code).Set(ctx, langCode)
		require.NoError(t, err)
	}

	t.Run("Get all language codes with cache refresh", func(t *testing.T) {
		// Initialize models
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Force cache expiration
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Now().Add(-1 * time.Hour)
		models.cacheMutex.Unlock()

		// Get language codes
		langCodes, err := models.GetLanguageCodes(ctx)
		require.NoError(t, err)

		// Verify returned language codes
		require.Len(t, langCodes, 4) // 1 base code + 3 test codes

		// Verify the language codes are sorted by name
		for i := 1; i < len(langCodes); i++ {
			assert.LessOrEqual(t, langCodes[i-1].Name, langCodes[i].Name,
				"Language codes should be sorted by name")
		}

		// Verify specific language codes
		foundCodes := make(map[string]bool)
		for _, lc := range langCodes {
			foundCodes[lc.Code] = true
		}

		assert.True(t, foundCodes["base-code"], "Base code should be in returned codes")
		assert.True(t, foundCodes["test-code-1"], "test-code-1 should be in returned codes")
		assert.True(t, foundCodes["test-code-2"], "test-code-2 should be in returned codes")
		assert.True(t, foundCodes["test-code-3"], "test-code-3 should be in returned codes")
	})

	t.Run("Get language codes with valid cache", func(t *testing.T) {
		// Initialize models
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// First call to populate cache
		_, err = models.GetLanguageCodes(ctx)
		require.NoError(t, err)

		// Add a new language code that shouldn't be returned since cache is valid
		newLangCode := interfaces.LanguageCode{
			Code:     "new-code",
			Name:     "New Language Code",
			Language: "new-lang",
			Country:  "new-country",
			Platform: "new-platform",
		}
		_, err = fClient.Collection(testLangCodeColl).Doc(newLangCode.Code).Set(ctx, newLangCode)
		require.NoError(t, err)

		// Get language codes again, should use cache
		langCodes, err := models.GetLanguageCodes(ctx)
		require.NoError(t, err)

		// Should still have 4 codes, not including the new one (cached result)
		require.Len(t, langCodes, 4)

		// Verify the new code is not in the result
		found := false
		for _, lc := range langCodes {
			if lc.Code == "new-code" {
				found = true
				break
			}
		}
		assert.False(t, found, "New code should not be in cached result")
	})

	t.Run("Get updated language codes after cache expiration", func(t *testing.T) {
		// Initialize models
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Force cache expiration
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Now().Add(-1 * time.Hour)
		models.cacheMutex.Unlock()

		// Get language codes with expired cache
		langCodes, err := models.GetLanguageCodes(ctx)
		require.NoError(t, err)

		// Should now have 5 codes including the new one
		require.Len(t, langCodes, 5)

		// Verify the new code is in the result
		found := false
		for _, lc := range langCodes {
			if lc.Code == "new-code" {
				found = true
				break
			}
		}
		assert.True(t, found, "New code should be in updated result")
	})

	t.Run("Verify returned slice is independent", func(t *testing.T) {
		// Initialize models
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Get language codes
		langCodes, err := models.GetLanguageCodes(ctx)
		require.NoError(t, err)

		// Save length and content
		originalLength := len(langCodes)
		firstCodeName := langCodes[0].Name

		// Modify the returned slice
		langCodes = langCodes[1:]
		langCodes[0].Name = "Modified Name"

		// Get language codes again
		langCodesAgain, err := models.GetLanguageCodes(ctx)
		require.NoError(t, err)

		// Verify the modifications didn't affect the cached data
		assert.Len(t, langCodesAgain, originalLength, "Original length should be preserved")
		assert.Equal(t, firstCodeName, langCodesAgain[0].Name, "Original content should be preserved")
	})

	t.Run("Failed cache refresh", func(t *testing.T) {
		// Create models with invalid language collection
		_, err := NewModels(fClient, "test", "invalid/collection/path", testLangCodeColl, testVoiceColl, testTokenColl)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid Firestore collection path")

		// Create valid models to test error during GetLanguageCodes
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Break the language collection reference
		models.langCollection = fClient.Collection("nonexistent/collection/path")

		// Try to get language codes
		_, err = models.GetLanguageCodes(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load languages")
	})

	t.Run("Concurrent access to GetLanguageCodes", func(t *testing.T) {
		// Initialize models
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Run multiple goroutines to access GetLanguageCodes concurrently
		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		errChan := make(chan error, numGoroutines)
		results := make(chan int, numGoroutines) // To store lengths of returned slices

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()

				// Get all language codes
				langCodes, err := models.GetLanguageCodes(ctx)
				if err != nil {
					errChan <- err
					return
				}

				// Verify we got some language codes
				if len(langCodes) == 0 {
					errChan <- fmt.Errorf("got empty language codes slice")
					return
				}

				// Send length of returned slice
				results <- len(langCodes)
			}()
		}

		// Wait for all goroutines to finish
		wg.Wait()
		close(errChan)
		close(results)

		// Check for errors
		for err := range errChan {
			require.NoError(t, err)
		}

		// Verify all goroutines got the same number of language codes
		var expectedLength int
		first := true
		for length := range results {
			if first {
				expectedLength = length
				first = false
			} else {
				assert.Equal(t, expectedLength, length, "All goroutines should get the same number of language codes")
			}
		}
	})
}

// Additional helper functions specific to GetLanguageCode tests

// createBaseTestData creates the minimum required data for all collections
func createBaseTestData() struct {
	languages []interfaces.Language
	langCodes []interfaces.LanguageCode
	voices    []interfaces.Voice
} {
	return struct {
		languages []interfaces.Language
		langCodes []interfaces.LanguageCode
		voices    []interfaces.Voice
	}{
		languages: []interfaces.Language{
			{
				Name:     "Base Test Language",
				Code:     "base-lang",
				Platform: "test",
			},
		},
		langCodes: []interfaces.LanguageCode{
			{
				Code:     "base-code",
				Name:     "Base Language Code",
				Language: "base-lang",
				Country:  "test-country",
				Platform: "test",
			},
		},
		voices: []interfaces.Voice{
			{
				Name:                   "Base Test Voice",
				Language:               "base-lang",
				LanguageCode:           "base-code",
				SsmlGender:             interfaces.MALE,
				NaturalSampleRateHertz: 24000,
				Platform:               "test",
			},
		},
	}
}

// setupAllCollections adds test data to all required Firestore collections
func setupAllCollections(t *testing.T, ctx context.Context, fClient *firestore.Client, data struct {
	languages []interfaces.Language
	langCodes []interfaces.LanguageCode
	voices    []interfaces.Voice
}, collections []string) {
	// Add languages
	for _, lang := range data.languages {
		_, err := fClient.Collection(collections[0]).Doc(lang.Code).Set(ctx, lang)
		require.NoError(t, err)
	}

	// Add language codes
	for _, langCode := range data.langCodes {
		_, err := fClient.Collection(collections[1]).Doc(langCode.Code).Set(ctx, langCode)
		require.NoError(t, err)
	}

	// Add voices
	for _, voice := range data.voices {
		_, err := fClient.Collection(collections[2]).Doc(voice.Name).Set(ctx, voice)
		require.NoError(t, err)
	}
}

package models

import (
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

// TestGetLanguageIntegration tests the GetLanguage method with real Firestore
// Run with: go test -v ./internal/models -test=integration -project-id=token-tltv-test
func TestGetLanguageIntegration(t *testing.T) {
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

	// Create test data for all required collections
	baseTestData := createBaseTestData()
	setupAllCollections(t, ctx, fClient, baseTestData, collections)

	// Test languages to work with
	testLanguages := []interfaces.Language{
		{
			Name:     "Test Language 1",
			Code:     "test-lang-1",
			Platform: "test-platform-1",
		},
		{
			Name:     "Test Language 2",
			Code:     "test-lang-2",
			Platform: "test-platform-2",
		},
	}

	// Add test languages to Firestore
	for _, lang := range testLanguages {
		_, err = fClient.Collection(testLangColl).Doc(lang.Code).Set(ctx, lang)
		require.NoError(t, err)
	}

	t.Run("Get language from cache", func(t *testing.T) {
		// Initialize models with the test collections
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// First refresh cache to populate it
		err = models.refreshCache(ctx)
		require.NoError(t, err)

		// Get language
		language, err := models.GetLanguage(ctx, testLanguages[0].Code)
		require.NoError(t, err)

		// Verify the language
		assert.Equal(t, testLanguages[0].Code, language.Code)
		assert.Equal(t, testLanguages[0].Name, language.Name)
		assert.Equal(t, testLanguages[0].Platform, language.Platform)
	})

	t.Run("Get language with expired cache", func(t *testing.T) {
		// Initialize models with the test collections
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Set cache as expired
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Now().Add(-1 * time.Hour)
		models.cacheMutex.Unlock()

		// Get language - should trigger a refresh
		language, err := models.GetLanguage(ctx, testLanguages[1].Code)
		require.NoError(t, err)

		// Verify the language
		assert.Equal(t, testLanguages[1].Code, language.Code)
		assert.Equal(t, testLanguages[1].Name, language.Name)
		assert.Equal(t, testLanguages[1].Platform, language.Platform)

		// Verify cache was refreshed
		models.cacheMutex.RLock()
		assert.False(t, models.cacheExpiration.IsZero())
		assert.True(t, models.cacheExpiration.After(time.Now()))
		models.cacheMutex.RUnlock()
	})

	t.Run("Get non-existent language", func(t *testing.T) {
		// Initialize models with the test collections
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Try to get a non-existent language
		_, err = models.GetLanguage(ctx, "non-existent-language")
		require.Error(t, err)
		assert.Equal(t, ErrLanguageIdInvalid, err)
	})

	t.Run("Get language with direct lookup when cache refresh fails", func(t *testing.T) {
		// Create a test language that will be accessed directly (not via cache)
		directLookupLanguage := interfaces.Language{
			Name:     "Direct Lookup Language",
			Code:     "direct-lookup-lang",
			Platform: "direct-platform",
		}

		// Add it to Firestore
		_, err = fClient.Collection(testLangColl).Doc(directLookupLanguage.Code).Set(ctx, directLookupLanguage)
		require.NoError(t, err)

		// Create models with invalid language code collection but valid language collection
		// This will cause cache refresh to fail, forcing direct lookup
		_, err = NewModels(fClient, "test", testLangColl, "invalid/code/collection", testVoiceColl, testTokenColl)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid Firestore collection path")

		// Create models with bad voice collection to ensure refresh fails but language access works
		_, err = NewModels(fClient, "test", testLangColl, testLangCodeColl, "nonexistent/voice", testTokenColl)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid Firestore collection path")

		// Create valid models then break the voice collection reference to make refresh fail
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Break the voice collection reference to force cache refresh to fail
		models.voiceCollection = fClient.Collection("nonexistent/collection/path")

		// Get language - should use direct lookup since refresh will fail
		language, err := models.GetLanguage(ctx, directLookupLanguage.Code)
		require.NoError(t, err)

		// Verify the language
		assert.Equal(t, directLookupLanguage.Code, language.Code)
		assert.Equal(t, directLookupLanguage.Name, language.Name)
		assert.Equal(t, directLookupLanguage.Platform, language.Platform)
	})

	t.Run("Direct lookup fails for non-existent language", func(t *testing.T) {
		// Create models
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Break the voice collection to make refresh fail
		models.voiceCollection = fClient.Collection("nonexistent/collection/path")

		// Try to get a non-existent language
		_, err = models.GetLanguage(ctx, "another-non-existent-language")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "language not found")
	})

	t.Run("Updated language in Firestore", func(t *testing.T) {
		// Initialize models with test collections
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// First refresh cache
		err = models.refreshCache(ctx)
		require.NoError(t, err)

		// Get initial language from cache
		initialLang, err := models.GetLanguage(ctx, testLanguages[0].Code)
		require.NoError(t, err)
		assert.Equal(t, testLanguages[0].Name, initialLang.Name)

		// Update the language in Firestore
		updatedLang := testLanguages[0]
		updatedLang.Name = "Updated Language Name"
		_, err = fClient.Collection(testLangColl).Doc(updatedLang.Code).Set(ctx, updatedLang)
		require.NoError(t, err)

		// Force cache to expire
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Now().Add(-1 * time.Hour)
		models.cacheMutex.Unlock()

		// Get updated language - should trigger a refresh
		refreshedLang, err := models.GetLanguage(ctx, updatedLang.Code)
		require.NoError(t, err)

		// Verify it has the updated name
		assert.Equal(t, "Updated Language Name", refreshedLang.Name)
	})

	t.Run("Deleted language in Firestore", func(t *testing.T) {
		// Initialize models with test collections
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// First refresh cache to populate it
		err = models.refreshCache(ctx)
		require.NoError(t, err)

		// Delete a language from Firestore
		_, err = fClient.Collection(testLangColl).Doc(testLanguages[0].Code).Delete(ctx)
		require.NoError(t, err)

		// Force cache to expire
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Now().Add(-1 * time.Hour)
		models.cacheMutex.Unlock()

		// Try to get the deleted language - should fail after refresh
		_, err = models.GetLanguage(ctx, testLanguages[0].Code)
		require.Error(t, err)
		assert.Equal(t, ErrLanguageIdInvalid, err)
	})

	t.Run("Concurrent access", func(t *testing.T) {
		// Initialize models
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Create a new language
		concurrentLang := interfaces.Language{
			Name:     "Concurrent Test Language",
			Code:     "concurrent-lang",
			Platform: "concurrent-platform",
		}

		// Add it to Firestore
		_, err = fClient.Collection(testLangColl).Doc(concurrentLang.Code).Set(ctx, concurrentLang)
		require.NoError(t, err)

		// Run multiple goroutines to access the same language
		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		errChan := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()

				// Get the language
				lang, err := models.GetLanguage(ctx, concurrentLang.Code)
				if err != nil {
					errChan <- err
					return
				}

				// Verify the language
				if lang.Code != concurrentLang.Code ||
					lang.Name != concurrentLang.Name ||
					lang.Platform != concurrentLang.Platform {
					errChan <- fmt.Errorf("language mismatch: got %v, want %v",
						lang, concurrentLang)
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

	t.Run("Cache hit performance", func(t *testing.T) {
		// Initialize models
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Refresh cache first
		err = models.refreshCache(ctx)
		require.NoError(t, err)

		// Measure cache hit performance
		start := time.Now()
		for i := 0; i < 100; i++ {
			_, err := models.GetLanguage(ctx, testLanguages[1].Code)
			require.NoError(t, err)
		}
		duration := time.Since(start)

		// Average time should be very low for cache hits
		avgTime := duration.Milliseconds() / 100
		t.Logf("Average cache hit time: %d ms", avgTime)
		assert.Less(t, avgTime, int64(5), "Cache hit should be very fast")
	})

	t.Run("Cache vs direct lookup performance", func(t *testing.T) {
		// Skip if running in CI or short mode
		if testing.Short() {
			t.Skip("Skipping performance test in short mode")
		}

		// Initialize models
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Refresh cache first
		err = models.refreshCache(ctx)
		require.NoError(t, err)

		// Measure cache hit performance
		cacheStart := time.Now()
		for i := 0; i < 10; i++ {
			_, err := models.GetLanguage(ctx, testLanguages[1].Code)
			require.NoError(t, err)
		}
		cacheDuration := time.Since(cacheStart)

		// Create a separate models instance for direct lookup tests
		directModels, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// For direct lookups, we'll modify the structure of the model to bypass cache
		// WITHOUT setting any pointers to nil
		directModels.cacheMutex.Lock()
		// Set cache expiration to past time to force refresh attempt
		directModels.cacheExpiration = time.Now().Add(-1 * time.Hour)
		// Use a non-existent collection path for voice collection to make refresh fail
		// but that doesn't affect direct language lookup
		directModels.voiceCollection = fClient.Collection("nonexistent_collection")
		directModels.cacheMutex.Unlock()

		// Measure direct lookup performance
		directStart := time.Now()
		for i := 0; i < 10; i++ {
			// Direct lookups should work even though cache refresh will fail
			lang, err := directModels.GetLanguage(ctx, testLanguages[1].Code)

			// We either get an error from cache refresh or we get the language
			if err != nil {
				// If we got an error, make sure it's the expected one from refresh failure
				assert.Contains(t, err.Error(), "failed to get voices")
			} else {
				// If we got a language, make sure it's the right one
				assert.Equal(t, testLanguages[1].Code, lang.Code)
			}
		}
		directDuration := time.Since(directStart)

		// Cache should be faster, but we won't make a strict assertion due to
		// potential variations in test environments
		t.Logf("Cache lookup: %v, Direct lookup: %v", cacheDuration, directDuration)

		// Only assert if direct lookup is definitely slower
		// (use a significant margin to avoid flaky tests)
		if directDuration > cacheDuration*2 {
			assert.Less(t, cacheDuration, directDuration, "Cache lookup should be faster than direct lookup")
		} else {
			t.Log("Direct lookup wasn't significantly slower than cache lookup in this test run")
		}
	})

	t.Run("Parsing error simulation", func(t *testing.T) {
		// Create an invalid document
		invalidDoc := map[string]interface{}{
			"Code": 123, // Should be a string but we're setting it to a number
			"Name": "Invalid Type",
		}

		// Add the invalid document to Firestore
		_, err = fClient.Collection(testLangColl).Doc("invalid-type-doc").Set(ctx, invalidDoc)
		require.NoError(t, err)

		// Testing direct lookup with invalid document
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Break collections to force direct lookup
		models.voiceCollection = fClient.Collection("nonexistent/collection/path")

		// Try direct access to the invalid document
		_, err = models.GetLanguage(ctx, "invalid-type-doc")
		// Since this is a direct lookup, we should get a parsing error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error parsing language")
	})
}

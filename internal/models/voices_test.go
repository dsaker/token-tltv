package models

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"talkliketv.com/tltv/internal/interfaces"
	"talkliketv.com/tltv/internal/testutil"
	"talkliketv.com/tltv/internal/util"
)

// TestGetVoiceIntegration tests the GetVoice method with a real Firestore client
// Run with: go test -v ./internal/models -test=integration -project-id=token-tltv-test
func TestGetVoiceIntegration(t *testing.T) {
	if util.Test != "integration" && !testing.Short() {
		t.Skip("skipping integration test")
	}

	// Use the application default credentials
	ctx := context.Background()
	fClient, err := testCfg.FirestoreClient()
	require.NoError(t, err)
	defer fClient.Close()

	// Create unique collection names using a random string to avoid conflicts
	testName := t.Name()
	testLangColl := testName + LangCollString
	testLangCodeColl := testName + LangCodeCollString
	testVoiceColl := testName + VoiceCollString
	testTokenColl := testName + TokenCollString

	collections := []string{testLangColl, testLangCodeColl, testVoiceColl, testTokenColl}

	// Initialize the model with our test collections
	models, err := NewModels(fClient, "test", collections[0], collections[1], collections[2], collections[3])
	require.NoError(t, err)

	// Ensure test collections are cleaned up
	defer func() {
		for _, coll := range collections {
			err := testutil.DeleteCollection(ctx, fClient, coll, 10)
			require.NoError(t, err)
		}
	}()

	// Create test voices
	voices := []interfaces.Voice{
		{
			Name:                   "voice-1",
			Language:               "en",
			LanguageCode:           "en-US",
			SsmlGender:             interfaces.MALE,
			NaturalSampleRateHertz: 24000,
			Platform:               "google",
		},
		{
			Name:                   "voice-2",
			Language:               "es",
			LanguageCode:           "es-ES",
			SsmlGender:             interfaces.FEMALE,
			NaturalSampleRateHertz: 22050,
			Platform:               "amazon",
		},
	}

	// Add voices to Firestore
	for _, voice := range voices {
		_, err := fClient.Collection(testVoiceColl).Doc(voice.Name).Set(ctx, voice)
		require.NoError(t, err)
	}

	// Create base test data for all required collections
	baseTestData := createBaseTestData()
	setupAllCollections(t, ctx, fClient, baseTestData, collections)

	// Test cases
	t.Run("Get voice from empty cache, forcing direct lookup", func(t *testing.T) {
		// Ensure cache is empty
		models.cacheMutex.Lock()
		models.voiceCache = []interfaces.Voice{}
		models.cacheExpiration = time.Time{} // Zero time means cache is invalid
		models.cacheMutex.Unlock()

		// Get voice directly from Firestore
		voice, err := models.GetVoice(ctx, "voice-1")
		require.NoError(t, err)

		assert.Equal(t, "voice-1", voice.Name)
		assert.Equal(t, "en", voice.Language)
		assert.Equal(t, "en-US", voice.LanguageCode)
		assert.Equal(t, interfaces.MALE, voice.SsmlGender)
		assert.Equal(t, int32(24000), voice.NaturalSampleRateHertz)
		assert.Equal(t, "google", voice.Platform)
	})

	t.Run("Get voice from cache after refresh", func(t *testing.T) {
		// Force refresh cache
		err := models.refreshCache(ctx)
		require.NoError(t, err)

		// Get voice from cache
		voice, err := models.GetVoice(ctx, "voice-2")
		require.NoError(t, err)

		assert.Equal(t, "voice-2", voice.Name)
		assert.Equal(t, "es", voice.Language)
		assert.Equal(t, "es-ES", voice.LanguageCode)
		assert.Equal(t, interfaces.FEMALE, voice.SsmlGender)
		assert.Equal(t, int32(22050), voice.NaturalSampleRateHertz)
		assert.Equal(t, "amazon", voice.Platform)

		// Verify we're using the cache by checking the cache lock
		// This is a bit of a hack but helps ensure the test is valid
		models.cacheMutex.RLock()
		cacheTime := models.cacheExpiration
		models.cacheMutex.RUnlock()
		assert.False(t, cacheTime.IsZero(), "Cache should have been refreshed with valid expiration time")
	})

	t.Run("Add new voice and get it directly", func(t *testing.T) {
		// Add new voice that isn't in the cache
		newVoice := interfaces.Voice{
			Name:                   "new-voice",
			Language:               "fr",
			LanguageCode:           "fr-FR",
			SsmlGender:             interfaces.MALE,
			NaturalSampleRateHertz: 24000,
			Platform:               "google",
		}

		// Add voice to Firestore
		_, err := fClient.Collection(testVoiceColl).Doc(newVoice.Name).Set(ctx, newVoice)
		require.NoError(t, err)

		// The issue is likely that when calling GetVoice, it first tries to refresh the cache,
		// but due to timing issues, the newly added voice might not be immediately visible
		// in Firestore's query results, causing inconsistent behavior.

		// Solution 1: Explicitly force cache expiration and add delay
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Now().Add(-1 * time.Hour) // Force cache to be invalid
		models.cacheMutex.Unlock()

		// Small delay to ensure Firestore has propagated the write
		time.Sleep(500 * time.Millisecond)

		// Now get the voice - specifying the full document path to bypass cache lookup
		voice, err := models.GetVoice(ctx, "new-voice")
		require.NoError(t, err)

		// Verify the voice was retrieved correctly
		assert.Equal(t, "new-voice", voice.Name)
		assert.Equal(t, "fr", voice.Language)
		assert.Equal(t, "fr-FR", voice.LanguageCode)
		assert.Equal(t, interfaces.MALE, voice.SsmlGender)
		assert.Equal(t, int32(24000), voice.NaturalSampleRateHertz)
		assert.Equal(t, "google", voice.Platform)
	})

	t.Run("Get non-existent voice", func(t *testing.T) {
		// Try to get a voice that doesn't exist
		_, err := models.GetVoice(ctx, "non-existent-voice")
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrVoiceIdInvalid.Error())
	})

	t.Run("Get voice after cache refresh", func(t *testing.T) {
		// Add another new voice
		anotherVoice := interfaces.Voice{
			Name:                   "another-voice",
			Language:               "de",
			LanguageCode:           "de-DE",
			SsmlGender:             interfaces.FEMALE,
			NaturalSampleRateHertz: 22050,
			Platform:               "amazon",
		}

		_, err := fClient.Collection(testVoiceColl).Doc(anotherVoice.Name).Set(ctx, anotherVoice)
		require.NoError(t, err)

		// Force cache to expire
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Now().Add(-1 * time.Hour)
		models.cacheMutex.Unlock()

		// Get voice, which should trigger a cache refresh
		voice, err := models.GetVoice(ctx, "another-voice")
		require.NoError(t, err)

		assert.Equal(t, "another-voice", voice.Name)

		// Verify all voices are now in cache
		models.cacheMutex.RLock()
		defer models.cacheMutex.RUnlock()

		allVoiceNames := make([]string, len(models.voiceCache))
		for i, v := range models.voiceCache {
			allVoiceNames[i] = v.Name
		}

		assert.Contains(t, allVoiceNames, "voice-1")
		assert.Contains(t, allVoiceNames, "voice-2")
		assert.Contains(t, allVoiceNames, "another-voice")
	})

	t.Run("Delete a voice and verify cache refresh removes it", func(t *testing.T) {
		// Delete a voice
		_, err := fClient.Collection(testVoiceColl).Doc("voice-1").Delete(ctx)
		require.NoError(t, err)

		// Force cache to expire
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Now().Add(-1 * time.Hour)
		models.cacheMutex.Unlock()

		// Get any other voice to trigger cache refresh
		_, err = models.GetVoice(ctx, "voice-2")
		require.NoError(t, err)

		// Verify deleted voice is no longer in cache
		models.cacheMutex.RLock()
		var foundDeletedVoice bool
		for _, v := range models.voiceCache {
			if v.Name == "voice-1" {
				foundDeletedVoice = true
				break
			}
		}
		models.cacheMutex.RUnlock()

		assert.False(t, foundDeletedVoice, "Deleted voice should not be in the cache after refresh")
	})

	t.Run("Test cache lookup behavior with stale cache", func(t *testing.T) {
		// Force cache refresh to ensure cache is populated
		err := models.refreshCache(ctx)
		require.NoError(t, err)

		// Update a voice directly in Firestore without updating cache
		updatedVoice := voices[1]
		updatedVoice.LanguageCode = "es-MX" // Change from es-ES to es-MX

		_, err = fClient.Collection(testVoiceColl).Doc(updatedVoice.Name).Set(ctx, updatedVoice)
		require.NoError(t, err)

		// Get the voice from cache (should return old value)
		cachedVoice, err := models.GetVoice(ctx, updatedVoice.Name)
		require.NoError(t, err)
		assert.Equal(t, "es-ES", cachedVoice.LanguageCode, "Should get cached value before refresh")

		// Force cache to expire
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Now().Add(-1 * time.Hour)
		models.cacheMutex.Unlock()

		// Get the voice again (should trigger refresh and return updated value)
		refreshedVoice, err := models.GetVoice(ctx, updatedVoice.Name)
		require.NoError(t, err)
		assert.Equal(t, "es-MX", refreshedVoice.LanguageCode, "Should get updated value after refresh")
	})
}

// TestGetVoicesIntegration tests the GetVoices method with a real Firestore client
// Run with: go test -v ./internal/models -test=integration -project-id=token-tltv-test
func TestGetVoicesIntegration(t *testing.T) {
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

	// Test data setup
	testData := prepareTestData(5) // Create 5 test voices

	// Setup the test data for subsequent tests
	setupTestData(t, ctx, fClient, testData, collections)

	t.Run("Get all voices from cache", func(t *testing.T) {
		// Initialize models with the test collections
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// First call to GetVoices should trigger a refresh
		voices, err := models.GetVoices(ctx)
		require.NoError(t, err)

		// Verify correct number of voices
		assert.Len(t, voices, len(testData.voices), "Should return all test voices")

		// Verify voice content
		for _, expectedVoice := range testData.voices {
			found := false
			for _, actualVoice := range voices {
				if actualVoice.Name == expectedVoice.Name {
					found = true
					assert.Equal(t, expectedVoice.Language, actualVoice.Language)
					assert.Equal(t, expectedVoice.LanguageCode, actualVoice.LanguageCode)
					assert.Equal(t, expectedVoice.SsmlGender, actualVoice.SsmlGender)
					assert.Equal(t, expectedVoice.Platform, actualVoice.Platform)
					break
				}
			}
			assert.True(t, found, "Voice %s should be in results", expectedVoice.Name)
		}

		// Verify second call uses cache
		start := time.Now()
		cachedVoices, err := models.GetVoices(ctx)
		duration := time.Since(start)
		require.NoError(t, err)
		assert.Len(t, cachedVoices, len(voices), "Cached result should have same length")
		assert.Less(t, duration, 50*time.Millisecond, "Cache hit should be very fast")
	})

	t.Run("Get updated voices after cache expiration", func(t *testing.T) {
		// Initialize models with test collections
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// First get original voices
		originalVoices, err := models.GetVoices(ctx)
		require.NoError(t, err)

		// Add a new voice to Firestore
		newVoice := interfaces.Voice{
			Name:                   "New Test Voice",
			Language:               "test-lang-new",
			LanguageCode:           "test-code-new",
			SsmlGender:             interfaces.FEMALE,
			NaturalSampleRateHertz: 24000,
			Platform:               "test-new",
		}
		_, err = fClient.Collection(testVoiceColl).Doc(newVoice.Name).Set(ctx, newVoice)
		require.NoError(t, err)

		// Set cache as expired
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Now().Add(-1 * time.Hour)
		models.cacheMutex.Unlock()

		// Get updated voices - should trigger a refresh
		updatedVoices, err := models.GetVoices(ctx)
		require.NoError(t, err)

		// Verify new total count
		assert.Len(t, updatedVoices, len(originalVoices)+1, "Should have one more voice")

		// Verify new voice is present
		found := false
		for _, voice := range updatedVoices {
			if voice.Name == newVoice.Name {
				found = true
				assert.Equal(t, newVoice.Language, voice.Language)
				assert.Equal(t, newVoice.LanguageCode, voice.LanguageCode)
				assert.Equal(t, newVoice.SsmlGender, voice.SsmlGender)
				assert.Equal(t, newVoice.Platform, voice.Platform)
				break
			}
		}
		assert.True(t, found, "New voice should be in results")
	})

	t.Run("Handle cache refresh error", func(t *testing.T) {
		// Create models with valid collections
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Break the voice collection reference to force cache refresh to fail
		models.voiceCollection = fClient.Collection("invalid/collection/path")

		// Set cache as expired to force refresh attempt
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Now().Add(-1 * time.Hour)
		models.cacheMutex.Unlock()

		// Getting voices should now fail
		_, err = models.GetVoices(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load voices")
	})

	t.Run("Result is a copy of cache", func(t *testing.T) {
		// Initialize models
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Get voices
		voices, err := models.GetVoices(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, voices)

		// Modify the returned slice
		originalName := voices[0].Name
		voices[0].Name = "Modified Voice Name"

		// Get voices again
		newVoices, err := models.GetVoices(ctx)
		require.NoError(t, err)

		// Verify the cache wasn't modified
		found := false
		for _, voice := range newVoices {
			if voice.Name == originalName {
				found = true
				break
			}
		}
		assert.True(t, found, "Original voice name should still be in cache")

		notFound := true
		for _, voice := range newVoices {
			if voice.Name == "Modified Voice Name" {
				notFound = false
				break
			}
		}
		assert.True(t, notFound, "Modified voice name should not be in cache")
	})

	t.Run("Concurrent access", func(t *testing.T) {
		// Initialize models
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// Run multiple goroutines to call GetVoices concurrently
		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		resultsChan := make(chan int, numGoroutines)
		errChan := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()

				voices, err := models.GetVoices(ctx)
				if err != nil {
					errChan <- err
					return
				}

				resultsChan <- len(voices)
			}()
		}

		// Wait for all goroutines to finish
		wg.Wait()
		close(resultsChan)
		close(errChan)

		// Check for errors
		for err := range errChan {
			require.NoError(t, err)
		}

		// Verify all results have the same length
		var expectedLen int
		first := true
		for length := range resultsChan {
			if first {
				expectedLen = length
				first = false
			} else {
				assert.Equal(t, expectedLen, length, "All concurrent calls should return same number of voices")
			}
		}
	})

	t.Run("Performance test", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping performance test in short mode")
		}

		// Initialize models
		models, err := NewModels(fClient, "test", testLangColl, testLangCodeColl, testVoiceColl, testTokenColl)
		require.NoError(t, err)

		// First call to warm up cache
		_, err = models.GetVoices(ctx)
		require.NoError(t, err)

		// Measure performance of multiple cache hits
		start := time.Now()
		iterations := 100
		for i := 0; i < iterations; i++ {
			_, err := models.GetVoices(ctx)
			require.NoError(t, err)
		}
		duration := time.Since(start)

		// Calculate average time per call
		avgTime := duration.Microseconds() / int64(iterations)
		t.Logf("Average GetVoices cache hit time: %d μs", avgTime)

		// Even though we don't have a strict requirement, cache hits should be very fast
		assert.Less(t, avgTime, int64(1000), "GetVoices cache hit should be under 1ms on average")
	})
}

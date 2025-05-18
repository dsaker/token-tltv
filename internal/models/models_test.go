package models

import (
	"cloud.google.com/go/firestore"
	"context"
	"flag"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"talkliketv.com/tltv/internal/config"
	"talkliketv.com/tltv/internal/interfaces"
	"talkliketv.com/tltv/internal/testflags"
	"talkliketv.com/tltv/internal/testutil"
	"talkliketv.com/tltv/internal/util"
	"testing"
	"time"
)

var (
	testCfg config.Config
)

func TestMain(m *testing.M) {
	err := testCfg.SetConfigs()
	if err != nil {
		panic(err)
	}

	testflags.ParseFlags()
	flag.Parse()
	util.Test = testflags.TestType
	os.Exit(testflags.RunTests(m))
}

// TestRefreshCacheIntegration tests the refreshCache method with a real Firestore client
// Run with: go test -v ./internal/models -test=integration -project-id=token-tltv-test
// TestRefreshCacheIntegration provides improved tests for the refreshCache function
func TestRefreshCacheIntegration(t *testing.T) {
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

	// Test setup: Create multiple test entities of each type
	testData := prepareTestData(3) // Create multiple entities of each type
	setupTestData(t, ctx, fClient, testData, collections)

	t.Run("Initial empty cache load", func(t *testing.T) {
		// Start with a clean slate - empty collections
		for _, coll := range collections {
			err := testutil.DeleteCollection(ctx, fClient, coll, 10)
			require.NoError(t, err)
		}

		// Create empty models instance
		emptyModels, err := NewModels(fClient, "test", collections[0], collections[1], collections[2], collections[3])
		require.NoError(t, err)

		// Verify caches are empty but initialized
		emptyModels.cacheMutex.RLock()
		defer emptyModels.cacheMutex.RUnlock()

		assert.Empty(t, emptyModels.languageCache)
		assert.Empty(t, emptyModels.languageCodeCache)
		assert.Empty(t, emptyModels.voiceCache)
		assert.True(t, emptyModels.cacheExpiration.IsZero())
	})

	// Setup the test data again
	setupTestData(t, ctx, fClient, testData, collections)

	t.Run("Initial bulk cache load", func(t *testing.T) {
		// Reset cache expiration
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Time{}
		models.cacheMutex.Unlock()

		// Call refreshCache
		err = models.refreshCache(ctx)
		require.NoError(t, err)

		// Verify cache was fully populated
		verifyCache(t, models, testData)

		// Verify cache has appropriate expiration
		models.cacheMutex.RLock()
		assert.False(t, models.cacheExpiration.IsZero())
		assert.True(t, models.cacheExpiration.After(time.Now()))
		assert.True(t, models.cacheExpiration.Before(time.Now().Add(2*time.Hour))) // Should be less than max duration
		models.cacheMutex.RUnlock()
	})

	t.Run("Cache still valid - should be a no-op", func(t *testing.T) {
		// Set the cache expiration to a future time
		models.cacheMutex.Lock()
		originalExpiration := models.cacheExpiration
		models.cacheExpiration = time.Now().Add(1 * time.Hour)
		models.cacheMutex.Unlock()

		// Call refreshCache
		startTime := time.Now()
		err = models.refreshCache(ctx)
		require.NoError(t, err)
		duration := time.Since(startTime)

		// Should return quickly (no DB operations)
		assert.Less(t, duration, 100*time.Millisecond, "Should return quickly when cache is valid")

		// Verify cache values and expiration didn't change
		models.cacheMutex.RLock()
		assert.WithinDuration(t, time.Now().Add(1*time.Hour), models.cacheExpiration, 5*time.Second)
		models.cacheMutex.RUnlock()

		// Restore original state
		models.cacheMutex.Lock()
		models.cacheExpiration = originalExpiration
		models.cacheMutex.Unlock()
	})

	t.Run("Update cache when expired", func(t *testing.T) {
		// Set expiration to the past
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Now().Add(-1 * time.Hour)
		models.cacheMutex.Unlock()

		// Call refreshCache
		err = models.refreshCache(ctx)
		require.NoError(t, err)

		// Verify cache expiration was updated
		models.cacheMutex.RLock()
		newExpiration := models.cacheExpiration
		models.cacheMutex.RUnlock()

		assert.True(t, newExpiration.After(time.Now()))
	})

	t.Run("Handle data modifications", func(t *testing.T) {
		// Add new items to collections
		newItems := prepareTestData(1)
		newItems.languages[0].Name = "Updated Language"
		newItems.languages[0].Code = "updated-lang"
		newItems.langCodes[0].Name = "Updated Code"
		newItems.langCodes[0].Code = "updated-code"
		newItems.voices[0].Name = "Updated Voice"

		// Add to Firestore
		_, err = fClient.Collection(testLangColl).Doc(newItems.languages[0].Code).Set(ctx, newItems.languages[0])
		require.NoError(t, err)
		_, err = fClient.Collection(testLangCodeColl).Doc(newItems.langCodes[0].Code).Set(ctx, newItems.langCodes[0])
		require.NoError(t, err)
		_, err = fClient.Collection(testVoiceColl).Doc(newItems.voices[0].Name).Set(ctx, newItems.voices[0])
		require.NoError(t, err)

		// Modify an existing item
		if len(testData.languages) > 0 {
			modifiedLang := testData.languages[0]
			modifiedLang.Name = "Modified Name"
			_, err = fClient.Collection(testLangColl).Doc(modifiedLang.Code).Set(ctx, modifiedLang)
			require.NoError(t, err)
		}

		// Delete an item
		if len(testData.voices) > 1 {
			_, err = fClient.Collection(testVoiceColl).Doc(testData.voices[1].Name).Delete(ctx)
			require.NoError(t, err)
		}

		// Force cache refresh
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Now().Add(-1 * time.Hour)
		models.cacheMutex.Unlock()

		// Refresh cache
		err = models.refreshCache(ctx)
		require.NoError(t, err)

		// Verify new items are in cache
		models.cacheMutex.RLock()
		defer models.cacheMutex.RUnlock()

		// Check new language was added
		lang, ok := models.languageCache[newItems.languages[0].Code]
		assert.True(t, ok, "New language should be in cache")
		assert.Equal(t, newItems.languages[0].Name, lang.Name)

		// Check modified language was updated
		if len(testData.languages) > 0 {
			modifiedLang, ok := models.languageCache[testData.languages[0].Code]
			assert.True(t, ok, "Modified language should be in cache")
			assert.Equal(t, "Modified Name", modifiedLang.Name)
		}

		// Check deleted voice is gone
		if len(testData.voices) > 1 {
			deletedFound := false
			for _, v := range models.voiceCache {
				if v.Name == testData.voices[1].Name {
					deletedFound = true
					break
				}
			}
			assert.False(t, deletedFound, "Deleted voice should not be in cache")
		}

		// Find new voice
		newVoiceFound := false
		for _, v := range models.voiceCache {
			if v.Name == newItems.voices[0].Name {
				newVoiceFound = true
				break
			}
		}
		assert.True(t, newVoiceFound, "New voice should be in cache")
	})

	t.Run("Handle errors in collection reads", func(t *testing.T) {
		// Create a new model with an invalid collection name to force an error
		_, err := NewModels(fClient, "test", "invalid/collection/path", collections[1], collections[2], collections[3])

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "warning: Invalid Firestore collection path: invalid/collection/path, error: collection path cannot contain '/'")
	})

	t.Run("Sorting works correctly", func(t *testing.T) {
		// Add languages and voices with specific names to test sorting
		err = testDataWithSortedNames(ctx, fClient, collections)
		require.NoError(t, err)

		// Force cache refresh
		models.cacheMutex.Lock()
		models.cacheExpiration = time.Now().Add(-1 * time.Hour)
		models.cacheMutex.Unlock()

		// Refresh cache
		err = models.refreshCache(ctx)
		require.NoError(t, err)

		// Verify items are sorted correctly
		models.cacheMutex.RLock()
		defer models.cacheMutex.RUnlock()

		// Check language codes are sorted alphabetically by name
		for i := 1; i < len(models.languageCodeCache); i++ {
			assert.LessOrEqual(t,
				models.languageCodeCache[i-1].Name,
				models.languageCodeCache[i].Name,
				"Language codes should be sorted by name")
		}

		// Check voices are sorted alphabetically by name
		for i := 1; i < len(models.voiceCache); i++ {
			assert.LessOrEqual(t,
				models.voiceCache[i-1].Name,
				models.voiceCache[i].Name,
				"Voices should be sorted by name")
		}
	})
}

func TestValidateCollectionPath(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Valid path",
			path:    "validCollection",
			wantErr: false,
		},
		{
			name:    "Empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "Path with slash",
			path:    "invalid/path",
			wantErr: true,
		},
		{
			name:    "Path with dot",
			path:    "invalid.path",
			wantErr: true,
		},
		{
			name:    "Path with double dot",
			path:    "invalid..path",
			wantErr: true,
		},
		{
			name:    "Path with double slash",
			path:    "invalid//path",
			wantErr: true,
		},
		{
			name:    "Very long path",
			path:    strings.Repeat("a", 1501),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCollectionPath(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewModelsWithInvalidCollections(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}
	// Create a mock client
	mockClient := &MockFirestoreClient{}

	// Call NewModels with an invalid collection path
	models, err := NewModels(mockClient, "test", "valid", "invalid/path", "valid2", "valid3")
	require.Error(t, err)
	require.Nil(t, models)
	assert.Contains(t, err.Error(), "warning: Invalid Firestore collection path: invalid/path, error: collection path cannot contain '/'")
}

// Helper structures and functions for the tests

type testDataSet struct {
	languages []interfaces.Language
	langCodes []interfaces.LanguageCode
	voices    []interfaces.Voice
}

// prepareTestData creates a given number of test entities
func prepareTestData(count int) testDataSet {
	data := testDataSet{
		languages: make([]interfaces.Language, count),
		langCodes: make([]interfaces.LanguageCode, count),
		voices:    make([]interfaces.Voice, count),
	}

	for i := 0; i < count; i++ {
		data.languages[i] = interfaces.Language{
			Name:     fmt.Sprintf("Test Language %d", i),
			Code:     fmt.Sprintf("test-lang-%d", i),
			Platform: "test",
		}

		data.langCodes[i] = interfaces.LanguageCode{
			Code:     fmt.Sprintf("test-code-%d", i),
			Name:     fmt.Sprintf("Test Language Code %d", i),
			Language: fmt.Sprintf("test-lang-%d", i),
			Country:  "test",
			Platform: "test",
		}

		data.voices[i] = interfaces.Voice{
			Name:                   fmt.Sprintf("Test Voice %d", i),
			Language:               fmt.Sprintf("test-lang-%d", i),
			LanguageCode:           fmt.Sprintf("test-code-%d", i),
			SsmlGender:             interfaces.MALE,
			NaturalSampleRateHertz: 24000,
			Platform:               "test",
		}
	}

	return data
}

// setupTestData adds test data to Firestore
func setupTestData(t *testing.T, ctx context.Context, fClient *firestore.Client, data testDataSet, collections []string) {
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

// verifyCache checks that all test data is properly loaded in the cache
func verifyCache(t *testing.T, models *Models, data testDataSet) {
	models.cacheMutex.RLock()
	defer models.cacheMutex.RUnlock()

	// Check languages
	for _, lang := range data.languages {
		cachedLang, ok := models.languageCache[lang.Code]
		assert.True(t, ok, "Language %s should be in cache", lang.Code)
		if ok {
			assert.Equal(t, lang.Name, cachedLang.Name)
			assert.Equal(t, lang.Code, cachedLang.Code)
			assert.Equal(t, lang.Platform, cachedLang.Platform)
		}
	}

	// Check language codes
	for _, expected := range data.langCodes {
		found := false
		for _, actual := range models.languageCodeCache {
			if actual.Code == expected.Code {
				found = true
				assert.Equal(t, expected.Name, actual.Name)
				assert.Equal(t, expected.Language, actual.Language)
				assert.Equal(t, expected.Country, actual.Country)
				assert.Equal(t, expected.Platform, actual.Platform)
				break
			}
		}
		assert.True(t, found, "Language code %s should be in cache", expected.Code)
	}

	// Check voices
	for _, expected := range data.voices {
		found := false
		for _, actual := range models.voiceCache {
			if actual.Name == expected.Name {
				found = true
				assert.Equal(t, expected.Language, actual.Language)
				assert.Equal(t, expected.LanguageCode, actual.LanguageCode)
				assert.Equal(t, expected.SsmlGender, actual.SsmlGender)
				assert.Equal(t, expected.NaturalSampleRateHertz, actual.NaturalSampleRateHertz)
				assert.Equal(t, expected.Platform, actual.Platform)
				break
			}
		}
		assert.True(t, found, "Voice %s should be in cache", expected.Name)
	}
}

// testDataWithSortedNames creates test data with names that test sorting
func testDataWithSortedNames(ctx context.Context, fClient *firestore.Client, collections []string) error {
	data := testDataSet{
		languages: []interfaces.Language{},
		langCodes: []interfaces.LanguageCode{
			{Code: "z-code", Name: "Z Language Code", Language: "test", Country: "test", Platform: "test"},
			{Code: "a-code", Name: "A Language Code", Language: "test", Country: "test", Platform: "test"},
			{Code: "m-code", Name: "M Language Code", Language: "test", Country: "test", Platform: "test"},
		},
		voices: []interfaces.Voice{
			{Name: "Z Voice", Language: "test", LanguageCode: "test-code", SsmlGender: interfaces.MALE, NaturalSampleRateHertz: 24000, Platform: "test"},
			{Name: "A Voice", Language: "test", LanguageCode: "test-code", SsmlGender: interfaces.MALE, NaturalSampleRateHertz: 24000, Platform: "test"},
			{Name: "M Voice", Language: "test", LanguageCode: "test-code", SsmlGender: interfaces.MALE, NaturalSampleRateHertz: 24000, Platform: "test"},
		},
	}

	// Add language codes
	for _, langCode := range data.langCodes {
		_, err := fClient.Collection(collections[1]).Doc(langCode.Code).Set(ctx, langCode)
		if err != nil {
			return err
		}
	}

	// Add voices
	for _, voice := range data.voices {
		_, err := fClient.Collection(collections[2]).Doc(voice.Name).Set(ctx, voice)
		if err != nil {
			return err
		}
	}

	return nil
}

// MockFirestoreClient for testing
type MockFirestoreClient struct{}

func (m *MockFirestoreClient) Collection(path string) *firestore.CollectionRef {
	// Just return a nil or mock collection ref
	return nil
}

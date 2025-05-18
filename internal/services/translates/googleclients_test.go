package translates

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
	"talkliketv.com/tltv/internal/interfaces"
	"talkliketv.com/tltv/internal/util"
)

// TestGoogleClientsIntegration tests the GoogleClients implementation with actual Google APIs
// Run with: go test -v ./internal/services/translates -test=integration -project-id=token-tltv-test
func TestGoogleClientsIntegration(t *testing.T) {
	if util.Test != "integration" && !testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create a context with timeout for all tests
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize GoogleClients client
	client, err := NewGoogleTTSClient(ctx)
	require.NoError(t, err)
	require.NotNil(t, client)

	invalidLangTag, _ := language.Parse("xx")

	t.Run("DetectLanguage", func(t *testing.T) {
		testCases := []struct {
			name           string
			texts          []string
			expectedLang   language.Tag
			expectError    bool
			errorSubstring string
		}{
			{
				name:         "Detect English",
				texts:        []string{"Hello world", "This is a test"},
				expectedLang: language.English,
				expectError:  false,
			},
			{
				name:         "Detect Spanish",
				texts:        []string{"Hola mundo", "Esto es una prueba"},
				expectedLang: language.Spanish,
				expectError:  false,
			},
			{
				name:         "Detect French",
				texts:        []string{"Bonjour le monde", "Ceci est un test"},
				expectedLang: language.French,
				expectError:  false,
			},
			{
				name:         "Detect Mixed Languages",
				texts:        []string{"Hello world", "Hola mundo", "Bonjour le monde", "Hello world"},
				expectedLang: language.English, // Should detect most common or highest confidence
				expectError:  false,
			},
			{
				name:           "Empty Texts",
				texts:          []string{},
				expectedLang:   language.Und,
				expectError:    true,
				errorSubstring: "no texts provided",
			},
			{
				name:         "Very Short Text",
				texts:        []string{"Hi"},
				expectedLang: language.English,
				expectError:  false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				detectedLang, err := client.DetectLanguage(ctx, tc.texts)

				if tc.expectError {
					require.Error(t, err)
					if tc.errorSubstring != "" {
						assert.Contains(t, err.Error(), tc.errorSubstring)
					}
				} else {
					require.NoError(t, err)
					// For language detection, we only check the base language
					// since we might get en-US, en-GB, etc.
					assert.Equal(t, tc.expectedLang.String(), detectedLang.String()[:2],
						"Base language should match expected")
				}
			})
		}
	})

	t.Run("TranslateTexts", func(t *testing.T) {
		testCases := []struct {
			name            string
			texts           []string
			targetLang      language.Tag
			expectedResults []string
			expectError     bool
			errorSubstring  string
		}{
			{
				name:            "Translate English to Spanish",
				texts:           []string{"Hello", "World"},
				targetLang:      language.Spanish,
				expectedResults: []string{"Hola", "Mundo"},
				expectError:     false,
			},
			{
				name:            "Translate English to French",
				texts:           []string{"Hello", "World"},
				targetLang:      language.French,
				expectedResults: []string{"Bonjour", "Monde"},
				expectError:     false,
			},
			{
				name:            "Translate Longer Text",
				texts:           []string{"This is a longer text that should be translated completely."},
				targetLang:      language.Spanish,
				expectedResults: []string{"Este es un texto más largo que debería traducirse completamente."},
				expectError:     false,
			},
			{
				name:           "Empty Texts",
				texts:          []string{},
				targetLang:     language.Spanish,
				expectError:    true,
				errorSubstring: "Required Text",
			},
			{
				name:           "Invalid Target Language",
				texts:          []string{"Hello", "World"},
				targetLang:     invalidLangTag,
				expectError:    true,
				errorSubstring: "Invalid Value",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				translations, err := client.TranslateTexts(ctx, tc.texts, tc.targetLang)

				if tc.expectError {
					require.Error(t, err)
					if tc.errorSubstring != "" {
						assert.Contains(t, err.Error(), tc.errorSubstring)
					}
				} else {
					require.NoError(t, err)
					require.Equal(t, len(tc.texts), len(translations))

					// Check if translations contain expected content
					// We do a case-insensitive contains check since translations may vary slightly
					for i, expected := range tc.expectedResults {
						assert.True(t,
							strings.Contains(strings.ToLower(translations[i]), strings.ToLower(expected)),
							"Translation '%s' should contain '%s'", translations[i], expected)
					}
				}
			})
		}
	})

	t.Run("ProcessPhrase", func(t *testing.T) {
		testCases := []struct {
			name          string
			phrase        interfaces.Phrase
			voiceParams   *texttospeechpb.VoiceSelectionParams
			expectError   bool
			expectedAudio bool // Whether we expect non-empty audio content
		}{
			{
				name: "English TTS",
				phrase: interfaces.Phrase{
					ID:   1,
					Text: "Hello, this is a text-to-speech test.",
				},
				voiceParams: &texttospeechpb.VoiceSelectionParams{
					LanguageCode: "en-US",
					SsmlGender:   texttospeechpb.SsmlVoiceGender_MALE,
				},
				expectError:   false,
				expectedAudio: true,
			},
			{
				name: "Spanish TTS",
				phrase: interfaces.Phrase{
					ID:   2,
					Text: "Hola, esta es una prueba de texto a voz.",
				},
				voiceParams: &texttospeechpb.VoiceSelectionParams{
					LanguageCode: "es-ES",
					SsmlGender:   texttospeechpb.SsmlVoiceGender_FEMALE,
				},
				expectError:   false,
				expectedAudio: true,
			},
			{
				name: "Invalid Language Code",
				phrase: interfaces.Phrase{
					ID:   4,
					Text: "This shouldn't work.",
				},
				voiceParams: &texttospeechpb.VoiceSelectionParams{
					LanguageCode: "xx-XX", // Invalid language code
					SsmlGender:   texttospeechpb.SsmlVoiceGender_MALE,
				},
				expectError:   true,
				expectedAudio: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				response, err := client.ProcessPhrase(ctx, tc.phrase, tc.voiceParams)

				if tc.expectError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.NotNil(t, response)

					if tc.expectedAudio {
						// Verify audio content was returned
						assert.NotEmpty(t, response.AudioContent, "Audio content should not be empty")
						assert.Greater(t, len(response.AudioContent), 100, "Audio content should be substantial")
					}
				}
			})
		}
	})

	t.Run("Context Cancellation", func(t *testing.T) {
		// Create a context that's already canceled
		canceledCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Try to use the canceled context with each method
		_, err := client.DetectLanguage(canceledCtx, []string{"Hello world"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context")

		_, err = client.TranslateTexts(canceledCtx, []string{"Hello"}, language.Spanish)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context")

		phrase := interfaces.Phrase{ID: 1, Text: "Hello world"}
		voiceParams := &texttospeechpb.VoiceSelectionParams{
			LanguageCode: "en-US",
			SsmlGender:   texttospeechpb.SsmlVoiceGender_MALE,
		}
		_, err = client.ProcessPhrase(canceledCtx, phrase, voiceParams)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context")
	})

	t.Run("Context Timeout", func(t *testing.T) {
		// Create a context with a very short timeout
		shortCtx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// Wait for timeout to occur
		time.Sleep(5 * time.Millisecond)

		// Try operations with timed-out context
		_, err := client.DetectLanguage(shortCtx, []string{"Hello world"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context")
	})
}

// TestTTSClientPerformance measures the performance of TTSClient operations
// Run with: go test -v ./internal/services/translates -test=integration -project-id=token-tltv-test
func TestTTSClientPerformance(t *testing.T) {
	if util.Test != "integration" && !testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create a context with timeout for all tests
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Initialize GoogleClients client
	client, err := NewGoogleTTSClient(ctx)
	require.NoError(t, err)
	require.NotNil(t, client)

	t.Run("TranslateTexts Performance", func(t *testing.T) {
		texts := []string{
			"This is the first sentence to translate.",
			"Here is another one with different words.",
			"The quick brown fox jumps over the lazy dog.",
			"All good things come to those who wait.",
			"Programming is a blend of logic and creativity.",
		}

		targetLang := language.Spanish

		// Measure time for translation
		start := time.Now()
		translations, err := client.TranslateTexts(ctx, texts, targetLang)
		duration := time.Since(start)

		// Assertions
		require.NoError(t, err)
		require.Equal(t, len(texts), len(translations))
		t.Logf("Translation of %d texts took %v (%.2f ms per text)",
			len(texts), duration, float64(duration.Milliseconds())/float64(len(texts)))
	})

	t.Run("ProcessPhrase Performance", func(t *testing.T) {
		phrases := []interfaces.Phrase{
			{ID: 1, Text: "This is a short phrase."},
			{ID: 2, Text: "This is a slightly longer phrase with more words to process."},
			{ID: 3, Text: "This is an even longer phrase that contains multiple clauses, and should take more time to process into speech."},
		}

		voiceParams := &texttospeechpb.VoiceSelectionParams{
			LanguageCode: "en-US",
			SsmlGender:   texttospeechpb.SsmlVoiceGender_NEUTRAL,
		}

		for _, phrase := range phrases {
			// Measure time for TTS conversion
			start := time.Now()
			response, err := client.ProcessPhrase(ctx, phrase, voiceParams)
			duration := time.Since(start)

			// Assertions
			require.NoError(t, err)
			require.NotNil(t, response)
			assert.NotEmpty(t, response.AudioContent)

			// Calculate audio size in KB
			audioSizeKB := float64(len(response.AudioContent)) / 1024.0

			t.Logf("TTS of %d characters took %v (%.2f ms/char, audio size: %.2f KB)",
				len(phrase.Text), duration,
				float64(duration.Milliseconds())/float64(len(phrase.Text)),
				audioSizeKB)
		}
	})
}

// TestConcurrentTTSRequests tests how the client handles concurrent requests
// Run with: go test -v ./internal/services/translates -test=integration -project-id=token-tltv-test
func TestConcurrentTTSRequests(t *testing.T) {
	if util.Test != "integration" && !testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create a context with timeout for all tests
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Initialize GoogleClients client
	client, err := NewGoogleTTSClient(ctx)
	require.NoError(t, err)
	require.NotNil(t, client)

	t.Run("Concurrent TranslateTexts", func(t *testing.T) {
		const numGoroutines = 5
		const textsPerGoroutine = 3

		// Create sample texts
		texts := [][]string{}
		for i := 0; i < numGoroutines; i++ {
			goroutineTexts := []string{}
			for j := 0; j < textsPerGoroutine; j++ {
				goroutineTexts = append(goroutineTexts,
					fmt.Sprintf("This is test text %d-%d.", i+1, j+1))
			}
			texts = append(texts, goroutineTexts)
		}

		// Create channels for results and errors
		results := make(chan []string, numGoroutines)
		errors := make(chan error, numGoroutines)

		// Start goroutines
		start := time.Now()
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				translations, err := client.TranslateTexts(ctx, texts[idx], language.Spanish)
				if err != nil {
					errors <- err
					return
				}
				results <- translations
			}(i)
		}

		// Collect results
		successCount := 0
		for i := 0; i < numGoroutines; i++ {
			select {
			case result := <-results:
				require.Len(t, result, textsPerGoroutine)
				successCount++
			case err := <-errors:
				t.Errorf("Error in goroutine %d: %v", i, err)
			}
		}

		duration := time.Since(start)
		t.Logf("%d/%d concurrent translation requests completed successfully in %v",
			successCount, numGoroutines, duration)

		require.Equal(t, numGoroutines, successCount, "All goroutines should succeed")
	})

	t.Run("Concurrent ProcessPhrase", func(t *testing.T) {
		const numGoroutines = 5

		// Create sample phrases with different texts
		phrases := []interfaces.Phrase{
			{ID: 1, Text: "This is the first test phrase."},
			{ID: 2, Text: "This is the second test phrase."},
			{ID: 3, Text: "This is the third test phrase."},
			{ID: 4, Text: "This is the fourth test phrase."},
			{ID: 5, Text: "This is the fifth test phrase."},
		}

		// Sample voice params
		voiceParams := &texttospeechpb.VoiceSelectionParams{
			LanguageCode: "en-US",
			SsmlGender:   texttospeechpb.SsmlVoiceGender_NEUTRAL,
		}

		// Create channels for results and errors
		results := make(chan *texttospeechpb.SynthesizeSpeechResponse, numGoroutines)
		errors := make(chan error, numGoroutines)

		// Start goroutines
		start := time.Now()
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				response, err := client.ProcessPhrase(ctx, phrases[idx], voiceParams)
				if err != nil {
					errors <- err
					return
				}
				results <- response
			}(i)
		}

		// Collect results
		successCount := 0
		for i := 0; i < numGoroutines; i++ {
			select {
			case result := <-results:
				require.NotNil(t, result)
				require.NotEmpty(t, result.AudioContent)
				successCount++
			case err := <-errors:
				t.Errorf("Error in goroutine %d: %v", i, err)
			}
		}

		duration := time.Since(start)
		t.Logf("%d/%d concurrent TTS requests completed successfully in %v",
			successCount, numGoroutines, duration)

		require.Equal(t, numGoroutines, successCount, "All goroutines should succeed")
	})
}

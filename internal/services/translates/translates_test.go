package translates

import (
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"talkliketv.com/tltv/internal/interfaces"
	"talkliketv.com/tltv/internal/mock"
	"talkliketv.com/tltv/internal/testflags"
	"talkliketv.com/tltv/internal/util"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/text/language"
)

// internal/interfaces/interfaces_test.go
func TestMain(m *testing.M) {
	testflags.ParseFlags()
	flag.Parse()

	util.Test = testflags.TestType

	os.Exit(testflags.RunTests(m))
}

// Run with: go test -v ./internal/services/translates -test=unit
func TestTranslatePhrases(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}

	t.Parallel()

	tests := []struct {
		name           string
		title          interfaces.Title
		language       interfaces.Language
		setupMocks     func(mockClient *mock.MockTTSClientInterface, mockModels *mock.MockModelsStore)
		expectedResult []interfaces.Phrase
		expectedError  bool
	}{
		{
			name: "Translate success",
			title: interfaces.Title{
				TitleLang:    "en",
				TitlePhrases: []interfaces.Phrase{{ID: 0, Text: "Hello"}, {ID: 1, Text: "World"}},
			},
			language: interfaces.Language{
				Code:     "es",
				Name:     "Spanish",
				Platform: "google",
			},
			setupMocks: func(mockClient *mock.MockTTSClientInterface, mockModels *mock.MockModelsStore) {
				// Mock TranslateTexts to return translated texts
				mockClient.EXPECT().
					TranslateTexts(gomock.Any(), []string{"Hello", "World"}, language.Spanish).
					Return([]string{"Hola", "Mundo"}, nil)
			},
			expectedResult: []interfaces.Phrase{
				{ID: 0, Text: "Hola"},
				{ID: 1, Text: "Mundo"},
			},
			expectedError: false,
		},
		{
			name: "Translate error",
			title: interfaces.Title{
				TitleLang:    "en",
				TitlePhrases: []interfaces.Phrase{{ID: 0, Text: "Hello"}, {ID: 1, Text: "World"}},
			},
			language: interfaces.Language{
				Code:     "es",
				Name:     "Spanish",
				Platform: "google",
			},
			setupMocks: func(mockClient *mock.MockTTSClientInterface, mockModels *mock.MockModelsStore) {
				// Mock a translation error
				mockClient.EXPECT().
					TranslateTexts(gomock.Any(), []string{"Hello", "World"}, language.Spanish).
					Return(nil, errors.New("translation API error"))
			},
			expectedResult: nil,
			expectedError:  true,
		},
		{
			name: "Invalid language code",
			title: interfaces.Title{
				TitleLang:    "en",
				TitlePhrases: []interfaces.Phrase{{ID: 0, Text: "Hello"}},
			},
			language: interfaces.Language{
				Code:     "invalid",
				Name:     "Invalid",
				Platform: "google",
			},
			setupMocks: func(mockClient *mock.MockTTSClientInterface, mockModels *mock.MockModelsStore) {
				// No mocks needed - the language parsing will fail
			},
			expectedResult: nil,
			expectedError:  true,
		},
		{
			name: "Empty phrases",
			title: interfaces.Title{
				TitleLang:    "en",
				TitlePhrases: []interfaces.Phrase{},
			},
			language: interfaces.Language{
				Code:     "es",
				Name:     "Spanish",
				Platform: "google",
			},
			setupMocks: func(mockClient *mock.MockTTSClientInterface, mockModels *mock.MockModelsStore) {
				// No mocks needed since there are no phrases to translate
			},
			expectedResult: nil,
			expectedError:  true,
		},
		{
			name: "Empty response from translation service",
			title: interfaces.Title{
				TitleLang:    "en",
				TitlePhrases: []interfaces.Phrase{{ID: 0, Text: "Hello"}},
			},
			language: interfaces.Language{
				Code:     "es",
				Name:     "Spanish",
				Platform: "google",
			},
			setupMocks: func(mockClient *mock.MockTTSClientInterface, mockModels *mock.MockModelsStore) {
				// Mock a successful call but with empty response
				mockClient.EXPECT().
					TranslateTexts(gomock.Any(), []string{"Hello"}, language.Spanish).
					Return([]string{}, nil)
			},
			expectedResult: nil,
			expectedError:  true,
		},
		{
			name: "Large number of phrases",
			title: func() interfaces.Title {
				phrases := make([]interfaces.Phrase, 50)
				for i := 0; i < 50; i++ {
					phrases[i] = interfaces.Phrase{ID: i, Text: fmt.Sprintf("Text %d", i)}
				}
				return interfaces.Title{
					TitleLang:    "en",
					TitlePhrases: phrases,
				}
			}(),
			language: interfaces.Language{
				Code:     "es",
				Name:     "Spanish",
				Platform: "google",
			},
			setupMocks: func(mockClient *mock.MockTTSClientInterface, mockModels *mock.MockModelsStore) {
				// Create the expected input texts slice
				inputTexts := make([]string, 50)
				for i := 0; i < 50; i++ {
					inputTexts[i] = fmt.Sprintf("Text %d", i)
				}

				// Create the expected translations
				translations := make([]string, 50)
				for i := 0; i < 50; i++ {
					translations[i] = fmt.Sprintf("Translated Text %d", i)
				}

				// Mock a single batch call for all 50 phrases
				mockClient.EXPECT().
					TranslateTexts(gomock.Any(), inputTexts, language.Spanish).
					Return(translations, nil)
			},
			expectedResult: func() []interfaces.Phrase {
				phrases := make([]interfaces.Phrase, 50)
				for i := 0; i < 50; i++ {
					phrases[i] = interfaces.Phrase{ID: i, Text: fmt.Sprintf("Translated Text %d", i)}
				}
				return phrases
			}(),
			expectedError: false,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable for parallel testing
		t.Run(tc.name, func(t *testing.T) {
			// Create controller and mocks
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mock.NewMockTTSClientInterface(ctrl)
			mockModels := mock.NewMockModelsStore(ctrl)

			// Set up mocks
			tc.setupMocks(mockClient, mockModels)

			// Create the service with the mock client
			translateService := New(mockClient, mockModels)

			// Create context
			ctx := context.Background()

			// Call the function
			result, err := translateService.TranslatePhrases(ctx, tc.title, tc.language)

			// Check the results
			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, len(tc.expectedResult), len(result))

				// For large phrase sets, just check the length
				if len(result) <= 10 {
					for i, phrase := range result {
						assert.Equal(t, tc.expectedResult[i].ID, phrase.ID)
						assert.Equal(t, tc.expectedResult[i].Text, phrase.Text)
					}
				} else {
					for i := range result {
						assert.Equal(t, tc.expectedResult[i].ID, result[i].ID)
					}
				}
			}
		})
	}
}

// TestTranslatePhrasesContext tests that the context is properly used and cancellations are handled
func TestTranslatePhrasesContext(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}

	t.Parallel()

	tests := []struct {
		name          string
		setupMocks    func(mockClient *mock.MockTTSClientInterface, mockModels *mock.MockModelsStore)
		contextAction func(cancel context.CancelFunc)
		expectedError string
	}{
		{
			name: "Context cancellation propagates",
			setupMocks: func(mockClient *mock.MockTTSClientInterface, mockModels *mock.MockModelsStore) {
				// Set up a mock that blocks until context is cancelled
				mockClient.EXPECT().
					TranslateTexts(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, texts []string, targetLang language.Tag) ([]string, error) {
						<-ctx.Done() // Block until context is cancelled
						return nil, ctx.Err()
					})
			},
			contextAction: func(cancel context.CancelFunc) {
				time.Sleep(100 * time.Millisecond) // Small delay to ensure the test function has started
				cancel()                           // Cancel the context
			},
			expectedError: "context deadline exceeded",
		},
		{
			name: "Context timeout",
			setupMocks: func(mockClient *mock.MockTTSClientInterface, mockModels *mock.MockModelsStore) {
				// Set up a mock that takes longer than the timeout
				mockClient.EXPECT().
					TranslateTexts(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, texts []string, targetLang language.Tag) ([]string, error) {
						select {
						case <-ctx.Done():
							return nil, ctx.Err()
						case <-time.After(200 * time.Millisecond): // This is longer than our timeout
							return []string{"Should not reach here"}, nil
						}
					})
			},
			contextAction: func(cancel context.CancelFunc) {
				// No explicit cancellation needed - timeout will occur
			},
			expectedError: "context deadline exceeded",
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable for parallel testing
		t.Run(tc.name, func(t *testing.T) {
			// Create controller and mocks
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mock.NewMockTTSClientInterface(ctrl)
			mockModels := mock.NewMockModelsStore(ctrl)

			// Set up mocks
			tc.setupMocks(mockClient, mockModels)

			// Create translate service with the mock client
			translateService := New(mockClient, mockModels)

			// Create test data
			title := interfaces.Title{
				TitleLang:    "en",
				TitlePhrases: []interfaces.Phrase{{ID: 0, Text: "Hello"}},
			}
			language := interfaces.Language{
				Code:     "es",
				Name:     "Spanish",
				Platform: "google",
			}

			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			// Execute the context action in a goroutine
			go tc.contextAction(cancel)

			// Call the function
			_, err := translateService.TranslatePhrases(ctx, title, language)

			// Check the error
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

// TestDetectLanguage tests the DetectLanguage method
func TestDetectLanguage(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}

	t.Parallel()

	tests := []struct {
		name           string
		texts          []string
		setupMocks     func(mockClient *mock.MockTTSClientInterface)
		expectedResult language.Tag
		expectedError  bool
	}{
		{
			name:  "Successful language detection",
			texts: []string{"Hello", "World"},
			setupMocks: func(mockClient *mock.MockTTSClientInterface) {
				// Mock a successful language detection
				mockClient.EXPECT().
					DetectLanguage(gomock.Any(), []string{"Hello", "World"}).
					Return(language.English, nil)
			},
			expectedResult: language.English,
			expectedError:  false,
		},
		{
			name:  "Error in language detection",
			texts: []string{"Hello", "World"},
			setupMocks: func(mockClient *mock.MockTTSClientInterface) {
				// Mock an error in language detection
				mockClient.EXPECT().
					DetectLanguage(gomock.Any(), []string{"Hello", "World"}).
					Return(language.Und, errors.New("detection API error"))
			},
			expectedResult: language.Und,
			expectedError:  true,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable for parallel testing
		t.Run(tc.name, func(t *testing.T) {
			// Create controller and mocks
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mock.NewMockTTSClientInterface(ctrl)
			mockModels := mock.NewMockModelsStore(ctrl)

			// Set up mocks
			tc.setupMocks(mockClient)

			// Create the service with the mock client
			translateService := New(mockClient, mockModels)

			// Create context
			ctx := context.Background()

			// Call the function
			result, err := translateService.DetectLanguage(ctx, tc.texts)

			// Check the results
			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result)
			}
		})
	}
}

// TestCreateTTS tests the CreateTTS method
func TestCreateTTS(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("/tmp", "tts_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name           string
		title          interfaces.Title
		voice          interfaces.Voice
		setupMocks     func(mockClient *mock.MockTTSClientInterface, mockModels *mock.MockModelsStore)
		expectedResult []interfaces.Phrase
		expectedError  bool
	}{
		{
			name: "Successful TTS creation - different language",
			title: interfaces.Title{
				TitleLang: "English",
				TitlePhrases: []interfaces.Phrase{
					{ID: 0, Text: "Hello"},
					{ID: 1, Text: "World"},
				},
			},
			voice: interfaces.Voice{
				Name:         "es-ES-Standard-A",
				Language:     "Spanish",
				LanguageCode: "es-ES",
				SsmlGender:   interfaces.MALE,
			},
			setupMocks: func(mockClient *mock.MockTTSClientInterface, mockModels *mock.MockModelsStore) {
				// Mock the GetLanguage call
				mockModels.EXPECT().
					GetLanguage(gomock.Any(), "Spanish").
					Return(interfaces.Language{
						Name: "Spanish",
						Code: "es-ES",
					}, nil)

				langTag := language.Make("es-ES")
				// Mock the TranslatePhrases call
				mockClient.EXPECT().
					TranslateTexts(gomock.Any(), []string{"Hello", "World"}, langTag).
					Return([]string{"Hola", "Mundo"}, nil)

				// Mock the TextToSpeech call
				mockClient.EXPECT().
					ProcessPhrase(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&texttospeechpb.SynthesizeSpeechResponse{
						AudioContent: []byte("audio content"),
					}, nil).Times(2) // Once for each phrase
			},
			expectedResult: []interfaces.Phrase{
				{ID: 0, Text: "Hola"},
				{ID: 1, Text: "Mundo"},
			},
			expectedError: false,
		},
		{
			name: "GetLanguage error",
			title: interfaces.Title{
				TitleLang: "English",
				TitlePhrases: []interfaces.Phrase{
					{ID: 0, Text: "Hello"},
				},
			},
			voice: interfaces.Voice{
				Name:         "es-ES-Standard-A",
				Language:     "Spanish",
				LanguageCode: "es-ES",
				SsmlGender:   interfaces.MALE,
			},
			setupMocks: func(mockClient *mock.MockTTSClientInterface, mockModels *mock.MockModelsStore) {
				// Mock an error in GetLanguage
				mockModels.EXPECT().
					GetLanguage(gomock.Any(), "Spanish").
					Return(interfaces.Language{}, errors.New("database error"))
			},
			expectedResult: nil,
			expectedError:  true,
		},
		{
			name: "Successful TTS creation - same language",
			title: interfaces.Title{
				TitleLang: "English",
				TitlePhrases: []interfaces.Phrase{
					{ID: 0, Text: "Hello"},
					{ID: 1, Text: "World"},
				},
			},
			voice: interfaces.Voice{
				Name:         "en-US-Standard-A",
				Language:     "English",
				LanguageCode: "en-US",
				SsmlGender:   interfaces.MALE,
			},
			setupMocks: func(mockClient *mock.MockTTSClientInterface, mockModels *mock.MockModelsStore) {
				// Mock the GetLanguage call
				mockModels.EXPECT().
					GetLanguage(gomock.Any(), "English").
					Return(interfaces.Language{
						Name: "English",
						Code: "en-US",
					}, nil)

				// Mock the TextToSpeech call
				mockClient.EXPECT().
					ProcessPhrase(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&texttospeechpb.SynthesizeSpeechResponse{
						AudioContent: []byte("audio content"),
					}, nil).Times(2) // Once for each phrase
			},
			expectedResult: []interfaces.Phrase{
				{ID: 0, Text: "Hello"},
				{ID: 1, Text: "World"},
			},
			expectedError: false,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable for parallel testing
		t.Run(tc.name, func(t *testing.T) {
			// Create controller and mocks
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mock.NewMockTTSClientInterface(ctrl)
			mockModels := mock.NewMockModelsStore(ctrl)

			// Set up mocks
			tc.setupMocks(mockClient, mockModels)

			// Create the service with the mock client
			translateService := New(mockClient, mockModels)

			// Create context
			ctx := context.Background()

			basePath := filepath.Join(tempDir, tc.name)
			// Call the function
			result, err := translateService.CreateTTS(ctx, tc.title, tc.voice, basePath)

			// Check the results
			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, len(tc.expectedResult), len(result))
				for i, phrase := range result {
					assert.Equal(t, tc.expectedResult[i].ID, phrase.ID)
					assert.Equal(t, tc.expectedResult[i].Text, phrase.Text)
				}
			}
		})
	}
}

// TestTextToSpeech tests the TextToSpeech method
func TestTextToSpeech(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}

	t.Parallel()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("/tmp", "tts_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	basePath := filepath.Join(tempDir, "audio")

	tests := []struct {
		name          string
		phrases       []interfaces.Phrase
		voice         interfaces.Voice
		setupMocks    func(mockClient *mock.MockTTSClientInterface)
		expectedError bool
	}{
		{
			name: "Successful TTS generation",
			phrases: []interfaces.Phrase{
				{ID: 0, Text: "Hello"},
				{ID: 1, Text: "World"},
			},
			voice: interfaces.Voice{
				Name:         "en-US-Standard-A",
				LanguageCode: "en-US",
				SsmlGender:   interfaces.MALE,
			},
			setupMocks: func(mockClient *mock.MockTTSClientInterface) {
				// Mock successful ProcessPhrase calls
				mockClient.EXPECT().
					ProcessPhrase(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&texttospeechpb.SynthesizeSpeechResponse{
						AudioContent: []byte("audio content"),
					}, nil).Times(2) // Once for each phrase
			},
			expectedError: false,
		},
		{
			name:    "Empty phrases",
			phrases: []interfaces.Phrase{},
			voice: interfaces.Voice{
				Name:         "en-US-Standard-A",
				LanguageCode: "en-US",
				SsmlGender:   interfaces.MALE,
			},
			setupMocks: func(mockClient *mock.MockTTSClientInterface) {
				// No mocks needed for empty phrases
			},
			expectedError: false, // Should return early without error
		},
		{
			name: "Error in TTS generation",
			phrases: []interfaces.Phrase{
				{ID: 0, Text: "Hello"},
			},
			voice: interfaces.Voice{
				Name:         "en-US-Standard-A",
				LanguageCode: "en-US",
				SsmlGender:   interfaces.MALE,
			},
			setupMocks: func(mockClient *mock.MockTTSClientInterface) {
				// Mock an error in ProcessPhrase
				mockClient.EXPECT().
					ProcessPhrase(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("TTS API error"))
			},
			expectedError: true,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable for parallel testing
		t.Run(tc.name, func(t *testing.T) {
			// Create controller and mocks
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mock.NewMockTTSClientInterface(ctrl)
			mockModels := mock.NewMockModelsStore(ctrl)

			// Set up mocks
			tc.setupMocks(mockClient)

			// Create the service with the mock client
			translateService := New(mockClient, mockModels)

			// Create context
			ctx := context.Background()

			// Call the function
			err := translateService.TextToSpeech(ctx, tc.phrases, tc.voice, basePath)

			// Check the results
			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

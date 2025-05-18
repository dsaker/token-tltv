package translates

import (
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"context"
	"fmt"
	"golang.org/x/text/language"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"talkliketv.com/tltv/internal/interfaces"
	"time"
)

type Translate struct {
	ttsClient TTSClientInterface
	m         interfaces.ModelsStore
}

var _ TranslateX = (*Translate)(nil)

func New(ttsClient TTSClientInterface, m interfaces.ModelsStore) *Translate {
	return &Translate{
		ttsClient: ttsClient,
		m:         m,
	}
}

// TranslateX creates an interface for Translates
type TranslateX interface {
	CreateTTS(c context.Context, title interfaces.Title, voice interfaces.Voice, basePath string) ([]interfaces.Phrase, error)
	TranslatePhrases(c context.Context, title interfaces.Title, lang interfaces.Language) ([]interfaces.Phrase, error)
	DetectLanguage(c context.Context, phrases []string) (language.Tag, error)
}

func (t *Translate) TextToSpeech(ctx context.Context, ts []interfaces.Phrase, voice interfaces.Voice, bp string) error {
	// Early return for empty input
	if len(ts) == 0 {
		return nil
	}

	// Ensure the output directory exists
	if err := os.MkdirAll(bp, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create a context with reasonable timeout
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 30*time.Second)
	defer timeoutCancel()

	// Configure voice parameters
	voiceParams := &texttospeechpb.VoiceSelectionParams{
		LanguageCode: voice.LanguageCode,
		SsmlGender:   texttospeechpb.SsmlVoiceGender_MALE,
		Name:         voice.Name,
	}
	if voice.SsmlGender == interfaces.FEMALE {
		voiceParams.SsmlGender = texttospeechpb.SsmlVoiceGender_FEMALE
	}

	// Setup concurrency control
	const maxConcurrent = 10
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	errChan := make(chan error, 1) // Buffer of 1 to avoid blocking on error reporting

	// Process all phrases concurrently
	for _, phrase := range ts {
		// Skip if context was already canceled
		if timeoutCtx.Err() != nil {
			break
		}

		wg.Add(1)
		phrase := phrase // Capture variable for goroutine

		go func() {
			defer wg.Done()

			// Acquire semaphore or exit if context is done
			select {
			case sem <- struct{}{}:
				// Successfully acquired semaphore
				defer func() { <-sem }() // Release semaphore when done
			case <-timeoutCtx.Done():
				// Context canceled while waiting for semaphore
				return
			}

			// Process the phrase
			resp, err := t.ttsClient.ProcessPhrase(timeoutCtx, phrase, voiceParams)
			if err != nil {
				// Report error (non-blocking)
				select {
				case errChan <- fmt.Errorf("failed to synthesize speech for phrase %d: %w", phrase.ID, err):
					// Successfully sent error
				default:
					// Channel already has an error, just log this one
					log.Printf("TTS error for phrase %d: %v", phrase.ID, err)
				}
				return
			}

			// Save the audio file
			filename := filepath.Join(bp, strconv.Itoa(phrase.ID))
			err = os.WriteFile(filename, resp.AudioContent, 0600)
			if err != nil {
				// Report significant error
				select {
				case errChan <- fmt.Errorf("failed to save audio for phrase %d: %w", phrase.ID, err):
					// Successfully sent error
				default:
					// Channel already has an error, just log this one
					log.Printf("File write error for phrase %d: %v", phrase.ID, err)
				}
			}
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Check for errors
	select {
	case err := <-errChan:
		return err
	default:
		// No errors in channel
	}

	// Check if context was canceled
	if err := timeoutCtx.Err(); err != nil {
		return fmt.Errorf("text-to-speech operation timed out: %w", err)
	}

	return nil
}

// DetectLanguage uses the TTS client to detect the language of text
func (t *Translate) DetectLanguage(ctx context.Context, texts []string) (language.Tag, error) {
	return t.ttsClient.DetectLanguage(ctx, texts)
}

// TranslatePhrases takes a slice of phrases and a language and returns translated phrases
func (t *Translate) TranslatePhrases(c context.Context, title interfaces.Title, lang interfaces.Language) ([]interfaces.Phrase, error) {
	// Use a single timeout context
	ctx, cancel := context.WithTimeout(c, 30*time.Second)
	defer cancel() // Always call the cancel function to release resources

	// Early return for empty input
	if len(title.TitlePhrases) == 0 {
		return nil, fmt.Errorf("no phrases to translate")
	}

	// Check if context is already done
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Extract texts from phrases
	phrases := make([]string, len(title.TitlePhrases))
	for i, phrase := range title.TitlePhrases {
		phrases[i] = phrase.Text
	}

	// Parse language tag
	langTag, err := language.Parse(lang.Code)
	if err != nil {
		return nil, fmt.Errorf("parsing language code %s: %w", lang.Code, err)
	}

	// Make the translation API call using the client interface
	translatedTexts, err := t.ttsClient.TranslateTexts(ctx, phrases, langTag)
	if err != nil {
		return nil, fmt.Errorf("translating text: %w", err)
	}

	// Validate response
	if len(translatedTexts) == 0 {
		return nil, fmt.Errorf("translate returned empty response")
	}

	if len(translatedTexts) != len(phrases) {
		return nil, fmt.Errorf("translate returned mismatched results: got %d, expected %d",
			len(translatedTexts), len(phrases))
	}

	// Create response with original IDs
	responses := make([]interfaces.Phrase, len(title.TitlePhrases))
	for i, text := range translatedTexts {
		responses[i] = interfaces.Phrase{
			ID:   title.TitlePhrases[i].ID, // Preserve original ID
			Text: text,
		}
	}

	return responses, nil
}

// CreateTTS is called from api.createAudioFromTitle.
// It checks if the mp3 audio files exist and if not it creates them.
func (t *Translate) CreateTTS(ctx context.Context, title interfaces.Title, voice interfaces.Voice, basePath string) ([]interfaces.Phrase, error) {
	// Check if the directory already exists to avoid redundant API calls
	exists, err := pathExists(basePath)
	if err != nil {
		return nil, fmt.Errorf("checking path existence: %w", err)
	}

	// If audio files already exist, return early
	if exists {
		return title.TitlePhrases, nil
	}

	// Fetch voice details if not fully populated (but only if needed)
	if voice.Language == "" {
		voice, err = t.m.GetVoice(ctx, voice.Name)
		if err != nil {
			return nil, fmt.Errorf("getting voice: %w", err)
		}
	}

	// Fetch language details
	lang, err := t.m.GetLanguage(ctx, voice.Language)
	if err != nil {
		return nil, fmt.Errorf("getting language: %w", err)
	}

	// If target language matches title language, no translation needed
	if title.TitleLang == lang.Name {
		// Create directory for audio files
		if err = os.MkdirAll(basePath, 0777); err != nil {
			return nil, fmt.Errorf("creating directory: %w", err)
		}

		// Generate speech for original phrases
		if err = t.TextToSpeech(ctx, title.TitlePhrases, voice, basePath); err != nil {
			return nil, fmt.Errorf("generating speech: %w", err)
		}

		return title.TitlePhrases, nil
	}

	// Translation needed - translate first
	translates, err := t.TranslatePhrases(ctx, title, lang)
	if err != nil {
		return nil, fmt.Errorf("translating phrases: %w", err)
	}

	// Create directory for audio files
	if err = os.MkdirAll(basePath, 0777); err != nil {
		return nil, fmt.Errorf("creating directory: %w", err)
	}

	// Generate speech from translated text
	if err = t.TextToSpeech(ctx, translates, voice, basePath); err != nil {
		return nil, fmt.Errorf("generating speech: %w", err)
	}

	return translates, nil
}

// Helper function for path existence check
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

package translates

import (
	tts "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"cloud.google.com/go/translate"
	"context"
	"fmt"
	"github.com/googleapis/gax-go/v2"
	"golang.org/x/text/language"
	"talkliketv.com/tltv/internal/interfaces"
)

// TTSClientInterface defines the operations needed from a text-to-speech client
type TTSClientInterface interface {
	// ProcessPhrase handles the TTS process for a single phrase
	ProcessPhrase(ctx context.Context, phrase interfaces.Phrase, params *texttospeechpb.VoiceSelectionParams) (*texttospeechpb.SynthesizeSpeechResponse, error)

	// TranslateTexts translates a set of texts from one language to another
	TranslateTexts(ctx context.Context, texts []string, targetLang language.Tag) ([]string, error)

	// DetectLanguage detects the language of provided texts
	DetectLanguage(ctx context.Context, texts []string) (language.Tag, error)
}

// GoogleTranslateClientX creates an interface for google translate.Translate so it can
// be mocked for testing
type GoogleTranslateClientX interface {
	Translate(context.Context, []string, language.Tag, *translate.Options) ([]translate.Translation, error)
	DetectLanguage(context.Context, []string) ([][]translate.Detection, error)
}

// GoogleTTSClientX creates an interface for google texttospeechpb.SynthesizeSpeech so
// it can be mocked for testing
type GoogleTTSClientX interface {
	SynthesizeSpeech(context.Context, *texttospeechpb.SynthesizeSpeechRequest, ...gax.CallOption) (*texttospeechpb.SynthesizeSpeechResponse, error)
}

// GoogleClients implements TTSClientInterface using Google APIs
type GoogleClients struct {
	gtc  GoogleTranslateClientX
	gtts GoogleTTSClientX
}

// Ensure GoogleClients implements TTSClientInterface
var _ TTSClientInterface = (*GoogleClients)(nil)

// NewGoogleTTSClient creates a new TTSClientInterface implementation using Google services
func NewGoogleTTSClient(c context.Context) (TTSClientInterface, error) {
	// Create Google clients
	transClient, err := translate.NewClient(c)
	if err != nil {
		return nil, fmt.Errorf("creating Google Translate client: %w", err)
	}

	ttsClient, err := tts.NewClient(c)
	if err != nil {
		return nil, fmt.Errorf("creating Google Text-to-Speech client: %w", err)
	}

	return &GoogleClients{
		gtc:  transClient,
		gtts: ttsClient,
	}, nil
}

// ProcessPhrase handles the TTS process for a single phrase
func (g *GoogleClients) ProcessPhrase(
	ctx context.Context,
	phrase interfaces.Phrase,
	params *texttospeechpb.VoiceSelectionParams,
) (*texttospeechpb.SynthesizeSpeechResponse, error) {
	// Check if context is already canceled
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Build the TTS request
	req := &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: phrase.Text},
		},
		Voice: params,
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_MP3,
		},
	}

	// Execute TTS request
	return g.gtts.SynthesizeSpeech(ctx, req)
}

// TranslateTexts translates a set of texts from one language to another
func (g *GoogleClients) TranslateTexts(
	ctx context.Context,
	texts []string,
	targetLang language.Tag,
) ([]string, error) {
	translations, err := g.gtc.Translate(ctx, texts, targetLang, nil)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(translations))
	for i, t := range translations {
		result[i] = t.Text
	}
	return result, nil
}

// DetectLanguage detects the language of provided texts
func (g *GoogleClients) DetectLanguage(ctx context.Context, texts []string) (language.Tag, error) {
	if len(texts) == 0 {
		return language.Und, fmt.Errorf("no texts provided")
	}

	detections, err := g.gtc.DetectLanguage(ctx, texts)
	if err != nil {
		return language.Und, err
	}

	// Language detection logic (same as original implementation)
	if len(detections) == 0 {
		return language.Und, fmt.Errorf("no languages detected")
	}

	// Count language occurrences
	langCounts := make(map[string]int)
	var highestConfidence float64
	var mostConfidentLang language.Tag

	for _, textDetections := range detections {
		if len(textDetections) == 0 {
			continue
		}

		// Track the language with highest confidence from each text
		for _, detection := range textDetections {
			langCode := detection.Language.String()
			langCounts[langCode]++

			// Also keep track of the single highest confidence detection
			if detection.Confidence > highestConfidence {
				highestConfidence = detection.Confidence
				mostConfidentLang = detection.Language
			}
		}
	}

	// Find the most common language
	var mostCommonLang string
	maxCount := 1
	for lang, count := range langCounts {
		if count > maxCount {
			mostCommonLang = lang
			maxCount = count
		}
	}

	// If we found a most common language, use that
	if mostCommonLang != "" {
		tag, err := language.Parse(mostCommonLang)
		if err == nil {
			return tag, nil
		}
	}

	// Fall back to the highest confidence detection
	if mostConfidentLang != language.Und {
		return mostConfidentLang, nil
	}

	return language.Und, fmt.Errorf("could not determine most common language")
}

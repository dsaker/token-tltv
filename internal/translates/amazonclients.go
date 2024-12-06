package translates

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/aws/aws-sdk-go-v2/service/polly/types"
	"github.com/aws/aws-sdk-go-v2/service/translate"
	"github.com/labstack/echo/v4"
	"log"
	"os"
	"strconv"
	"sync"
	"talkliketv.click/tltv/internal/models"
)

// AmazonTranslateClientX creates an interface for amazon translate so it can be mocked for testing
type AmazonTranslateClientX interface {
	TranslateText(context.Context, *translate.TranslateTextInput, ...func(*translate.Options)) (*translate.TranslateTextOutput, error)
}

// AmazonTTSClientX creates an interface for amazon texttospeechpb so it can be mocked for testing
type AmazonTTSClientX interface {
	SynthesizeSpeech(context.Context, *polly.SynthesizeSpeechInput, ...func(*polly.Options)) (*polly.SynthesizeSpeechOutput, error)
}

type AmazonClients struct {
	atc  AmazonTranslateClientX
	atts AmazonTTSClientX
}

// NewAmazonClients creates new amazon translate and text-to-speech clients; constructs
// the dependencies and returns them
func NewAmazonClients() *AmazonClients {
	// create amazon translate and text-to-speech clients
	ctx := context.Background()
	// Initialize AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// Create an AWS Translate client
	translateClient := translate.NewFromConfig(cfg)

	// Create a Polly client
	ttsClient := polly.NewFromConfig(cfg)

	// Input text to translate
	text := "Hello, how are you?"
	sourceLanguage := "en" // Source language code (e.g., "en" for English)
	targetLanguage := "es" // Target language code (e.g., "es" for Spanish)

	// Call the TranslateText API
	output, err := translateClient.TranslateText(context.TODO(), &translate.TranslateTextInput{
		Text:               &text,
		SourceLanguageCode: &sourceLanguage,
		TargetLanguageCode: &targetLanguage,
	})
	if err != nil {
		log.Fatalf("Failed to translate text: %v", err)
	}

	// Print the translated text
	fmt.Printf("Original text: %s\n", text)
	fmt.Printf("Translated text: %s\n", *output.TranslatedText)
	return &AmazonClients{atc: translateClient, atts: ttsClient}
}

// GetTranslate is a helper function for TranslatePhrases that allows concurrent calls to
// google translate.Translate.
// It receives a context.CancelFunc that is invoked on an error so all subsequent calls to
// google translate.Translate can be aborted
func (g *AmazonClients) GetTranslate(e echo.Context,
	ctx context.Context,
	cancel context.CancelFunc,
	phrase models.Phrase,
	wg *sync.WaitGroup,
	toLang string,
	fromLang string,
	responses []models.Phrase,
	i int,
) {
	defer wg.Done()
	select {
	case <-ctx.Done():
		return // Error somewhere, terminate
	default: // Default to avoid blocking

		resp, err := g.atc.TranslateText(ctx, &translate.TranslateTextInput{
			Text:               &phrase.Text,
			SourceLanguageCode: &fromLang,
			TargetLanguageCode: &toLang,
		})
		if err != nil {
			switch {
			case errors.Is(err, context.Canceled):
				return
			default:
				e.Logger().Error(fmt.Errorf("error translating text: %s", err))
				cancel()
				return
			}
		}

		responses[i] = models.Phrase{
			ID:   phrase.ID,
			Text: *resp.TranslatedText,
		}
	}
}

// GetSpeech is a helper function for TextToSpeech that is run concurrently.
// it is passed a cancel context, so if one routine fails, the following routines can
// be canceled
func (g *AmazonClients) GetSpeech(
	e echo.Context,
	ctx context.Context,
	cancel context.CancelFunc,
	translate models.Phrase,
	voiceID string,
	outputFormat string,
	wg *sync.WaitGroup,
	basePath string) {
	defer wg.Done()
	select {
	case <-ctx.Done():
		return // Error somewhere, terminate
	default:
		resp, err := g.atts.SynthesizeSpeech(ctx, &polly.SynthesizeSpeechInput{
			Text:         &translate.Text,
			VoiceId:      types.VoiceId(voiceID),
			OutputFormat: types.OutputFormat(outputFormat),
		})
		if err != nil {
			switch {
			case errors.Is(err, context.Canceled):
				return
			default:
				e.Logger().Error(fmt.Errorf("error creating Synthesize Speech client: %s", err))
				cancel()
				return
			}
		}

		// Save the output to an MP3 file
		outputFile := basePath + strconv.Itoa(translate.ID)
		file, err := os.Create(outputFile)
		if err != nil {
			log.Fatalf("failed to create file, %v", err)
		}
		defer file.Close()

		// Write the audio stream to the file
		_, err = file.ReadFrom(resp.AudioStream)
		if err != nil {
			log.Fatalf("failed to write audio stream to file, %v", err)
		}
	}
}

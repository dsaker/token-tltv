package translates

import (
	tts "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"cloud.google.com/go/translate"
	"context"
	"errors"
	"fmt"
	"github.com/googleapis/gax-go/v2"
	"github.com/labstack/echo/v4"
	"golang.org/x/text/language"
	"log"
	"os"
	"strconv"
	"sync"
	"talkliketv.click/tltv/internal/models"
)

// GoogleTranslateClientX creates an interface for google translate.Translate so it can
// be mocked for testing
type GoogleTranslateClientX interface {
	Translate(context.Context, []string, language.Tag, *translate.Options) ([]translate.Translation, error)
}

// GoogleTTSClientX creates an interface for google texttospeechpb.SynthesizeSpeech so
// it can be mocked for testing
type GoogleTTSClientX interface {
	SynthesizeSpeech(context.Context, *texttospeechpb.SynthesizeSpeechRequest, ...gax.CallOption) (*texttospeechpb.SynthesizeSpeechResponse, error)
}

type GoogleClients struct {
	gtc  GoogleTranslateClientX
	gtts GoogleTTSClientX
}

// NewGoogleClients creates a new google translate and text-to-speech clients; constructs
// the translate and audiofile dependencies and returns them
func NewGoogleClients() *GoogleClients {
	// create google translate and text-to-speech clients
	ctx := context.Background()
	transClient, err := translate.NewClient(ctx)
	if err != nil {
		log.Fatalf("Error creating google api translate client\n: %s", err)
	}
	ttsClient, err := tts.NewClient(ctx)
	if err != nil {
		log.Fatalf("Error creating google api translate client\n: %s", err)
	}
	return &GoogleClients{
		gtc:  transClient,
		gtts: ttsClient,
	}
}

// GetTranslate is a helper function for TranslatePhrases that allows concurrent calls to
// google translate.Translate.
// It receives a context.CancelFunc that is invoked on an error so all subsequent calls to
// google translate.Translate can be aborted
func (g *GoogleClients) GetTranslate(e echo.Context,
	ctx context.Context,
	cancel context.CancelFunc,
	phrase models.Phrase,
	wg *sync.WaitGroup,
	lang language.Tag,
	responses []models.Phrase,
	i int,
) {
	defer wg.Done()
	select {
	case <-ctx.Done():
		return // Error somewhere, terminate
	default: // Default to avoid blocking
		resp, err := g.gtc.Translate(ctx, []string{phrase.Text}, lang, nil)
		if err != nil {
			switch {
			case errors.Is(err, context.Canceled):
				return
			default:
				e.Logger().Error(fmt.Errorf("error translating text: %s", err))
				cancel()
			}
			return
		}

		if len(resp) == 0 {
			e.Logger().Error(fmt.Errorf("translate returned empty response to text: %s", err))
			cancel()
		}

		responses[i] = models.Phrase{
			ID:   phrase.ID,
			Text: resp[0].Text,
		}
	}
}

// GetSpeech is a helper function for TextToSpeech that is run concurrently.
// it is passed a cancel context, so if one routine fails, the following routines can
// be canceled
func (g *GoogleClients) GetSpeech(
	e echo.Context,
	ctx context.Context,
	cancel context.CancelFunc,
	translate models.Phrase,
	wg *sync.WaitGroup,
	params *texttospeechpb.VoiceSelectionParams,
	basePath string) {
	defer wg.Done()
	select {
	case <-ctx.Done():
		return // Error somewhere, terminate
	default:
		// Perform the text-to-speech request on the text input with the selected
		// voice parameters and audio file type.
		req := texttospeechpb.SynthesizeSpeechRequest{
			// Set the text input to be synthesized.
			Input: &texttospeechpb.SynthesisInput{
				InputSource: &texttospeechpb.SynthesisInput_Text{Text: translate.Text},
			},
			// Build the voice request, select the language code ("en-US") and the SSML
			// voice gender ("neutral").
			Voice: params,
			// Select the type of audio file you want returned.
			AudioConfig: &texttospeechpb.AudioConfig{
				AudioEncoding: texttospeechpb.AudioEncoding_MP3,
			},
		}

		resp, err := g.gtts.SynthesizeSpeech(ctx, &req)
		if err != nil {
			e.Logger().Error(fmt.Errorf("error creating Synthesize Speech client: %s", err))
			cancel()
			return
		}

		// The resp AudioContent is binary.
		filename := basePath + strconv.Itoa(translate.ID)
		err = os.WriteFile(filename, resp.AudioContent, 0600)
		if err != nil {
			e.Logger().Error(fmt.Errorf("error creating translate client: %s", err))
			cancel()
			return
		}
	}
}

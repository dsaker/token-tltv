package translates

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/aws/aws-sdk-go-v2/service/polly/types"
	"github.com/aws/aws-sdk-go-v2/service/translate"
	"github.com/labstack/echo/v4"
	"talkliketv.click/tltv/internal/models"
)

type AmazonTranslateClientX interface {
	TranslateText(context.Context, *translate.TranslateTextInput, ...func(*translate.Options)) (*translate.TranslateTextOutput, error)
}

type AmazonTTSClientX interface {
	SynthesizeSpeech(context.Context, *polly.SynthesizeSpeechInput, ...func(*polly.Options)) (*polly.SynthesizeSpeechOutput, error)
}

type AmazonClients struct {
	atc  AmazonTranslateClientX
	atts AmazonTTSClientX
}

func NewAmazonClients(ctx context.Context) *AmazonClients {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}
	return &AmazonClients{
		atc:  translate.NewFromConfig(cfg),
		atts: polly.NewFromConfig(cfg),
	}
}

func (g *AmazonClients) GetTranslate(
	e echo.Context,
	ctx context.Context,
	cancel context.CancelFunc,
	phrase models.Phrase,
	wg *sync.WaitGroup,
	toLang, fromLang string,
	responses []models.Phrase,
	i int,
) {
	defer wg.Done()
	if ctx.Err() != nil {
		return
	}

	resp, err := g.atc.TranslateText(ctx, &translate.TranslateTextInput{
		Text:               &phrase.Text,
		SourceLanguageCode: &fromLang,
		TargetLanguageCode: &toLang,
	})
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			e.Logger().Error(fmt.Errorf("error translating text: %v", err))
			cancel()
		}
		return
	}

	responses[i] = models.Phrase{ID: phrase.ID, Text: *resp.TranslatedText}
}

func (g *AmazonClients) GetSpeech(
	e echo.Context,
	ctx context.Context,
	cancel context.CancelFunc,
	translate models.Phrase,
	voice models.Voice,
	wg *sync.WaitGroup,
	basePath string,
) {
	defer wg.Done()
	if ctx.Err() != nil {
		return
	}

	resp, err := g.atts.SynthesizeSpeech(ctx, &polly.SynthesizeSpeechInput{
		Text:         &translate.Text,
		VoiceId:      types.VoiceId(voice.Name),
		OutputFormat: "mp3",
	})
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			e.Logger().Error(fmt.Errorf("error synthesizing speech: %v", err))
			cancel()
		}
		return
	}

	outputFile := basePath + strconv.Itoa(translate.ID)
	file, err := os.Create(outputFile)
	if err != nil {
		e.Logger().Error(fmt.Errorf("error creating output file: %v", err))
		cancel()
		return
	}
	defer file.Close()

	if resp.AudioStream == nil {
		e.Logger().Error("error: audio stream is nil")
		cancel()
		return
	}

	if _, err = file.ReadFrom(resp.AudioStream); err != nil {
		e.Logger().Error(fmt.Errorf("failed to write audio stream to file: %v", err))
		cancel()
	}
}

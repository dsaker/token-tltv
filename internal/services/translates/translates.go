package translates

import (
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	"golang.org/x/text/language"
	"os"
	"sync"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/util"
	"time"
)

type Platform int

const (
	Google Platform = iota
	Amazon
)

type Translate struct {
	googleClients GoogleClients
	amazonClients AmazonClients
	m             models.ModelsX
	platform      Platform
}

func New(gc GoogleClients, ac AmazonClients, m models.ModelsX) *Translate {
	return &Translate{
		googleClients: gc,
		amazonClients: ac,
		m:             m,
	}
}

// TranslateX creates an interface for Translates
type TranslateX interface {
	CreateTTS(e echo.Context, title models.Title, voice models.Voice, basePath string) ([]models.Phrase, error)
	TranslatePhrases(e echo.Context, title models.Title, lang models.Language) ([]models.Phrase, error)
	DetectLanguage(context.Context, []string) (language.Tag, error)
}

// TranslatePhrases takes a slice of db.Translate{} and a db.Language and returns a slice
// of util.TranslatesReturn to be inserted into the db
func (t *Translate) TranslatePhrases(e echo.Context, title models.Title, lang models.Language) ([]models.Phrase, error) {
	//TODO translate as document instead of separate phrase

	// concurrently get all the responses from Google Translate
	var wg sync.WaitGroup
	responses := make([]models.Phrase, len(title.TitlePhrases)) // create string slice to hold all the responses
	// create context with cancel, so you can cancel all other requests after any error
	newCtx, cancel := context.WithCancel(context.Background())
	defer cancel() // Make sure it's called to release resources even if no errors

	// this is needed fro googleClients translate
	langTag, err := language.Parse(lang.Code)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}

	for i, nextTranslate := range title.TitlePhrases {
		// added intermittent sleep to fix TLS handshake errors on the client side
		if i%50 == 0 && i != 0 {
			time.Sleep(2 * time.Second)
		}
		wg.Add(1)
		// get responses concurrently with go routines depending on platform
		if t.platform == Google {
			go t.googleClients.GetTranslate(e, newCtx, cancel, nextTranslate, &wg, langTag, responses, i)
		} else {
			titleLang, err := t.m.GetLanguage(e.Request().Context(), title.TitleLang)
			if err != nil {
				e.Logger().Error(err)
				return nil, err
			}
			go t.amazonClients.GetTranslate(e, newCtx, cancel, nextTranslate, &wg, lang.Code, titleLang.Code, responses, i)
		}
	}
	wg.Wait()

	if newCtx.Err() != nil {
		e.Logger().Error(newCtx.Err())
		return nil, newCtx.Err()
	}

	return responses, nil
}

// CreateTTS is called from api.createAudioFromTitle.
// It checks if the mp3 audio files exist and if not it creates them.
func (t *Translate) CreateTTS(e echo.Context, title models.Title, voice models.Voice, basePath string) ([]models.Phrase, error) {
	voice, err := t.m.GetVoice(e.Request().Context(), voice.Name)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}

	lang, err := t.m.GetLanguage(e.Request().Context(), voice.Language)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}

	// if the audio files already exist, no need to request them again
	skip, err := util.PathExists(basePath)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}

	var translates []models.Phrase
	// if they do not exist, then request them
	if !skip {
		translates, err = t.CreateTranslates(e, title, lang)
		if err != nil {
			return nil, err
		}

		err = os.MkdirAll(basePath, 0777)
		if err != nil {
			e.Logger().Error(err)
			return nil, err
		}

		if err = t.TextToSpeech(e, translates, voice, basePath); err != nil {
			e.Logger().Error(err)
			return nil, err
		}
	}

	return translates, nil
}

// TextToSpeech takes a slice of db.Translate and get the speech mp3's adding them
// to the machines local file system
func (t *Translate) TextToSpeech(e echo.Context, ts []models.Phrase, voice models.Voice, bp string) error {
	// concurrently get all the audio content from Google text-to-speech
	var wg sync.WaitGroup
	// create context with cancel, so you can cancel all other requests after any error
	newCtx, cancel := context.WithCancel(context.Background())
	defer cancel() // Make sure it's called to release resources even if no errors

	// set the texttospeec params from the db voice sent in the request
	voiceSelectionParams := &texttospeechpb.VoiceSelectionParams{
		LanguageCode: voice.LanguageCode,
		SsmlGender:   texttospeechpb.SsmlVoiceGender_MALE,
		Name:         voice.Name,
	}
	if voice.SsmlGender == models.FEMALE {
		voiceSelectionParams.SsmlGender = texttospeechpb.SsmlVoiceGender_FEMALE
	}

	for i, nextText := range ts {
		// added intermittent sleep to fix TLS handshake errors on the client side
		if i%50 == 0 && i != 0 {
			time.Sleep(2 * time.Second)
		}
		wg.Add(1)
		//get responses concurrently with go routines depending on the platform
		if t.platform == Google {
			go t.googleClients.GetSpeech(e, newCtx, cancel, nextText, &wg, voiceSelectionParams, bp)
		} else {
			go t.amazonClients.GetSpeech(e, newCtx, cancel, nextText, voice, &wg, bp)
		}
	}
	wg.Wait()

	if newCtx.Err() != nil {
		e.Logger().Error(newCtx.Err())
		return newCtx.Err()
	}

	return nil
}

// CreateTranslates creates the translates in the language
func (t *Translate) CreateTranslates(e echo.Context, title models.Title, lang models.Language) ([]models.Phrase, error) {
	titleLang, err := t.m.GetLanguage(e.Request().Context(), title.TitleLang)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}

	// if the original language of file matches the language you desire translates for return original phrases
	if titleLang.Name == lang.Name {
		return title.TitlePhrases, nil
	}

	// create translates for title and to language and return
	translated, err := t.TranslatePhrases(e, title, lang)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}

	return translated, nil
}

// DetectLanguage uses Google's Natural Language API to detect the language of text
func (t *Translate) DetectLanguage(ctx context.Context, texts []string) (language.Tag, error) {
	detections, err := t.googleClients.gtc.DetectLanguage(ctx, texts)
	if err != nil {
		return language.Und, err
	}

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
	var maxCount int
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

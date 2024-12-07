package translates

import (
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"context"
	"errors"
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
}

func New(gc GoogleClients, ac AmazonClients) *Translate {
	return &Translate{
		googleClients: gc,
		amazonClients: ac,
	}
}

// TranslateX creates an interface for Translates
type TranslateX interface {
	CreateTTS(echo.Context, models.Title, int, string) ([]models.Phrase, error)
	TranslatePhrases(echo.Context, []models.Phrase, models.Language) ([]models.Phrase, error)
}

// TranslatePhrases takes a slice of db.Translate{} and a db.Language and returns a slice
// of util.TranslatesReturn to be inserted into the db
func (t *Translate) TranslatePhrases(e echo.Context, phrases []models.Phrase, lang models.Language) ([]models.Phrase, error) {
	// get language tag to translate to
	langTag, err := language.Parse(lang.Code)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}

	// concurrently get all the responses from Google Translate
	var wg sync.WaitGroup
	responses := make([]models.Phrase, len(phrases)) // create string slice to hold all the responses
	// create context with cancel, so you can cancel all other requests after any error
	newCtx, cancel := context.WithCancel(context.Background())
	defer cancel() // Make sure it's called to release resources even if no errors

	for i, nextTranslate := range phrases {
		// added intermittent sleep to fix TLS handshake errors on the client side
		if i%50 == 0 && i != 0 {
			time.Sleep(2 * time.Second)
		}
		wg.Add(1)
		//get responses concurrently with go routines
		go t.googleClients.GetTranslate(e, newCtx, cancel, nextTranslate, &wg, langTag, responses, i)
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
func (t *Translate) CreateTTS(e echo.Context, title models.Title, voiceId int, basePath string) ([]models.Phrase, error) {
	voice, ok := models.Voices[voiceId]
	if !ok {
		return nil, util.ErrVoiceIdInvalid
	}
	lang, ok := models.Languages[voice.LangId]
	if !ok {
		return nil, errors.New("invalid language id")
	}

	// if the audio files already exist no need to request them again
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
	// set the texttospeec params from the db voice sent in the request
	voiceSelectionParams := &texttospeechpb.VoiceSelectionParams{
		LanguageCode: voice.LanguageCodes[0],
		SsmlGender:   texttospeechpb.SsmlVoiceGender_MALE,
		Name:         voice.Name,
	}
	if voice.SsmlGender == "FEMALE" {
		voiceSelectionParams.SsmlGender = texttospeechpb.SsmlVoiceGender_FEMALE
	}
	// concurrently get all the audio content from Google text-to-speech
	var wg sync.WaitGroup
	// create context with cancel, so you can cancel all other requests after any error
	newCtx, cancel := context.WithCancel(context.Background())
	defer cancel() // Make sure it's called to release resources even if no errors

	for i, nextText := range ts {
		// added intermittent sleep to fix TLS handshake errors on the client side
		if i%50 == 0 && i != 0 {
			time.Sleep(2 * time.Second)
		}
		wg.Add(1)
		//get responses concurrently with go routines
		go t.googleClients.GetSpeech(e, newCtx, cancel, nextText, &wg, voiceSelectionParams, bp)
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
	// if the original language of file matches the language you desire translates for return original phrases
	if title.TitleLangId == lang.ID {
		return title.Phrases, nil
	}

	// create translates for title and to language and return
	translatesReturn, err := t.TranslatePhrases(e, title.Phrases, lang)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}

	return translatesReturn, nil
}

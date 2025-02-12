package models

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
)

// aws_languages => aws translate list-Languages > aws_languages.json
// aws_voices => aws polly describe-Voices > aws_voices.json
// google_languages => /scripts/python/supported_languages.json
// google_voices => /scripts/python/voices_api.json

var (
	ErrTooManyPhrases    = errors.New("too many phrases")
	ErrVoiceIdInvalid    = errors.New("voice id invalid")
	ErrPauseNotFound     = errors.New("audio pause file not found")
	ErrLanguageIdInvalid = errors.New("language id invalid")
	ErrPauseInvalid      = errors.New("pause out of range (must be between 3 and 10")
)

type Title struct {
	Name         string
	TitleLangId  int
	ToVoiceId    int
	FromVoiceId  int
	Pause        int
	TitlePhrases []Phrase
	ToPhrases    []Phrase
	Pattern      int
}

type Phrase struct {
	ID   int
	Text string
}

type Gender int

const (
	MALE Gender = iota + 1
	FEMALE
	NEUTRAL
)

type GoogleJsonVoice struct {
	LanguageCodes          []string `json:"language_codes"`
	SsmlGender             Gender   `json:"ssml_gender"`
	Name                   string   `json:"name"`
	NaturalSampleRateHertz int      `json:"natural_sample_rate_hertz"`
}

type GoogleJsonLanguage struct {
	Language string `json:"Tag"`
	Name     string `json:"Name"`
}

type AmazonLanguageArray struct {
	Languages []AmazonJsonLanguage
}
type AmazonJsonLanguage struct {
	LanguageCode string
	LanguageName string
}

type AmazonVoiceArray struct {
	Voices []AmazonJsonVoice
}

type AmazonJsonVoice struct {
	Gender           string
	Id               string
	LanguageCode     string
	LanguageName     string
	Name             string
	SupportedEngines []string
}

type Language struct {
	ID   int
	Code string
	Name string
}

type Voice struct {
	ID                     int
	LanguageCodes          []string
	Gender                 Gender
	VoiceName              string
	LanguageName           string
	NaturalSampleRateHertz int
	Engine                 string
	LangId                 int
}

type Status int

const (
	New Status = iota
	Used
)

type ModelsX interface {
	GetLanguage(int) (Language, error)
	GetVoice(int) (Voice, error)
	GetLanguages() map[int]Language
	GetVoices() map[int]Voice
}

type Models struct {
	Languages map[int]Language
	Voices    map[int]Voice
}

func (m *Models) GetLanguage(id int) (Language, error) {
	lang, ok := m.Languages[id]
	if !ok {
		return Language{}, ErrLanguageIdInvalid
	}
	return lang, nil
}

func (m *Models) GetVoice(id int) (Voice, error) {
	voice, ok := m.Voices[id]
	if !ok {
		return Voice{}, ErrVoiceIdInvalid
	}
	return voice, nil
}

func (m *Models) GetLanguages() map[int]Language {
	return m.Languages
}

func (m *Models) GetVoices() map[int]Voice {
	return m.Voices
}

func MakeGoogleMaps() (map[int]Language, map[int]Voice) {
	languageFile, err := JsonModels.Open("jsonmodels/google_languages.json")
	if err != nil {
		log.Fatal(err)
	}
	// Decode the JSON data into a struct
	var glangs []GoogleJsonLanguage
	decoder := json.NewDecoder(languageFile)
	err = decoder.Decode(&glangs)
	if err != nil {
		log.Fatal("Error decoding JSON:", err)
	}

	voiceFile, err := JsonModels.Open("jsonmodels/google_voices.json")
	if err != nil {
		log.Fatal(err)
	}
	// Decode the JSON data into a struct
	var voices []GoogleJsonVoice
	decoder = json.NewDecoder(voiceFile)
	err = decoder.Decode(&voices)
	if err != nil {
		log.Fatal("Error decoding JSON:", err)
	}

	usedLangs := make(map[int]bool)
	voiceMap := make(map[int]Voice)
	for i, voice := range voices {
		langCode := voice.LanguageCodes[0]
		// get the language id for the voice from the language tag
		langTag := strings.Split(langCode, "-")
		found := false
		langId := -1
		// find the language id (key) for the language that corresponds to the voice
		for j, lang := range glangs {
			// filipino voice langTag does not match language tag
			if langTag[0] == "fil" && lang.Language == "tl" {
				found = true
				langId = j
				break
			}
			// norwegian voice langTag does not match language tag
			if langTag[0] == "nb" && lang.Language == "no" {
				found = true
				langId = j
				break
			}
			if lang.Language == langTag[0] {
				found = true
				langId = j
				break
			}
		}
		if !found {
			//log.Println("langId not found for " + voice.Name + " : " + voice.LanguageCodes[0])
		} else {
			usedLangs[langId] = true
			// add to VoiceLangId map
			voiceMap[i] = Voice{
				ID:                     i,
				LanguageCodes:          voice.LanguageCodes,
				Gender:                 voice.SsmlGender,
				VoiceName:              voice.Name,
				NaturalSampleRateHertz: voice.NaturalSampleRateHertz,
				LangId:                 langId,
			}
		}
	}
	languageMap := make(map[int]Language)
	// only add google language to models.Language if it has a voice
	for i, lang := range glangs {
		_, ok := usedLangs[i]
		// If the key exists
		if ok {
			languageMap[i] = Language{
				ID:   i,
				Code: lang.Language,
				Name: lang.Name,
			}
		}
	}
	return languageMap, voiceMap
}

func MakeAmazonMaps() (map[int]Language, map[int]Voice) {
	languageFile, err := JsonModels.Open("jsonmodels/aws_languages.json")
	if err != nil {
		log.Fatal(err)
	}
	// Decode the JSON data into a struct
	var langArray AmazonLanguageArray
	decoder := json.NewDecoder(languageFile)
	err = decoder.Decode(&langArray)
	if err != nil {
		log.Fatal("Error decoding JSON:", err)
	}
	languages := langArray.Languages

	voiceFile, err := JsonModels.Open("jsonmodels/aws_voices.json")
	if err != nil {
		log.Fatal(err)
	}
	// Decode the JSON data into a struct
	var voices AmazonVoiceArray
	decoder = json.NewDecoder(voiceFile)
	err = decoder.Decode(&voices)
	if err != nil {
		log.Fatal("Error decoding JSON:", err)
	}

	usedLangs := make(map[int]bool)
	voiceMap := make(map[int]Voice)
	for i, voice := range voices.Voices {
		langCode := voice.LanguageCode
		// get the language id for the voice from the language tag
		langTag := strings.Split(langCode, "-")
		found := false
		langId := -1
		// find the language id (key) for the language that corresponds to the voice
		for j, lang := range languages {
			if lang.LanguageCode == langTag[0] {
				found = true
				langId = j
				break
			}
		}
		if !found {
			log.Println("langId not found for " + voice.Name + voice.LanguageCode)
		} else {
			usedLangs[langId] = true
			// add to VoiceLangId map
			var gender = MALE
			if voice.Gender == "Female" {
				gender = FEMALE
			}
			voiceMap[i] = Voice{
				ID:            i,
				LanguageCodes: []string{voice.LanguageCode},
				Gender:        gender,
				VoiceName:     voice.Id,
				LanguageName:  voice.LanguageName,
				LangId:        langId,
				Engine:        voice.SupportedEngines[0],
			}
		}
	}

	langaugeMap := make(map[int]Language)
	// only add the language to models.Language if it has a voice
	for i, lang := range languages {
		_, ok := usedLangs[i]
		// If the key exists
		if ok {
			langaugeMap[i] = Language{
				ID:   i,
				Code: lang.LanguageCode,
				Name: lang.LanguageName,
			}
		}
	}
	return langaugeMap, voiceMap
}

package models

import (
	"encoding/json"
	"log"
	"strings"
	"talkliketv.click/tltv/internal/util"
)

// aws_languages => aws translate list-languages > aws_languages.json
// aws_voices => aws polly describe-voices > aws_voices.json
// google_languages => /scripts/python/supported_languages.json
// google_voices => /scripts/python/voices_api.json

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

type GoogleJsonVoice struct {
	LanguageCodes          []string
	SsmlGender             string
	Name                   string
	NaturalSampleRateHertz int
}

type GoogleJsonLanguage struct {
	Language string
	Name     string
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
	Gender                 string
	VoiceName              string
	LanguageName           string
	NaturalSampleRateHertz int
	Engine                 string
	LangId                 int
}

var languages = make(map[int]Language)
var voices = make(map[int]Voice)

type ModelsX interface {
	GetLanguage(int) (Language, error)
	GetVoice(int) (Voice, error)
}

type Models struct{}

func (m *Models) GetLanguage(id int) (Language, error) {
	lang, ok := languages[id]
	if !ok {
		return Language{}, util.ErrLanguageIdInvalid
	}
	return lang, nil
}

func (m *Models) GetVoice(id int) (Voice, error) {
	voice, ok := voices[id]
	if !ok {
		return Voice{}, util.ErrVoiceIdInvalid
	}
	return voice, nil
}

func GetLanguagesLength() int {
	return len(languages)
}

func GetVoicesLength() int {
	return len(voices)
}

func MakeGoogleMaps() {
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
	// add each voice to the Languages map using key for the id
	for i, lang := range glangs {
		languages[i] = Language{
			ID:   i,
			Code: lang.Language,
			Name: lang.Name,
		}
	}

	voiceFile, err := JsonModels.Open("jsonmodels/google_voices.json")
	if err != nil {
		log.Fatal(err)
	}
	// Decode the JSON data into a struct
	var gvoices []GoogleJsonVoice
	decoder = json.NewDecoder(voiceFile)
	err = decoder.Decode(&gvoices)
	if err != nil {
		log.Fatal("Error decoding JSON:", err)
	}

	for i, voice := range gvoices {
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
			// add to VoiceLangId map
			voices[i] = Voice{
				ID:                     i,
				LanguageCodes:          voice.LanguageCodes,
				Gender:                 voice.SsmlGender,
				VoiceName:              voice.Name,
				NaturalSampleRateHertz: voice.NaturalSampleRateHertz,
				LangId:                 langId,
			}
		}
	}
}

func MakeAmazonMaps() {
	languageFile, err := JsonModels.Open("jsonmodels/aws_languages.json")
	if err != nil {
		log.Fatal(err)
	}
	// Decode the JSON data into a struct
	var array AmazonLanguageArray
	decoder := json.NewDecoder(languageFile)
	err = decoder.Decode(&array)
	if err != nil {
		log.Fatal("Error decoding JSON:", err)
	}
	// add each voice to the Languages map using key for the id
	for i, lang := range array.Languages {
		languages[i] = Language{
			ID:   i,
			Code: lang.LanguageCode,
			Name: lang.LanguageName,
		}
	}

	voiceFile, err := JsonModels.Open("jsonmodels/aws_voices.json")
	if err != nil {
		log.Fatal(err)
	}
	// Decode the JSON data into a struct
	var avoices AmazonVoiceArray
	decoder = json.NewDecoder(voiceFile)
	err = decoder.Decode(&avoices)
	if err != nil {
		log.Fatal("Error decoding JSON:", err)
	}

	for i, voice := range avoices.Voices {
		langCode := voice.LanguageCode
		// get the language id for the voice from the language tag
		langTag := strings.Split(langCode, "-")
		found := false
		langId := -1
		// find the language id (key) for the language that corresponds to the voice
		for j, lang := range array.Languages {
			if lang.LanguageCode == langTag[0] {
				found = true
				langId = j
				break
			}
		}
		if !found {
			log.Println("langId not found for " + voice.Name + voice.LanguageCode)
		} else {
			// add to VoiceLangId map
			voices[i] = Voice{
				ID:            i,
				LanguageCodes: []string{voice.LanguageCode},
				Gender:        voice.Gender,
				VoiceName:     voice.Name,
				LanguageName:  voice.LanguageName,
				LangId:        langId,
				Engine:        voice.SupportedEngines[0],
			}
		}
	}
}

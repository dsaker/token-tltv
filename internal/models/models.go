package models

import (
	"encoding/json"
	"github.com/labstack/echo/v4"
	"log"
	"strings"
)

type Title struct {
	Name        string
	TitleLangId int
	ToVoiceId   int
	FromVoiceId int
	Pause       int
	Phrases     []Phrase
	Translates  []Phrase
	Pattern     int
}

type Phrase struct {
	ID   int
	Text string
}

type JsonVoice struct {
	LanguageCodes          []string
	SsmlGender             string
	Name                   string
	NaturalSampleRateHertz int
}

type Voice struct {
	LanguageCodes          []string
	SsmlGender             string
	Name                   string
	NaturalSampleRateHertz int
	LangId                 int
}

type JsonLanguage struct {
	Language string
	Name     string
}

type Language struct {
	ID       int
	Language string
	Name     string
}

var Languages = make(map[int]Language)
var Voices = make(map[int]Voice)
var VoicesByLangId = make(map[int][]JsonVoice)

func MakeMaps(e *echo.Echo) {
	languageFile, err := JsonModels.Open("jsonmodels/google_languages.json")
	if err != nil {
		log.Fatal(err)
	}
	// Decode the JSON data into a struct
	var languages []JsonLanguage
	decoder := json.NewDecoder(languageFile)
	err = decoder.Decode(&languages)
	if err != nil {
		log.Fatal("Error decoding JSON:", err)
	}
	// add each voice to the Languages map using key for the id
	for i, lang := range languages {
		Languages[i] = Language{
			ID:       i,
			Language: lang.Language,
			Name:     lang.Name,
		}
	}

	voiceFile, err := JsonModels.Open("jsonmodels/google_voices.json")
	if err != nil {
		log.Fatal(err)
	}
	// Decode the JSON data into a struct
	var voices []JsonVoice
	decoder = json.NewDecoder(voiceFile)
	err = decoder.Decode(&voices)
	if err != nil {
		log.Fatal("Error decoding JSON:", err)
	}

	for i, voice := range voices {
		langCode := voice.LanguageCodes[0]
		// get the language id for the voice from the language tag
		langTag := strings.Split(langCode, "-")
		found := false
		langId := -1
		// find the language id (key) for the language that corresponds to the voice
		for j, lang := range languages {
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
			log.Println("langId not found for " + voice.Name + voice.LanguageCodes[0])
		} else {
			// add to VoiceLangId map
			Voices[i] = Voice{
				LanguageCodes:          voice.LanguageCodes,
				SsmlGender:             voice.SsmlGender,
				Name:                   voice.Name,
				NaturalSampleRateHertz: voice.NaturalSampleRateHertz,
				LangId:                 langId,
			}
		}
	}
}

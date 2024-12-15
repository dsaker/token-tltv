package models

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"strings"
	"talkliketv.click/tltv/internal/util"
	"time"
)

// aws_languages => aws translate list-Languages > aws_languages.json
// aws_voices => aws polly describe-Voices > aws_voices.json
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

type Token struct {
	Status  Status
	Created time.Time
	Hash    []byte
}

type TokenValue struct {
	Created time.Time
	Status  Status
}
type Status int

const (
	New Status = iota
	Used
)

var Languages = make(map[int]Language)
var Voices = make(map[int]Voice)
var tokens = make(map[string]TokenValue)

type ModelsX interface {
	GetLanguage(int) (Language, error)
	GetVoice(int) (Voice, error)
}

type Models struct{}

func (m *Models) GetLanguage(id int) (Language, error) {
	lang, ok := Languages[id]
	if !ok {
		return Language{}, util.ErrLanguageIdInvalid
	}
	return lang, nil
}

func (m *Models) GetVoice(id int) (Voice, error) {
	voice, ok := Voices[id]
	if !ok {
		return Voice{}, util.ErrVoiceIdInvalid
	}
	return voice, nil
}

func GetLanguagesLength() int {
	return len(Languages)
}

func GetVoicesLength() int {
	return len(Voices)
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
		Languages[i] = Language{
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
			Voices[i] = Voice{
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
		Languages[i] = Language{
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
			Voices[i] = Voice{
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

func LoadTokens(filePath string) {
	var tokenFile fs.File
	var err error
	if filePath == "" {
		tokenFile, err = JsonModels.Open("jsonmodels/tokens.json")
		if err != nil {
			log.Fatalf("add tokens.json file to /internal/models/jsonmodels/ : %s", err)
		}
	} else {
		tokenFile, err = os.Open(filePath)
		if err != nil {
			log.Fatalf("Error opening file to load tokens: %s", err)
			return
		}
	}
	// Decode the JSON data into a struct
	var array []Token
	decoder := json.NewDecoder(tokenFile)
	err = decoder.Decode(&array)
	if err != nil {
		log.Fatal("Error decoding JSON:", err)
	}
	// add each token to the tokens map
	for _, tok := range array {
		tokens[string(tok.Hash)] = TokenValue{
			Created: tok.Created,
			Status:  tok.Status,
		}
	}
}

func GetTokensLength() int {
	return len(tokens)
}

func CheckToken(token string) error {
	tokenHash := sha256.Sum256([]byte(token))
	tok, ok := tokens[string(tokenHash[:])]
	if !ok {
		return errors.New("token not found")
	}
	if tok.Status == Used {
		return errors.New("token already used")
	}
	return nil
}

func SetTokenStatus(token string, status Status) error {
	tokenHash := sha256.Sum256([]byte(token))
	tok, ok := tokens[string(tokenHash[:])]
	if !ok {
		return errors.New("something went wrong")
	}
	tok.Status = status
	tokens[string(tokenHash[:])] = tok
	return nil
}

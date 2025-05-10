package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/firestore"
	tts "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"cloud.google.com/go/translate"
	"golang.org/x/text/language"
	"talkliketv.click/tltv/internal/models"
)

// GoogleProvider implements VoiceProvider for Google Cloud TTS
type GoogleProvider struct {
	ttsClient       *tts.Client
	transClient     *translate.Client
	firestoreClient *firestore.Client
}

// NewGoogleProvider creates a new GoogleProvider instance with all necessary clients
func NewGoogleProvider(ctx context.Context, firestoreClient *firestore.Client) (*GoogleProvider, error) {
	ttsClient, err := tts.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create TTS client: %w", err)
	}

	transClient, err := translate.NewClient(ctx)
	if err != nil {
		ttsClient.Close()
		return nil, fmt.Errorf("failed to create translate client: %w", err)
	}

	return &GoogleProvider{
		ttsClient:       ttsClient,
		transClient:     transClient,
		firestoreClient: firestoreClient,
	}, nil
}

// Close closes all client connections
func (p *GoogleProvider) Close() {
	if p.ttsClient != nil {
		p.ttsClient.Close()
	}
	if p.transClient != nil {
		p.transClient.Close()
	}
}

// GetVoices retrieves all available voices and language information
func (p *GoogleProvider) GetVoices(ctx context.Context, outputDir string) ([]models.Voice, map[string]string, error) {
	// Get language data
	languageMap, err := p.getLanguageMap(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Get existing records from Firestore
	existingLanguages, existingVoices, existingLanguageCodes, err := p.getExistingRecords(ctx)
	if err != nil {
		return nil, nil, err
	}

	voicesToKeep := p.getFilteredVoices(ctx, outputDir)

	// Fetch and process Google voices
	googleVoices, voicesToAdd, languagesToAdd, languageCodesToAdd, err := p.processAndFilterVoices(ctx, languageMap, voicesToKeep, existingLanguages, existingVoices, existingLanguageCodes)
	if err != nil {
		return nil, nil, err
	}

	// Add new voices and languages to Firestore if needed
	if err := p.addNewRecordsToFirestore(ctx, voicesToAdd, languagesToAdd, languageCodesToAdd); err != nil {
		return nil, nil, err
	}

	return googleVoices, languageMap, nil
}

// CreateSampleMP3 creates a sample MP3 file for a given voice
func (p *GoogleProvider) CreateSampleMP3(ctx context.Context, voice models.Voice, langName string, outputDir string) error {
	return p.createSampleMP3(ctx, voice, langName, outputDir)
}

func (p *GoogleProvider) getFilteredVoices(ctx context.Context, outputDir string) map[string]models.Voice {
	resp, err := p.ttsClient.ListVoices(ctx, &texttospeechpb.ListVoicesRequest{})
	if err != nil {
		log.Fatalf("failed to list voices: %v", err)
	}

	// Group voices by language code
	voicesByLanguage := make(map[string][]models.Voice)
	for _, v := range resp.Voices {
		// Map Google SSML gender to our model
		gender := mapGender(v.SsmlGender)

		// Create a voice model
		voice := models.Voice{
			Name:                   v.Name,
			SsmlGender:             gender,
			NaturalSampleRateHertz: v.NaturalSampleRateHertz,
			Platform:               "google",
			SampleURL:              outputDir + v.Name + ".mp3",
		}

		// Each voice may support multiple language codes
		for _, langCode := range v.LanguageCodes {
			voice.LanguageCode = langCode
			voice.Language = strings.Split(langCode, "-")[0]

			// Add the voice to the appropriate language code list
			voicesByLanguage[langCode] = append(voicesByLanguage[langCode], voice)
		}
	}

	// Filter to keep only Chirp and Studio voices if we have enough of them
	for langCode, voices := range voicesByLanguage {
		filteredVoices := make([]models.Voice, 0, len(voices))
		for _, voice := range voices {
			if strings.Contains(voice.Name, "Chirp") || strings.Contains(voice.Name, "Studio") {
				filteredVoices = append(filteredVoices, voice)
			}
		}
		if len(filteredVoices) > 4 {
			voicesByLanguage[langCode] = filteredVoices
		}
	}

	// Log the count of voices per language code
	for langCode, voices := range voicesByLanguage {
		log.Printf("Language code %s has %d voices", langCode, len(voices))
	}

	// Create a set of all voice names that should be kept
	voicesToKeep := make(map[string]models.Voice)
	for _, voices := range voicesByLanguage {
		for _, voice := range voices {
			voicesToKeep[voice.Name] = voice
		}
	}

	// After filtering the voices, clean up MP3 files that aren't in the map
	p.cleanupUnusedMP3Files(outputDir, voicesToKeep)

	return voicesToKeep
}

func (p *GoogleProvider) cleanupUnusedMP3Files(outputDir string, voicesToKeep map[string]models.Voice) {

	// Get all MP3 files in the output directory
	files, err := os.ReadDir(outputDir)
	if err != nil {
		log.Printf("Failed to read output directory: %v", err)
		return
	}

	// Remove any MP3 files for voices not in our keep set
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".mp3") {
			voiceName := strings.TrimSuffix(file.Name(), ".mp3")
			if _, keep := voicesToKeep[voiceName]; !keep {
				filePath := filepath.Join(outputDir, file.Name())
				if err := os.Remove(filePath); err != nil {
					log.Printf("Failed to remove unused MP3 file %s: %v", filePath, err)
				} else {
					log.Printf("Removed unused MP3 file: %s", filePath)
				}
			}
		}
	}
}

// getLanguageMap gets supported languages and creates a lookup map
func (p *GoogleProvider) getLanguageMap(ctx context.Context) (map[string]string, error) {
	languages, err := p.transClient.SupportedLanguages(ctx, language.English)
	if err != nil {
		return nil, fmt.Errorf("error getting supported languages: %w", err)
	}

	// Create a map for quick lookup of name for language tag
	languageMap := map[string]string{}
	for _, lang := range languages {
		languageMap[lang.Tag.String()] = lang.Name
	}

	return languageMap, nil
}

// getExistingRecords retrieves existing languages and voices from Firestore
func (p *GoogleProvider) getExistingRecords(ctx context.Context) (map[string]bool, map[string]bool, map[string]bool, error) {
	// Get languages that are in firestore
	languageDocs, err := p.firestoreClient.Collection("languages").Where("platform", "==", "google").Documents(ctx).GetAll()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get languages from Firestore: %w", err)
	}

	// Create a map for quick lookup of existing language codes
	existingLanguages := map[string]bool{}
	for _, doc := range languageDocs {
		existingLanguages[doc.Ref.ID] = true
	}

	// Get voices that are in firestore
	voiceDocs, err := p.firestoreClient.Collection("voices").Where("platform", "==", "google").Documents(ctx).GetAll()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get voices from Firestore: %w", err)
	}

	// Create a map for quick lookup of existing voice names
	existingVoices := make(map[string]bool)
	for _, doc := range voiceDocs {
		existingVoices[doc.Ref.ID] = true
	}

	// Get languageCodes that are in firestore
	languageCodeDocs, err := p.firestoreClient.Collection("languageCodes").Where("platform", "==", "google").Documents(ctx).GetAll()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get voices from Firestore: %w", err)
	}

	// Create a map for quick lookup of existing language codes
	existingLanguageCodes := make(map[string]bool)
	for _, doc := range languageCodeDocs {
		existingLanguageCodes[doc.Ref.ID] = true
	}

	return existingLanguages, existingVoices, existingLanguageCodes, nil
}

// processGoogleVoices fetches and processes Google voices
func (p *GoogleProvider) processAndFilterVoices(ctx context.Context, languageMap map[string]string, voicesToKeep map[string]models.Voice,
	existingLanguages, existingVoices, existingLanguageCodes map[string]bool) ([]models.Voice, []models.Voice, []models.Language, []models.LanguageCode, error) {

	var googleVoices []models.Voice
	var voicesToAdd []models.Voice
	var languagesToAdd []models.Language
	var languageCodesToAdd []models.LanguageCode
	alreadyAddedLanguage := map[string]bool{}
	alreadyAddedLanguageCode := map[string]bool{}

	for _, v := range voicesToKeep {
		languageCode := v.LanguageCode
		languageId := strings.Split(languageCode, "-")[0]
		countryCode := strings.Split(languageCode, "-")[1]
		// Norwegian Bokmål is represented as "nb" in Google TTS, but we want to use "no"
		if languageId == "nb" {
			languageId = "no"
		}

		// Check if we need to add this language
		if _, exists := existingLanguages[languageId]; !exists {
			if _, exists = alreadyAddedLanguage[languageId]; !exists {
				langName, ok := languageMap[languageId]
				if !ok {
					log.Printf("Warning: Language %s not found for voice: %v", languageId, v.Name)
					continue
				}

				log.Printf("Language to add to firestore: %s", languageId)
				languagesToAdd = append(languagesToAdd, models.Language{
					Code:     languageId,
					Name:     langName,
					Platform: "google",
				})
				alreadyAddedLanguage[languageId] = true
			}
		}

		// Check if we need to add this language code
		if _, exists := existingLanguageCodes[languageCode]; !exists {
			if _, exists = alreadyAddedLanguageCode[languageCode]; !exists {
				langName, ok := languageMap[languageId]
				if !ok {
					log.Printf("Warning: Language %s not found for voice: %v", languageCode, v.Name)
					continue
				}

				log.Printf("LanguageCode to add to firestore: %s", languageCode)
				// Use the country name from the map
				country, ok := CountryNames[strings.ToUpper(countryCode)]
				if !ok {
					log.Fatalf("Warning: Country code %s not found for voice: %v", countryCode, v.Name)
				}
				languageCodesToAdd = append(languageCodesToAdd, models.LanguageCode{
					Code:     languageCode,
					Country:  country,
					Language: langName,
					Platform: "google",
				})
				alreadyAddedLanguageCode[languageCode] = true
			}
		}

		if _, exists := languageMap[languageId]; exists {
			voice := models.Voice{
				Name:                   v.Name,
				Language:               languageId,
				LanguageCode:           languageCode,
				SsmlGender:             v.SsmlGender,
				NaturalSampleRateHertz: v.NaturalSampleRateHertz,
				Platform:               "google",
				SampleURL:              "static/voices/google/" + v.Name + ".mp3",
			}

			googleVoices = append(googleVoices, voice)

			// Check if we need to add this voice
			if _, exists = existingVoices[v.Name]; !exists {
				log.Printf("Voice to add to firestore: %s", v.Name)
				voicesToAdd = append(voicesToAdd, voice)
			}
		} else {
			log.Printf("Warning: Language %s not found for voice: %v", languageId, v.Name)
		}
	}

	return googleVoices, voicesToAdd, languagesToAdd, languageCodesToAdd, nil
}

// ListVoicesInOutputDir returns a map of voice names that already have sample files in the output directory
func (p *GoogleProvider) ListVoicesInOutputDir(outputDir string) (map[string]bool, error) {
	existingMp3s := make(map[string]bool)

	// Check if directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return existingMp3s, fmt.Errorf("output directory does not exist: %s", outputDir)
	}

	// Read all files in the directory
	files, err := os.ReadDir(outputDir)
	if err != nil {
		return existingMp3s, fmt.Errorf("failed to read output directory: %w", err)
	}

	// Find all MP3 files and extract voice names
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".mp3") {
			// Remove .mp3 extension to get the voice name
			voiceName := strings.TrimSuffix(file.Name(), ".mp3")
			existingMp3s[voiceName] = true
		}
	}

	log.Printf("Found %d existing voice samples in %s", len(existingMp3s), outputDir)
	return existingMp3s, nil
}

// mapGender maps Google SSML gender to our model gender
func mapGender(ssmlGender texttospeechpb.SsmlVoiceGender) models.Gender {
	gender := models.MALE
	if ssmlGender == texttospeechpb.SsmlVoiceGender_NEUTRAL {
		gender = models.NEUTRAL
	}
	if ssmlGender == texttospeechpb.SsmlVoiceGender_FEMALE {
		gender = models.FEMALE
	}
	return gender
}

// addNewRecordsToFirestore adds new voices and languages to Firestore
func (p *GoogleProvider) addNewRecordsToFirestore(ctx context.Context, voicesToAdd []models.Voice, languagesToAdd []models.Language, languageCodesToAdd []models.LanguageCode) error {
	if len(voicesToAdd) > 0 {
		// Using the generic function for voices
		err := AddToFirestore(ctx, p.firestoreClient, "voices", voicesToAdd, func(v models.Voice) string {
			return v.Name
		})
		if err != nil {
			return fmt.Errorf("warning: failed to add voices to Firestore: %v", err)
		}
	}

	if len(languagesToAdd) > 0 {
		// Using the generic function for languages
		err := AddToFirestore(ctx, p.firestoreClient, "languages", languagesToAdd, func(l models.Language) string {
			return l.Code
		})
		if err != nil {
			return fmt.Errorf("warning: failed to add languages to Firestore: %v", err)
		}
	}

	if len(languageCodesToAdd) > 0 {
		err := AddToFirestore(ctx, p.firestoreClient, "languageCodes", languageCodesToAdd, func(lc models.LanguageCode) string {
			return lc.Code
		})
		if err != nil {
			return fmt.Errorf("warning: failed to add language codes to Firestore: %v", err)
		}
	}

	return nil
}

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

// processGoogleVoices fetches and processes Google voices
func (p *GoogleProvider) processAndFilterVoices(languageMap map[string]string, voicesToKeep map[string]models.Voice,
	existingLanguages, existingVoices map[string]bool) ([]models.Voice, []models.Voice, []models.Language, error) {

	var googleVoices []models.Voice
	var voicesToAdd []models.Voice
	var languagesToAdd []models.Language
	alreadyAddedLanguage := map[string]bool{}

	for _, v := range voicesToKeep {
		languageCode := v.LanguageCode
		languageId := strings.Split(languageCode, "-")[0]
		// Norwegian Bokmål is represented as "nb" in Google TTS, but we want to use "no"
		if languageId == "nb" {
			languageId = "no"
		}
		if languageId == "cmn" {
			if languageCode == "cmn-CN" {
				languageId = "zh-CN"
			} else if languageCode == "cmn-TW" {
				languageId = "zh-TW"
			}
		}

		// Check if we need to add this language
		if _, exists := existingLanguages[languageId]; !exists {
			if _, exists = alreadyAddedLanguage[languageId]; !exists {
				langName, ok := languageMap[languageId]
				if languageId == "zh-CN" {
					langName = "Chinese"
				}
				if languageId == "zh-TW" {
					langName = "Chinese"
				}
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

	return googleVoices, voicesToAdd, languagesToAdd, nil
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

// CreateSampleMP3 creates a sample MP3 file for the given voice
func (p *GoogleProvider) CreateSampleMP3(ctx context.Context, voice models.Voice, langName string, outputDir string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Check if file already exists
	outputFile := filepath.Join(outputDir, fmt.Sprintf("%s.mp3", voice.Name))
	if _, err := os.Stat(outputFile); err == nil {
		//log.Printf("MP3 file already exists for Google voice: %s", voice.Name)
		return nil
	}

	// Create sample text
	baseSampleText := "Hello, I am %s, a %s voice from Google Cloud Text-to-Speech. I hope you enjoy learning %s."
	sampleText := fmt.Sprintf(baseSampleText, voice.Name, langName, langName)

	// Create the synthesis input
	input := &texttospeechpb.SynthesisInput{
		InputSource: &texttospeechpb.SynthesisInput_Text{
			Text: sampleText,
		},
	}

	// Build the voice request
	voiceReq := &texttospeechpb.VoiceSelectionParams{
		LanguageCode: voice.LanguageCode,
		Name:         voice.Name,
		SsmlGender:   texttospeechpb.SsmlVoiceGender(voice.SsmlGender),
	}

	// Select the audio file type
	audioConfig := &texttospeechpb.AudioConfig{
		AudioEncoding: texttospeechpb.AudioEncoding_MP3,
	}

	// Perform the text-to-speech request
	resp, err := p.ttsClient.SynthesizeSpeech(ctx, &texttospeechpb.SynthesizeSpeechRequest{
		Input:       input,
		Voice:       voiceReq,
		AudioConfig: audioConfig,
	})
	if err != nil {
		return fmt.Errorf("failed to synthesize speech: %w", err)
	}

	// Write the response to the output file
	if err := os.WriteFile(outputFile, resp.AudioContent, 0644); err != nil {
		return fmt.Errorf("failed to write audio file: %w", err)
	}

	log.Printf("Created MP3 for Google voice: %s (%s)", voice.Name, voice.LanguageCode)
	return nil
}

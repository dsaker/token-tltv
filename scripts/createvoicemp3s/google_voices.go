package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"cloud.google.com/go/translate"
	"golang.org/x/text/language"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
	"talkliketv.click/tltv/internal/models"
)

// createSampleMP3 creates a sample MP3 file for a given voice
func (p *GoogleProvider) createSampleMP3(ctx context.Context, voice models.Voice, langName string, outputDir string) error {
	// Check if file already exists
	outputFile := filepath.Join(outputDir, fmt.Sprintf("%s.mp3", voice.Name))

	if _, err := os.Stat(outputFile); err == nil {
		log.Printf("MP3 file already exists for voice: %s", voice.Name)
		return nil
	}

	// Create a sample text
	baseSampleText := "This is a sample of the text to speech voice. I hope you enjoy learning %s!"
	sampleTextFormatted := fmt.Sprintf(baseSampleText, langName)

	// Translate the sample text
	translations, err := p.transClient.Translate(ctx, []string{sampleTextFormatted}, language.MustParse(voice.LanguageCode),
		&translate.Options{Format: translate.Text})

	sampleText := sampleTextFormatted
	if err == nil && len(translations) > 0 {
		sampleText = translations[0].Text
	}

	// Create request for speech synthesis
	req := &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{
				Text: sampleText,
			},
		},
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: voice.LanguageCode,
			Name:         voice.Name,
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_MP3,
		},
	}

	// Generate the speech
	response, err := p.ttsClient.SynthesizeSpeech(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to synthesize speech: %w", err)
	}

	// Write the audio content to file
	if err := os.WriteFile(outputFile, response.AudioContent, 0644); err != nil {
		return fmt.Errorf("failed to write audio file: %w", err)
	}

	log.Printf("Created MP3 for voice: %s (%s)", voice.Name, voice.LanguageCode)
	return nil
}

// getFilteredVoices retrieves and filters the list of available voices
func (p *GoogleProvider) getFilteredVoices(ctx context.Context, outputDir string) map[string]models.Voice {
	resp, err := p.ttsClient.ListVoices(ctx, &texttospeechpb.ListVoicesRequest{})
	if err != nil {
		log.Printf("Error listing voices: %v", err)
		return make(map[string]models.Voice)
	}

	voicesToKeep := make(map[string]models.Voice)
	for _, voice := range resp.Voices {
		// Skip WaveNet voices
		if voice.Name == "en-US-Wavenet-A" {
			continue
		}

		// Create voice model
		voiceModel := models.Voice{
			Name:         voice.Name,
			LanguageCode: voice.LanguageCodes[0],
			SsmlGender:   mapGender(voice.SsmlGender),
		}

		// Check if MP3 already exists
		outputFile := filepath.Join(outputDir, fmt.Sprintf("%s.mp3", voice.Name))
		if _, err := os.Stat(outputFile); err == nil {
			log.Printf("MP3 file already exists for voice: %s", voice.Name)
			continue
		}

		voicesToKeep[voice.Name] = voiceModel
	}

	return voicesToKeep
}

// processAndFilterVoices processes and filters the list of Google voices
func (p *GoogleProvider) processAndFilterVoices(ctx context.Context, languageMap map[string]string, voicesToKeep map[string]models.Voice,
	existingLanguages, existingVoices, existingLanguageCodes map[string]bool) ([]models.Voice, []models.Voice, []models.Language, []models.LanguageCode, error) {

	resp, err := p.ttsClient.ListVoices(ctx, &texttospeechpb.ListVoicesRequest{})
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to list voices: %w", err)
	}

	var googleVoices []models.Voice
	var voicesToAdd []models.Voice
	var languagesToAdd []models.Language
	var languageCodesToAdd []models.LanguageCode

	for _, voice := range resp.Voices {
		// Skip WaveNet voices
		if voice.Name == "en-US-Wavenet-A" {
			continue
		}

		// Create voice model
		voiceModel := models.Voice{
			Name:         voice.Name,
			LanguageCode: voice.LanguageCodes[0],
			SsmlGender:   mapGender(voice.SsmlGender),
		}

		// Add to voices to keep if not already processed
		if _, exists := voicesToKeep[voice.Name]; !exists {
			voicesToKeep[voice.Name] = voiceModel
		}

		// Check if voice already exists in Firestore
		if !existingVoices[voice.Name] {
			voicesToAdd = append(voicesToAdd, voiceModel)
		}

		// Process language
		langCode := voice.LanguageCodes[0]
		langName := languageMap[langCode]
		if langName == "" {
			langName = "Unknown"
		}

		// Add language if it doesn't exist
		if !existingLanguages[langName] {
			languagesToAdd = append(languagesToAdd, models.Language{
				Name: langName,
			})
		}

		// Add language code if it doesn't exist
		if !existingLanguageCodes[langCode] {
			languageCodesToAdd = append(languageCodesToAdd, models.LanguageCode{
				Code:     langCode,
				Language: langName,
			})
		}

		googleVoices = append(googleVoices, voiceModel)
	}

	return googleVoices, voicesToAdd, languagesToAdd, languageCodesToAdd, nil
}

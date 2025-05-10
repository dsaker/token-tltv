package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"talkliketv.click/tltv/internal/models"
)

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

	if err := p.updateLanguageCode(ctx, languageMap, voicesToKeep); err != nil {
		return nil, nil, err
	}

	// Fetch and process Google voices
	googleVoices, voicesToAdd, languagesToAdd, err := p.processAndFilterVoices(ctx, languageMap, voicesToKeep, existingLanguages, existingVoices, existingLanguageCodes)
	if err != nil {
		return nil, nil, err
	}

	// Add new voices and languages to Firestore if needed
	if err := p.addNewRecordsToFirestore(ctx, voicesToAdd, languagesToAdd); err != nil {
		return nil, nil, err
	}

	return googleVoices, languageMap, nil
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

package main

import (
	"cloud.google.com/go/firestore"
	tts "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"cloud.google.com/go/translate"
	"context"
	"fmt"
	"golang.org/x/text/language"
	"log"
	"os"
	"path/filepath"
	"strings"
	"talkliketv.click/tltv/internal/models"
)

// GoogleProvider implements VoiceProvider for Google Cloud TTS
type GoogleProvider struct {
	ttsClient       *tts.Client
	transClient     *translate.Client
	firestoreClient *firestore.Client
}

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

func (p *GoogleProvider) Close() {
	if p.ttsClient != nil {
		p.ttsClient.Close()
	}
	if p.transClient != nil {
		p.transClient.Close()
	}
}

func (p *GoogleProvider) GetVoices(ctx context.Context) ([]models.Voice, map[string]string, error) {
	// Get the list of all available voices
	resp, err := p.ttsClient.ListVoices(ctx, &texttospeechpb.ListVoicesRequest{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list voices: %w", err)
	}

	// Get supported languages
	languages, err := p.transClient.SupportedLanguages(ctx, language.English)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting supported languages: %w", err)
	}

	// create a map of quick lookup of name for language tag
	languageMap := map[string]string{}
	for _, lang := range languages {
		languageMap[lang.Tag.String()] = lang.Name
	}

	// Get languages that are in firestore
	languageDocs, err := p.firestoreClient.Collection("languages").Where("platform", "==", "google").Documents(ctx).GetAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get languages from Firestore: %w", err)
	}

	// Create a map for quick lookup of existing language codes
	existingLanguages := map[string]bool{}
	for _, doc := range languageDocs {
		existingLanguages[doc.Ref.ID] = true
	}

	// Get voices that are in firestore
	voiceDocs, err := p.firestoreClient.Collection("voices").Where("platform", "==", "google").Documents(ctx).GetAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get voices from Firestore: %w", err)
	}

	// Create a map for quick lookup of existing voice names
	existingVoices := make(map[string]bool)
	for _, doc := range voiceDocs {
		existingVoices[doc.Ref.ID] = true
	}

	var googleVoices []models.Voice
	var voicesToAdd []models.Voice
	var languagesToAdd []models.Language
	alreadyAdded := map[string]bool{}
	for _, v := range resp.Voices {
		languageCode := v.LanguageCodes[0]
		languageId := strings.Split(languageCode, "-")[0]
		if languageId == "nb" {
			languageId = "no"
		}
		// if language does not exist in firestore, add it
		if _, exists := existingLanguages[languageId]; !exists {
			if _, exists = alreadyAdded[languageId]; !exists {
				log.Printf("Language to add to firestore: %s", languageId)
				langName, ok := languageMap[languageId]
				if !ok {
					log.Printf("Warning: Language %s not found for voice: %v", languageId, v.Name)
					continue
				}
				languagesToAdd = append(languagesToAdd, models.Language{
					Code:     languageId,
					Name:     langName,
					Platform: "google",
				})
				alreadyAdded[languageId] = true
			}
		}

		gender := models.MALE
		if v.SsmlGender == texttospeechpb.SsmlVoiceGender_NEUTRAL {
			gender = models.NEUTRAL
		}
		if v.SsmlGender == texttospeechpb.SsmlVoiceGender_FEMALE {
			gender = models.FEMALE
		}
		if _, exists := languageMap[languageId]; exists {
			googleVoices = append(googleVoices, models.Voice{
				Name:                   v.Name,
				Language:               languageId,
				LanguageCode:           languageCode,
				SsmlGender:             gender,
				NaturalSampleRateHertz: v.NaturalSampleRateHertz,
				Platform:               "google",
			})
			// if voice does not exist in firestore, add it
			if _, exists = existingVoices[v.Name]; !exists {
				log.Printf("Voice to add to firestore: %s", v.Name)
				voicesToAdd = append(voicesToAdd, models.Voice{
					Name:                   v.Name,
					Language:               languageId,
					LanguageCode:           languageCode,
					SsmlGender:             gender,
					NaturalSampleRateHertz: v.NaturalSampleRateHertz,
					Platform:               "google",
				})
			}
		} else {
			log.Printf("Warning: Language %s not found for voice: %v", languageId, v.Name)
		}
	}

	// Add new voices and languages to Firestore if needed
	if len(voicesToAdd) > 0 {
		if err := AddVoicesToFirestore(ctx, p.firestoreClient, voicesToAdd); err != nil {
			return nil, nil, fmt.Errorf("warning: failed to add voices to Firestore: %v", err)
		}
	}

	if len(languagesToAdd) > 0 {
		if err := AddLanguagesToFirestore(ctx, p.firestoreClient, languagesToAdd); err != nil {
			return nil, nil, fmt.Errorf("warning: failed to add languages to Firestore: %v", err)
		}
	}

	return googleVoices, languageMap, nil
}

func (p *GoogleProvider) CreateSampleMP3(ctx context.Context, voice models.Voice, langName string, outputDir string) error {

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

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"talkliketv.com/tltv/internal/interfaces"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/aws/aws-sdk-go-v2/service/polly/types"
	"github.com/aws/aws-sdk-go-v2/service/translate"
)

// AmazonProvider implements VoiceProvider for Amazon Polly
type AmazonProvider struct {
	pollyClient     *polly.Client
	translateClient *translate.Client
	firestoreClient *firestore.Client
}

func NewAmazonProvider(ctx context.Context, firestoreClient *firestore.Client) (*AmazonProvider, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}

	pollyClient := polly.NewFromConfig(cfg)
	translateClient := translate.NewFromConfig(cfg)
	return &AmazonProvider{
		pollyClient:     pollyClient,
		translateClient: translateClient,
		firestoreClient: firestoreClient,
	}, nil
}

// Implement the GetVoices method for AmazonProvider
func (p *AmazonProvider) GetVoices(ctx context.Context, outputDir string) ([]interfaces.Voice, map[string]string, error) {
	// Get list of available voices
	resp, err := p.pollyClient.DescribeVoices(ctx, &polly.DescribeVoicesInput{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list Amazon Polly voices: %w", err)
	}

	amazonLangs, err := p.translateClient.ListLanguages(ctx, &translate.ListLanguagesInput{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list Amazon languages: %w", err)
	}

	// Create a map for quick lookup of existing language names
	langNameMap := make(map[string]string)
	for _, lang := range amazonLangs.Languages {
		if lang.LanguageName != nil && lang.LanguageCode != nil {
			langNameMap[*lang.LanguageCode] = *lang.LanguageName
		}
	}
	// Get languages that are in firestore
	languageDocs, err := p.firestoreClient.Collection("languages").Where("platform", "==", "amazon").Documents(ctx).GetAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get languages from Firestore: %w", err)
	}

	// Create a map for quick lookup of existing language codes
	existingLanguages := make(map[string]bool)
	for _, doc := range languageDocs {
		existingLanguages[doc.Ref.ID] = true
	}

	// Get voices that are in firestore
	voiceDocs, err := p.firestoreClient.Collection("voices").Where("platform", "==", "amazon").Documents(ctx).GetAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get voices from Firestore: %w", err)
	}

	// Create a map for quick lookup of existing voice names
	existingVoices := make(map[string]bool)
	for _, doc := range voiceDocs {
		existingVoices[doc.Ref.ID] = true
	}

	var amazonVoices []interfaces.Voice
	var voicesToAdd []interfaces.Voice
	var languagesToAdd []interfaces.Language
	alreadyAdded := map[string]bool{}
	// Process each voice
	for _, v := range resp.Voices {
		voiceName := v.Id
		languageCode := v.LanguageCode
		languageId := strings.Split(string(languageCode), "-")[0]

		// if language does not exist in firestore, add it
		if _, exists := existingLanguages[languageId]; !exists {
			if _, exists = alreadyAdded[languageId]; !exists {
				log.Printf("Language to add to firestore: %s", languageId)
				// Get the language name from the supported languages
				langName, ok := langNameMap[languageId]
				if !ok {
					log.Printf("Warning: Language %s not found for voice: %v", languageId, v.LanguageName)
					continue
				}
				languagesToAdd = append(languagesToAdd, interfaces.Language{
					Code:     languageId,
					Name:     langName,
					Platform: "amazon",
				})
				alreadyAdded[languageId] = true
			}
		}

		// Map Amazon gender to our model
		gender := interfaces.MALE
		if v.Gender == "Female" {
			gender = interfaces.FEMALE
		}

		// Create a voice struct
		voice := interfaces.Voice{
			Name:         *v.Name,
			Language:     languageId,
			LanguageCode: string(languageCode),
			SsmlGender:   gender,
			Platform:     "amazon",
		}

		// if voice does not exist in firestore, add it
		if _, exists := existingVoices[*v.Name]; !exists {
			log.Printf("Voice to add to firestore: %s", voiceName)
			voicesToAdd = append(voicesToAdd, voice)
		}
	}

	// Add new voices and languages to Firestore if needed
	if len(voicesToAdd) > 0 {
		err = AddToFirestore(ctx, p.firestoreClient, "voices", voicesToAdd, func(v interfaces.Voice) string {
			return v.Name
		})
		if err != nil {
			return nil, nil, fmt.Errorf("warning: failed to add voices to Firestore: %v", err)
		}
	}

	if len(languagesToAdd) > 0 {
		err = AddToFirestore(ctx, p.firestoreClient, "languages", languagesToAdd, func(v interfaces.Language) string {
			return v.Name
		})
		if err != nil {
			return nil, nil, fmt.Errorf("warning: failed to add languages to Firestore: %v", err)
		}
	}

	return amazonVoices, langNameMap, nil
}

// Implement the CreateSampleMP3 method for AmazonProvider
func (p *AmazonProvider) CreateSampleMP3(ctx context.Context, voice interfaces.Voice, langName string, outputDir string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Check if file already exists
	outputFile := filepath.Join(outputDir, fmt.Sprintf("%s.mp3", voice.Name))
	if _, err := os.Stat(outputFile); err == nil {
		log.Printf("MP3 file already exists for Amazon voice: %s", voice.Name)
		return nil
	}

	// Create sample text
	baseSampleText := "This is a sample of the Amazon Polly text to speech voice. I hope you enjoy learning %s!"
	sampleText := fmt.Sprintf(baseSampleText, langName)

	// Generate speech
	synthOutput, err := p.pollyClient.SynthesizeSpeech(ctx, &polly.SynthesizeSpeechInput{
		OutputFormat: types.OutputFormatMp3,
		Text:         aws.String(sampleText),
		VoiceId:      types.VoiceId(voice.Name),
	})
	if err != nil {
		return fmt.Errorf("failed to synthesize speech: %w", err)
	}

	// Read the audio stream
	audioBytes, err := io.ReadAll(synthOutput.AudioStream)
	if err != nil {
		return fmt.Errorf("failed to read audio stream: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputFile, audioBytes, 0600); err != nil {
		return fmt.Errorf("failed to write audio file: %w", err)
	}

	log.Printf("Created MP3 for Amazon voice: %s (%s)", voice.Name, voice.LanguageCode)
	return nil
}

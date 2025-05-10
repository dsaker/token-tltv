package main

import (
	"context"
	"fmt"
	"log"

	"talkliketv.click/tltv/internal/models"
)

// getExistingRecords retrieves existing records from Firestore
func (p *GoogleProvider) getExistingRecords(ctx context.Context) (map[string]bool, map[string]bool, map[string]bool, error) {
	// Get existing languages
	languagesRef := p.firestoreClient.Collection("languages")
	languages, err := languagesRef.Documents(ctx).GetAll()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get languages: %w", err)
	}

	existingLanguages := make(map[string]bool)
	for _, doc := range languages {
		var lang models.Language
		if err := doc.DataTo(&lang); err != nil {
			log.Printf("Error converting language document: %v", err)
			continue
		}
		existingLanguages[lang.Name] = true
	}

	// Get existing voices
	voicesRef := p.firestoreClient.Collection("voices")
	voices, err := voicesRef.Documents(ctx).GetAll()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get voices: %w", err)
	}

	existingVoices := make(map[string]bool)
	for _, doc := range voices {
		var voice models.Voice
		if err := doc.DataTo(&voice); err != nil {
			log.Printf("Error converting voice document: %v", err)
			continue
		}
		existingVoices[voice.Name] = true
	}

	// Get existing language codes
	languageCodesRef := p.firestoreClient.Collection("language_codes")
	languageCodes, err := languageCodesRef.Documents(ctx).GetAll()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get language codes: %w", err)
	}

	existingLanguageCodes := make(map[string]bool)
	for _, doc := range languageCodes {
		var langCode models.LanguageCode
		if err := doc.DataTo(&langCode); err != nil {
			log.Printf("Error converting language code document: %v", err)
			continue
		}
		existingLanguageCodes[langCode.Code] = true
	}

	return existingLanguages, existingVoices, existingLanguageCodes, nil
}

// addNewRecordsToFirestore adds new records to Firestore
func (p *GoogleProvider) addNewRecordsToFirestore(ctx context.Context, voicesToAdd []models.Voice, languagesToAdd []models.Language, languageCodesToAdd []models.LanguageCode) error {
	// Add new voices
	if len(voicesToAdd) > 0 {
		voicesRef := p.firestoreClient.Collection("voices")
		for _, voice := range voicesToAdd {
			_, err := voicesRef.Doc(voice.Name).Set(ctx, voice)
			if err != nil {
				return fmt.Errorf("failed to add voice %s: %w", voice.Name, err)
			}
		}
	}

	// Add new languages
	if len(languagesToAdd) > 0 {
		languagesRef := p.firestoreClient.Collection("languages")
		for _, lang := range languagesToAdd {
			_, err := languagesRef.Doc(lang.Name).Set(ctx, lang)
			if err != nil {
				return fmt.Errorf("failed to add language %s: %w", lang.Name, err)
			}
		}
	}

	// Add new language codes
	if len(languageCodesToAdd) > 0 {
		languageCodesRef := p.firestoreClient.Collection("language_codes")
		for _, langCode := range languageCodesToAdd {
			_, err := languageCodesRef.Doc(langCode.Code).Set(ctx, langCode)
			if err != nil {
				return fmt.Errorf("failed to add language code %s: %w", langCode.Code, err)
			}
		}
	}

	return nil
}

// getLanguageMap retrieves the language code to name mapping
func (p *GoogleProvider) getLanguageMap(ctx context.Context) (map[string]string, error) {
	languageCodesRef := p.firestoreClient.Collection("language_codes")
	docs, err := languageCodesRef.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get language codes: %w", err)
	}

	languageMap := make(map[string]string)
	for _, doc := range docs {
		var langCode models.LanguageCode
		if err := doc.DataTo(&langCode); err != nil {
			log.Printf("Error converting language code document: %v", err)
			continue
		}
		languageMap[langCode.Code] = langCode.Language
	}

	return languageMap, nil
}

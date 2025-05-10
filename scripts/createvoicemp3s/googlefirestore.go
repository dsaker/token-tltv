package main

import (
	"context"
	"fmt"

	"talkliketv.click/tltv/internal/models"
)

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

// addNewRecordsToFirestore adds new voices and languages to Firestore
func (p *GoogleProvider) addNewRecordsToFirestore(ctx context.Context, voicesToAdd []models.Voice, languagesToAdd []models.Language) error {
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

	return nil
}

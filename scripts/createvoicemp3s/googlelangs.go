package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/firestore"
	"golang.org/x/text/language"
	"talkliketv.click/tltv/internal/models"
)

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

func (p *GoogleProvider) updateLanguageCode(ctx context.Context, languageMap map[string]string, voicesToKeep map[string]models.Voice) error {
	languageCount := make(map[string]int)
	alreadyAddedLanguageCode := map[string]bool{}
	languageCodesToAdd := []models.LanguageCode{}

	// Get existing language codes from Firestore to check for name changes
	existingLangCodes := make(map[string]models.LanguageCode)
	docs, err := p.firestoreClient.Collection("languageCodes").Where("platform", "==", "google").Documents(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("failed to get existing language codes: %w", err)
	}
	for _, doc := range docs {
		var lc models.LanguageCode
		if err := doc.DataTo(&lc); err != nil {
			return fmt.Errorf("failed to parse language code: %w", err)
		}
		existingLangCodes[lc.Code] = lc
	}

	uniqueLanguageCodes := make(map[string]bool)
	for _, v := range voicesToKeep {
		uniqueLanguageCodes[v.LanguageCode] = true
	}

	// Count unique language codes per language
	for l := range uniqueLanguageCodes {
		languageId := strings.Split(l, "-")[0]
		languageCount[languageId]++
	}

	for langCode := range uniqueLanguageCodes {
		languageId := strings.Split(langCode, "-")[0]
		countryCode := strings.Split(langCode, "-")[1]
		if languageId == "nb" {
			languageId = "no"
		}
		// Check if we need to add or update this language code
		langName, ok := languageMap[languageId]
		if !ok {
			log.Printf("Warning: Language not found for language code: %v", langCode)
		}

		country, ok := CountryNames[strings.ToUpper(countryCode)]
		if !ok {
			return fmt.Errorf("Warning: Country code not found for language code: %v", langCode)
		}

		// Set name based on number of language codes for this language
		var name string
		if languageCount[languageId] > 1 {
			name = fmt.Sprintf("%s - %s", langName, country)
		} else {
			name = langName
		}

		// Check if this language code exists and if its name needs to be updated
		if existingLC, exists := existingLangCodes[langCode]; exists {
			if existingLC.Name != name {
				// Name has changed, update it in Firestore
				docRef := p.firestoreClient.Collection("languageCodes").Doc(langCode)
				_, err := docRef.Update(ctx, []firestore.Update{
					{Path: "name", Value: name},
				})
				if err != nil {
					log.Printf("Warning: Failed to update language code name for %s: %v", langCode, err)
				} else {
					log.Printf("Updated language code name for %s from '%s' to '%s'", langCode, existingLC.Name, name)
				}
			}
		} else if _, exists = existingLangCodes[langCode]; !exists {
			// New language code, add it
			if _, exists = alreadyAddedLanguageCode[langCode]; !exists {
				log.Printf("LanguageCode to add to firestore: %s", langCode)
				languageCodesToAdd = append(languageCodesToAdd, models.LanguageCode{
					Code:     langCode,
					Country:  country,
					Language: langName,
					Name:     name,
					Platform: "google",
				})
				alreadyAddedLanguageCode[langCode] = true
			}
		}
	}

	if len(languageCodesToAdd) > 0 {
		log.Printf("LanguageCodes to add to firestore: %v", languageCodesToAdd)
		err := AddToFirestore(ctx, p.firestoreClient, "languageCodes", languageCodesToAdd, func(lc models.LanguageCode) string {
			return lc.Code
		})
		if err != nil {
			return fmt.Errorf("failed to add language codes to firestore: %w", err)
		}
	}

	return nil
}

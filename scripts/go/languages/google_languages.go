package main

import (
	"cloud.google.com/go/translate"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/text/language"
	"io"
	"log"
	"os"
	"talkliketv.click/tltv/internal/models"
)

func listSupportedLanguages(w io.Writer, targetLanguage string) error {
	// targetLanguage := "th"
	ctx := context.Background()

	lang, err := language.Parse(targetLanguage)
	if err != nil {
		return fmt.Errorf("language.Parse: %w", err)
	}

	client, err := translate.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("translate.NewClient: %w", err)
	}
	defer client.Close()

	langs, err := client.SupportedLanguages(ctx, lang)
	if err != nil {
		return fmt.Errorf("SupportedLanguages: %w", err)
	}

	var googleLanguages []models.GoogleJsonLanguage
	for _, l := range langs {
		googleLanguages = append(googleLanguages, models.GoogleJsonLanguage{
			Language: l.Tag.String(),
			Name:     l.Name,
		})
	}

	j, _ := json.MarshalIndent(googleLanguages, "", "  ")
	_, err = os.Stdout.Write(j)
	if err != nil {
		return fmt.Errorf("os.Stdout.Write: %w", err)
	}
	//log.Println(string(j))

	return nil
}

func main() {
	err := listSupportedLanguages(os.Stdout, "en")
	if err != nil {
		log.Fatal(err)
		return
	}
}

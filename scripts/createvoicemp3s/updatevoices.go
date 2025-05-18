package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/firestore"
)

// AddToFirestore adds any slice of documents to the specified Firestore collection in a batch operation
func AddToFirestore[T any](ctx context.Context, client *firestore.Client, collection string, docs []T, idFunc func(T) string) error {
	if len(docs) == 0 {
		return nil
	}

	// Create a bulk writer
	bw := client.BulkWriter(ctx)

	// Process each document
	for _, doc := range docs {
		docID := idFunc(doc)
		docRef := client.Collection(collection).Doc(docID)
		_, err := bw.Set(docRef, doc)
		if err != nil {
			return fmt.Errorf("failed to set document %s: %w", docID, err)
		}
	}

	// Close the bulk writer to ensure all operations are completed
	bw.End()

	log.Printf("Added %d documents to %s collection", len(docs), collection)
	return nil
}

func main() {
	// Get project ID from environment variable if not specified
	defaultProject := os.Getenv("GOOGLE_CLOUD_PROJECT")

	projectID := flag.String("p", defaultProject, "Firebase project ID (optional for Firestore upload)")
	outputDir := flag.String("o", "../../ui/static/voices/google/", "Output directory for MP3 files")
	platform := flag.String("platform", "google", "Platform to generate MP3s for (google, amazon)")
	help := flag.Bool("h", false, "Show help")
	flag.Parse()

	if *help {
		fmt.Println("Creates MP3 samples for all available Text-to-Speech voices and uploads language and voice info to Firestore")
		fmt.Println("\nUsage:")
		fmt.Println("  go run updatevoices.go [flags]")
		fmt.Println("\nThe flags are:")
		fmt.Println("  -p string")
		fmt.Println("      Firebase project ID (defaults to GOOGLE_CLOUD_PROJECT env var)")
		fmt.Println("  -platform string")
		fmt.Println("      platform to update voices for (google, amazon) (default \"google\")")
		fmt.Println("  -o string")
		fmt.Println("      Output directory for MP3 files (default \"../../ui/static/voices/google/\")")
		fmt.Println("  -h")
		fmt.Println("      Show this help")
		return
	}

	ctx := context.Background()
	if *projectID == "" {
		log.Fatal("Project ID is required. Set GOOGLE_CLOUD_PROJECT environment variable or use -p flag.")
	}
	// Create client with all necessary providers
	client, err := NewClient(ctx, *projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	var provider VoiceProvider
	var outputDirectory string

	// Select provider based on flag
	if *platform == "amazon" {
		provider = client.amazonProvider
		outputDirectory = *outputDir
	} else {
		provider = client.googleProvider
		outputDirectory = *outputDir
	}
	// Check if the output directory exists
	if _, err = os.Stat(outputDirectory); os.IsNotExist(err) {
		log.Fatalf("output directory does not exist: %v", outputDir)
	}

	// Process voices with the selected provider
	voices, langMap, err := provider.GetVoices(ctx, outputDirectory)
	if err != nil {
		log.Fatalf("Failed to get voices: %v", err)
	}

	// Create MP3 samples for each voice
	for _, voice := range voices {
		langName, ok := langMap[voice.Language]
		if !ok {
			log.Printf("Language %s not found for voice %s", voice.Language, voice.Name)
			continue
		}
		if err = provider.CreateSampleMP3(ctx, voice, langName, outputDirectory); err != nil {
			log.Printf("Failed to create MP3 for voice %s: %v", voice.Name, err)
		}
	}

	log.Println("Voice MP3 creation completed successfully")
}

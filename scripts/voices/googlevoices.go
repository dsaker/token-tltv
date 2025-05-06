package main

import (
	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// ListVoices lists the available text to speech voices.
func ListVoices() error {
	ctx := context.Background()

	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	// Performs the list voices request.
	resp, err := client.ListVoices(ctx, &texttospeechpb.ListVoicesRequest{})
	if err != nil {
		return err
	}

	j, _ := json.MarshalIndent(resp.Voices, "", "  ")
	_, err = os.Stdout.Write(j)
	if err != nil {
		return fmt.Errorf("os.Stdout.Write: %w", err)
	}
	return nil
}

func main() {
	if err := ListVoices; err != nil {
		log.Fatal(err)
	}
}

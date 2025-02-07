package main

import (
	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

// ListVoices lists the available text to speech voices.
func ListVoices(w io.Writer) error {
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
	if err := ListVoices(os.Stdout); err != nil {
		log.Fatal(err)
	}
}

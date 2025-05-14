package main

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"talkliketv.click/tltv/internal/models"
)

// VoiceProvider is an interface for different TTS providers
type VoiceProvider interface {
	// GetVoices retrieves all available voices from the provider
	GetVoices(ctx context.Context, outputDir string) ([]models.Voice, map[string]string, error)

	// CreateSampleMP3 creates a sample MP3 for a given voice
	CreateSampleMP3(ctx context.Context, voice models.Voice, langName string, outputDir string) error
}

// Client holds all clients needed for the voice generation process
type Client struct {
	firestoreClient *firestore.Client
	googleProvider  *GoogleProvider
	amazonProvider  *AmazonProvider
}

// NewClient creates a new client with all necessary providers
func NewClient(ctx context.Context, projectID string) (*Client, error) {
	// Initialize Firebase
	conf := &firebase.Config{ProjectID: projectID}
	app, err := firebase.NewApp(ctx, conf)
	if err != nil {
		return nil, fmt.Errorf("failed to create Firebase app: %w", err)
	}

	// Get Firestore client
	firestoreClient, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client: %w", err)
	}

	// Create Google provider
	googleProvider, err := NewGoogleProvider(ctx, firestoreClient)
	if err != nil {
		firestoreClient.Close()
		return nil, err
	}

	// Create Amazon provider
	amazonProvider, err := NewAmazonProvider(ctx, firestoreClient)
	if err != nil {
		firestoreClient.Close()
		return nil, err
	}

	return &Client{
		firestoreClient: firestoreClient,
		googleProvider:  googleProvider,
		amazonProvider:  amazonProvider,
	}, nil
}

// Close closes all connections
func (c *Client) Close() {
	if c.firestoreClient != nil {
		c.firestoreClient.Close()
	}
	c.googleProvider.Close()
}

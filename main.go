//go:build go1.23

package main

import (
	"cloud.google.com/go/logging"
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"talkliketv.com/tltv/api"
	"talkliketv.com/tltv/internal/config"
	"talkliketv.com/tltv/internal/models"
	"talkliketv.com/tltv/internal/services/audiofile"
	"talkliketv.com/tltv/internal/services/translates"
	"talkliketv.com/tltv/internal/util"
)

func main() {
	// Load configuration
	var cfg config.Config
	if err := cfg.SetConfigs(); err != nil {
		log.Fatal(err)
	}
	flag.Parse()

	// Validate project ID
	if cfg.ProjectId == "" {
		cfg.ProjectId = os.Getenv("PROJECT_ID")
	}

	if cfg.ProjectId == "" {
		log.Fatal("Provide PROJECT_ID environment variable or use -p flag.")
	}

	ctx := context.Background()

	// Initialize logger for production
	var logger *logging.Logger
	if cfg.Env == "prod" {
		client, err := logging.NewClient(ctx, cfg.ProjectId)
		if err != nil {
			log.Fatalf("Failed to create logging client: %v", err)
		}
		defer client.Close()

		vmName, err := util.GetVMName()
		if err != nil {
			log.Println("Error getting VM name:", err)
			vmName = "tltv-logger"
		}
		logger = client.Logger(vmName)
		log.Println("Logger name:", vmName)
	}

	// Ensure ffmpeg is installed
	if output, err := exec.Command("ffmpeg", "-version").CombinedOutput(); err != nil || !strings.Contains(string(output), "ffmpeg version") {
		log.Fatalf("Ensure ffmpeg is installed and in PATH: %s", err)
	}

	// Initialize services
	af := audiofile.New(&audiofile.RealCmdRunner{})

	// Firestore client setup
	fClient, err := cfg.FirestoreClient()
	if err != nil {
		log.Fatal("Error creating Firestore client: ", err)
	}

	// Initialize Firestore models
	mods, err := models.NewModels(fClient, cfg.Env, models.LangCollString, models.LangCodeCollString, models.VoiceCollString, models.TokenCollString)
	if err != nil {
		log.Fatal("Error creating models: ", err)
	}

	// Create Google clients
	googleClients, err := translates.NewGoogleTTSClient(ctx)
	if err != nil {
		log.Fatal("Failed to create Google clients: ", err)
	}

	if googleClients == nil {
		log.Fatal("Failed to create Google clients")
	}

	// Create the translate service with the clients
	t := translates.New(googleClients, mods)

	// Create server with proper token store based on environment
	server := api.NewServer(cfg, t, af, mods)

	// Start server
	e := server.NewEcho(logger)
	log.Printf("\n\n%senvironment: %s\n%s", util.StarString, cfg.Env, util.StarString)
	e.Logger.Fatal(e.Start(net.JoinHostPort("0.0.0.0", cfg.Port)))
}

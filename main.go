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
	"talkliketv.click/tltv/api"
	"talkliketv.click/tltv/internal/config"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/services/audiofile"
	"talkliketv.click/tltv/internal/services/translates"
	"talkliketv.click/tltv/internal/util"
)

func main() {
	// Load configuration
	var cfg config.Config
	if err := cfg.SetConfigs(); err != nil {
		log.Fatal(err)
	}
	flag.Parse()

	// Validate project ID
	if cfg.Env == "dev" {
		if cfg.ProjectId == "" {
			cfg.ProjectId = os.Getenv("TEST_PROJECT_ID")
			if cfg.ProjectId == "" {
				log.Fatal("Provide PROJECT_ID in dev mode")
			}
		}
	} else if cfg.Env == "prod" && cfg.ProjectId == "" {
		cfg.ProjectId = os.Getenv("PROJECT_ID")
		if cfg.ProjectId == "" {
			log.Fatal("Provide PROJECT_ID in prod mode")
		}
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
		log.Fatal("Error creating Firestore client:", err)
	}

	// Initialize Firestore models
	mods := models.NewModels(fClient, "languages", "voices", "languageCodes")
	tokens := models.Tokens{Coll: fClient.Collection(util.TokenColl)}

	t := translates.New(
		*translates.NewGoogleClients(ctx),
		*translates.NewAmazonClients(ctx),
		mods,
	)

	// Create server with proper token store based on environment
	var server *api.Server
	if cfg.Env == "local" {
		server = api.NewServer(cfg, t, af, &models.LocalTokens{}, mods)
	} else {
		server = api.NewServer(cfg, t, af, &tokens, mods)
	}

	// Start server
	e := server.NewEcho(logger)
	log.Printf("\n\n%senvironment: %s\n%s", util.StarString, cfg.Env, util.StarString)
	e.Logger.Fatal(e.Start(net.JoinHostPort("0.0.0.0", cfg.Port)))
}

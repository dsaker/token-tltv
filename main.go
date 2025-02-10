//go:build go1.22

package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"talkliketv.click/tltv/api"
	"talkliketv.click/tltv/internal/audio/audiofile"
	"talkliketv.click/tltv/internal/config"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/translates"
	"talkliketv.click/tltv/internal/util"
)

func main() {
	var cfg config.Config
	err := cfg.SetConfigs()
	if err != nil {
		log.Fatal(err)
	}
	flag.Parse()

	if cfg.Env == "dev" {
		cfg.ProjectId = os.Getenv("TEST_PROJECT_ID")
	}

	if cfg.Env == "prod" {
		cfg.ProjectId = os.Getenv("PROJECT_ID")
	}

	if cfg.ProjectId == "" {
		log.Fatal("PROJECT_ID env var not set")
	}

	log.Print("PROJECT_ID: " + cfg.ProjectId)
	// if ffmpeg is not installed and in PATH of host machine fail immediately
	cmd := exec.Command("ffmpeg", "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Please make sure ffmep is installed and in PATH\n: %s", err)
	}
	if !strings.Contains(string(output), "ffmpeg version") {
		log.Fatalf("Please make sure ffmep is installed and in PATH\n: %s", string(output))
	}

	//initialize audiofile with the real command runner
	af := audiofile.New(&audiofile.RealCmdRunner{})

	// create translates with google or amazon clients depending on the flag set in conifg
	// I also set a global platform since this will not be changed during execution
	t := translates.New(*translates.NewGoogleClients(), translates.AmazonClients{}, &models.Models{})
	if translates.GlobalPlatform == translates.Amazon {
		t = translates.New(translates.GoogleClients{}, *translates.NewAmazonClients(), &models.Models{})
	}

	fClient, err := cfg.FirestoreClient()
	if err != nil {
		log.Fatal("Error creating firestore client: ", err)
	}

	tokensColl := fClient.Collection(util.TokenColl)
	tokens := models.Tokens{Coll: tokensColl}
	// create new server
	e := api.NewServer(cfg, t, af, &tokens)

	// running in local mode allows you to create audio without using tokens
	// this should never be used in the cloud
	if cfg.Env == "local" {
		localTokens := models.LocalTokens{}
		e = api.NewServer(cfg, t, af, &localTokens)
	}

	log.Printf("\n" + util.StarString + "environment: " + cfg.Env + "\n" + util.StarString)
	e.Logger.Fatal(e.Start(net.JoinHostPort("0.0.0.0", cfg.Port)))
}

func readSecretFromFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

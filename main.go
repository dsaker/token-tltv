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
)

func main() {
	var cfg config.Config
	err := cfg.SetConfigs()
	if err != nil {
		log.Fatal(err)
	}
	flag.Parse()

	if cfg.Env == "prod" {
		cfg.GcpProjectID = os.Getenv("PROJECT_ID")
		cfg.FirestoreTokenColl = os.Getenv("FIRESTORE_TOKENS")
		if cfg.FirestoreTokenColl == "" || cfg.GcpProjectID == "" {
			log.Fatal("missing Firestore Token collection or project id")
		}
	}
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

	tokensColl := fClient.Collection(cfg.FirestoreTokenColl)
	tokens := models.Tokens{Coll: tokensColl}
	// create new server
	e := api.NewServer(cfg, t, af, &tokens)

	if cfg.Env == "local" {
		localTokens := models.LocalTokens{}
		e = api.NewServer(cfg, t, af, &localTokens)
	}

	e.Logger.Fatal(e.Start(net.JoinHostPort("0.0.0.0", cfg.Port)))
}

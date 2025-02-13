//go:build go1.22

package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime/pprof"
	"strings"
	"talkliketv.click/tltv/api"
	"talkliketv.click/tltv/internal/audio/audiofile"
	"talkliketv.click/tltv/internal/config"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/translates"
	"talkliketv.click/tltv/internal/util"
)

func main() {
	// start cpu profiling
	f, _ := os.Create("cpu.prof")
	err := pprof.StartCPUProfile(f)
	if err != nil {
		log.Fatal(err)
	}
	defer pprof.StopCPUProfile()

	// load config
	var cfg config.Config
	err = cfg.SetConfigs()
	if err != nil {
		log.Fatal(err)
	}
	flag.Parse()

	if cfg.Env == "dev" && cfg.ProjectId == "" {
		cfg.ProjectId = os.Getenv("TEST_PROJECT_ID")
		if cfg.ProjectId == "" {
			log.Fatal("In dev mode you must provide PROJECT_ID as environment variable or command argument")
		}
	}

	if cfg.Env == "prod" {
		cfg.ProjectId = os.Getenv("PROJECT_ID")
		if cfg.ProjectId == "" {
			log.Fatal("In prod mode you must provide PROJECT_ID as environment variable")
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
	// create maps of voices and languages depending on platform
	langs, voices := models.MakeGoogleMaps()
	if cfg.Platform == "amazon" {
		langs, voices = models.MakeAmazonMaps()
	}

	mods := models.Models{Languages: langs, Voices: voices}
	t := translates.New(*translates.NewGoogleClients(), translates.AmazonClients{}, &mods, translates.Google)
	if cfg.Platform == "amazon" {
		t = translates.New(translates.GoogleClients{}, *translates.NewAmazonClients(), &mods, translates.Amazon)
	}

	fClient, err := cfg.FirestoreClient()
	if err != nil {
		log.Fatal("Error creating firestore client: ", err)
	}

	tokensColl := fClient.Collection(util.TokenColl)
	tokens := models.Tokens{Coll: tokensColl}
	// create new server
	e := api.NewServer(cfg, t, af, &tokens, &mods)

	// running in local mode allows you to create audio without using tokens
	// this should never be used in the cloud
	if cfg.Env == "local" {
		localTokens := models.LocalTokens{}
		e = api.NewServer(cfg, t, af, &localTokens, &mods)
	}

	log.Print("\n\n" + util.StarString + "environment: " + cfg.Env + "\n" + util.StarString)

	e.Logger.Fatal(e.Start(net.JoinHostPort("0.0.0.0", cfg.Port)))
}

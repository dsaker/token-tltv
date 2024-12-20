//go:build go1.22

package main

import (
	"flag"
	"log"
	"net"
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
	err := config.SetConfigs(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	flag.Parse()

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

	// create new server
	e := api.NewServer(cfg, t, af)

	e.Logger.Fatal(e.Start(net.JoinHostPort("0.0.0.0", cfg.Port)))
}

//go:build go1.22

package main

import (
	"flag"
	"log"
	"net"
	"os/exec"
	"strings"
	"talkliketv.click/tltv/api"
	"talkliketv.click/tltv/internal/config"
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

	t, af := api.CreateDependencies()

	// create new server
	e := api.NewServer(cfg, t, af)

	e.Logger.Fatal(e.Start(net.JoinHostPort("0.0.0.0", cfg.Port)))
}

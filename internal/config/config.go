package config

import (
	"errors"
	"flag"
	_ "github.com/lib/pq"
	"talkliketv.click/tltv/internal/translates"
)

// Config Update the config struct to hold the SMTP server settings.
type Config struct {
	Port            string
	Env             string
	PhrasePause     int
	AudioPattern    int
	MaxNumPhrases   int
	TTSBasePath     string
	FileUploadLimit int64
	Platform        translates.Platform
}

func SetConfigs(config *Config) error {
	// get port and debug from commandline flags... if not present use defaults
	flag.StringVar(&config.Port, "port", "8080", "API server port")

	flag.StringVar(&config.Env, "env", "development", "Environment (development|staging|cloud)")

	flag.StringVar(&config.TTSBasePath, "tts-base-path", "/tmp/audio/", "text-to-speech base path temporary storage of mp3 audio files")

	flag.Int64Var(&config.FileUploadLimit, "upload-size-limit", 8*8000, "File upload size limit in KB (default is 8)")
	flag.IntVar(&config.PhrasePause, "phrase-pause", 5, "Pause in seconds between phrases (must be between 3 and 10)'")
	flag.IntVar(&config.MaxNumPhrases, "maximum-number-phrases", 100, "Maximum number of phrases to be turned into audio files")
	flag.IntVar(&config.AudioPattern, "audio-pattern", 2, "Audio pattern to be used in constructing mp3's {1: standard, 2: advanced, 3: review}")

	if !isValidPause(config.PhrasePause) {
		return errors.New("invalid pause value (must be between 3 and 10)")
	}

	var platform string
	flag.StringVar(&platform, "platform", "google", "which platform you are using [google|amazon]")
	if platform == "google" {
		config.Platform = translates.Google
	} else if platform == "amazon" {
		config.Platform = translates.Amazon
	} else {
		return errors.New("invalid platform (must be google|amazon)")
	}

	return nil
}

func isValidPause(port int) bool {
	return port >= 3 && port <= 10
}

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
	MaxNumPhrases   int
	TTSBasePath     string
	FileUploadLimit int64
	TokenFilePath   string
}

func SetConfigs(config *Config) error {
	// get port and debug from commandline flags... if not present use defaults
	flag.StringVar(&config.Port, "port", "8080", "API server port")

	flag.StringVar(&config.Env, "env", "development", "Environment (development|staging|cloud)")

	flag.StringVar(&config.TTSBasePath, "tts-base-path", "/tmp/audio/", "text-to-speech base path temporary storage of mp3 audio files")

	flag.Int64Var(&config.FileUploadLimit, "upload-size-limit", 8*8000, "File upload size limit in KB (default is 8)")
	flag.IntVar(&config.MaxNumPhrases, "maximum-number-phrases", 100, "Maximum number of phrases to be turned into audio files")

	// set the global variable GlobalPlatform to google or amazon
	var platform string
	flag.StringVar(&platform, "platform", "google", "which platform you are using [google|amazon]")
	if platform == "google" {
		translates.GlobalPlatform = translates.Google
	} else if platform == "amazon" {
		translates.GlobalPlatform = translates.Amazon
	} else {
		return errors.New("invalid platform (must be google|amazon)")
	}

	// get the token file path to make a map of tokens required to make a valid request
	tokenUsageString := "file path where the tokens needed to make a valid request are stored (see scripts/go/generatecoins.go"
	flag.StringVar(&config.TokenFilePath, "token-file-path", "", tokenUsageString)

	return nil
}

func isValidPause(port int) bool {
	return port >= 3 && port <= 10
}

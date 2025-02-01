package config

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	firebase "firebase.google.com/go"
	"flag"
	_ "github.com/lib/pq"
	"talkliketv.click/tltv/internal/test"
	"talkliketv.click/tltv/internal/translates"
)

// Config Update the config struct to hold the SMTP server settings.
type Config struct {
	Port            string
	Env             string
	MaxNumPhrases   int
	TTSBasePath     string
	FileUploadLimit int64
	GcpProjectID    string
}

func (cfg *Config) SetConfigs() error {
	// get port and debug from commandline flags... if not present use defaults
	flag.StringVar(&cfg.Port, "port", "8080", "API server port")

	flag.StringVar(&cfg.Env, "env", "development", "Environment (development|staging|cloud)")

	flag.StringVar(&cfg.TTSBasePath, "tts-base-path", "/tmp/audio/", "text-to-speech base path temporary storage of mp3 audio files")

	flag.Int64Var(&cfg.FileUploadLimit, "upload-size-limit", 8*8000, "File upload size limit in KB (default is 8)")
	flag.IntVar(&cfg.MaxNumPhrases, "maximum-number-phrases", 100, "Maximum number of phrases to be turned into audio files")

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

	// google cloud project id
	flag.StringVar(&cfg.GcpProjectID, "gcp-project-id", test.TestProject, "project id for google cloud platform that contains firestore")

	return nil
}

func (cfg *Config) FirestoreClient() (*firestore.Client, error) {
	// Use the application default credentials
	ctx := context.Background()
	conf := &firebase.Config{ProjectID: cfg.GcpProjectID}
	app, err := firebase.NewApp(ctx, conf)
	if err != nil {
		return nil, err
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}
	return client, nil
}

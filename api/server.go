package api

import (
	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/translate"
	"context"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	runtimemw "github.com/go-openapi/runtime/middleware"
	echomw "github.com/labstack/echo/v4/middleware"
	middleware "github.com/oapi-codegen/echo-middleware"
	"golang.org/x/time/rate"
	"log"
	"os"
	"sync"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/oapi"

	"github.com/labstack/echo/v4"
	"talkliketv.click/tltv/internal/audio"
	"talkliketv.click/tltv/internal/audio/audiofile"
	"talkliketv.click/tltv/internal/config"
	"talkliketv.click/tltv/internal/translates"
	"talkliketv.click/tltv/internal/util"
)

type Server struct {
	sync.RWMutex
	translates translates.TranslateX
	config     config.Config
	af         audiofile.AudioFileX
}

// NewServer creates a new HTTP server and sets up routing.
func NewServer(e *echo.Echo, cfg config.Config, t translates.TranslateX, af audiofile.AudioFileX) *Server {
	// make sure silence mp3s exist in your base path
	initSilence(e, cfg)

	// create maps of voices and languages we will use instead of database
	models.MakeMaps(e)

	spec, err := oapi.GetSwagger()
	if err != nil {
		log.Fatalln("loading spec: %w", err)
	}

	g := e.Group("/swagger")
	g.GET("/doc.json", func(ctx echo.Context) error {
		if err = json.NewEncoder(ctx.Response().Writer).Encode(spec); err != nil {
			log.Fatalf("Error encoding swagger spec\n: %s\n", err)
		}
		return nil
	})

	swaggerHandler := runtimemw.SwaggerUI(runtimemw.SwaggerUIOpts{
		Path:    "/swagger/",
		SpecURL: "/swagger/doc.json",
	}, nil)

	g.GET("/", echo.WrapHandler(swaggerHandler))

	spec.Servers = openapi3.Servers{&openapi3.Server{URL: "/v1"}}

	apiGrp := e.Group("/v1")
	// add middleware
	apiGrp.Use(echomw.RateLimiter(echomw.NewRateLimiterMemoryStore(rate.Limit(20))))
	apiGrp.Use(echomw.Logger())
	apiGrp.Use(echomw.Recover())

	// Use our validation middleware to check all requests against the
	// OpenAPI schema.
	apiGrp.Use(middleware.OapiRequestValidator(spec))

	srv := &Server{
		translates: t,
		config:     cfg,
		af:         af,
	}
	oapi.RegisterHandlersWithBaseURL(apiGrp, srv, "")
	return srv
}

// Make sure we conform to ServerInterface
var _ oapi.ServerInterface = (*Server)(nil)

// initSilence copies the silence mp3's from the embedded filesystem to the config TTSBasePath
func initSilence(e *echo.Echo, cfg config.Config) {
	// check if silence mp3s exist in your base path
	silencePath := cfg.TTSBasePath + audiofile.AudioPauseFilePath[cfg.PhrasePause]
	exists, err := util.PathExists(silencePath)
	if err != nil {
		log.Fatal(err)
	}
	// if it doesn't exist copy it from embedded FS to TTSBasePath
	if !exists {
		err = os.MkdirAll(cfg.TTSBasePath+"silence/", 0777)
		if err != nil {
			log.Fatal(err)
		}
		for key, value := range audiofile.AudioPauseFilePath {
			fmt.Printf("%d", key)
			pause, err := audio.Silence.ReadFile(value)
			if err != nil {
				log.Fatal(err)
			}
			// Create a new file
			file, err := os.Create(cfg.TTSBasePath + value)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()
			// Write to the file
			_, err = file.Write(pause)
			if err != nil {
				log.Fatal(err)
			}
			// Ensure data is written to disk
			err = file.Sync()
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

// CreateDependencies creates a new google translate and text-to-speech clients; constructs
// the translates and audiofile dependencies and returns them
func CreateDependencies(e *echo.Echo) (*translates.Translate, *audiofile.AudioFile) {
	// create google translate and text-to-speech clients
	ctx := context.Background()
	transClient, err := translate.NewClient(ctx)
	if err != nil {
		log.Fatalf("Error creating google api translate client\n: %s", err)
	}
	ttsClient, err := texttospeech.NewClient(ctx)
	if err != nil {
		log.Fatalf("Error creating google api translate client\n: %s", err)
	}
	t := translates.New(transClient, ttsClient)

	//initialize audiofile with the real command runner
	af := audiofile.New(&audiofile.RealCmdRunner{})

	return t, af
}

package api

import (
	"cloud.google.com/go/logging"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	oapimw "github.com/oapi-codegen/echo-middleware"
	"golang.org/x/time/rate"
	"io/fs"
	"log"
	"os"
	"strconv"
	"sync"
	"talkliketv.com/tltv/internal/config"
	"talkliketv.com/tltv/internal/interfaces"
	"talkliketv.com/tltv/internal/oapi"
	"talkliketv.com/tltv/internal/services"
	"talkliketv.com/tltv/internal/services/audiofile"
	"talkliketv.com/tltv/internal/services/templates"
	"talkliketv.com/tltv/internal/services/translates"
	"talkliketv.com/tltv/internal/util"
	"talkliketv.com/tltv/ui"
	"time"
)

type Server struct {
	sync.RWMutex
	translate translates.TranslateX
	af        audiofile.AudioFileX
	m         interfaces.ModelsStore
	config    config.Config
}

func NewServer(
	c config.Config,
	t translates.TranslateX,
	af audiofile.AudioFileX,
	m interfaces.ModelsStore,
) *Server {
	return &Server{
		translate: t,
		config:    c,
		af:        af,
		m:         m,
	}
}

// NewEcho creates a new echo server
func (s *Server) NewEcho(logger *logging.Logger) *echo.Echo {
	e := echo.New()
	// make sure silence mp3s exist in your base path
	initSilence(s.config)

	if s.config.Env == "prod" {
		if logger == nil {
			log.Fatal("logger is nil")
		}
		// Middleware to send logs to Google Cloud Logging
		e.Use(GoogleCloudLoggingMiddleWare(logger))
	}

	// add middleware
	e.Use(echomw.Logger())
	e.Use(echomw.RateLimiter(echomw.NewRateLimiterMemoryStore(rate.Limit(10))))
	e.Use(echomw.Recover())

	// Create a new template cache
	tempC, err := templates.NewTemplateCache()
	if err != nil {
		log.Fatal(err)
	}
	e.Renderer = &templates.TemplateRegistry{Templates: tempC}

	// Use our validation middleware to check all requests against the OpenAPI schema.
	apiGrp := e.Group("/v1")
	spec, err := oapi.GetSwagger()
	if err != nil {
		log.Fatalln("loading spec: %w", err)
	}
	spec.Servers = openapi3.Servers{&openapi3.Server{URL: "/v1"}}
	apiGrp.Use(oapimw.OapiRequestValidatorWithOptions(spec,
		&oapimw.Options{
			SilenceServersWarning: true,
		}))

	uiGrp := e.Group("")
	// Serve static files from the "static" directory
	staticFiles, err := fs.Sub(ui.Files, "static")
	if err != nil {
		log.Fatal(err)
	}
	uiGrp.StaticFS("/static", staticFiles)
	uiGrp.GET("/", templates.HomeView)
	uiGrp.GET("/ads.txt", templates.AdsView)
	uiGrp.GET("/robots.txt", templates.RobotsView)
	uiGrp.GET("/favicon.ico", templates.FaviconView)
	uiGrp.GET("/audio", templates.AudioView(s.m))
	uiGrp.GET("/parse", templates.ParseView(s.config.MaxNumPhrases))

	oapi.RegisterHandlersWithBaseURL(apiGrp, s, "")
	return e
}

func GoogleCloudLoggingMiddleWare(logger *logging.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c) // Process the request.
			stop := time.Now()
			latency := stop.Sub(start)

			message := ""
			if err != nil {
				message = "error: " + err.Error()
			}
			req := c.Request()
			res := c.Response()

			// Create labels and payload maps
			labels := map[string]string{
				"method":     req.Method,
				"uri":        req.RequestURI,
				"status":     strconv.Itoa(res.Status),
				"latency":    latency.String(),
				"user_agent": req.UserAgent(),
				"message":    message,
				"real_ip":    c.RealIP(),
			}

			payload := map[string]any{
				"method":     req.Method,
				"uri":        req.RequestURI,
				"status":     res.Status,
				"latency":    latency.String(),
				"user_agent": req.UserAgent(),
				"message":    message,
				"real_ip":    c.RealIP(),
			}

			// Check if token exists in form values and add masked version to logs
			if token := c.FormValue("token"); token != "" {
				maskedToken := maskToken(token)
				labels["token"] = maskedToken
				payload["token"] = maskedToken
			}

			severity := logging.Info
			if res.Status >= 400 {
				severity = logging.Warning
			}
			if res.Status >= 500 {
				severity = logging.Error
			}

			logger.Log(logging.Entry{
				Labels:   labels,
				Severity: severity,
				Payload:  payload,
			})
			return err
		}
	}
}

// maskToken returns a masked version of the token for logging purposes
// Tokens are always 26 characters long
func maskToken(token string) string {
	// Show first 4 and last 4 characters, mask the middle with asterisks
	if len(token) != 26 {
		return "invalid-token-length"
	}

	// Show first 4 and last 4 characters
	return token[:4] + "******************" + token[22:]
}

// Make sure we conform to ServerInterface
var _ oapi.ServerInterface = (*Server)(nil)

// initSilence copies the silence mp3's from the embedded filesystem to the config TTSBasePath
func initSilence(cfg config.Config) {
	exists := true
	var err error
	// check if silence mp3s exist in your base path
	for key := range audiofile.AudioPauseFilePath {
		silencePath := cfg.TTSBasePath + audiofile.AudioPauseFilePath[key]
		exists, err = util.PathExists(silencePath)
		if err != nil {
			log.Fatal(err)
		}
		if !exists {
			break
		}
	}

	// if it doesn't exist copy it from embedded FS to TTSBasePath
	if !exists {
		err = os.MkdirAll(cfg.TTSBasePath+"silence/", 0777)
		if err != nil {
			log.Fatal(err)
		}
		for key, value := range audiofile.AudioPauseFilePath {
			fmt.Printf("%d", key)
			pause, err := services.Silence.ReadFile(value)
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

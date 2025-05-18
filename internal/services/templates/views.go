package templates

import (
	"io/fs"
	"net/http"
	"talkliketv.com/tltv/internal/interfaces"
	"talkliketv.com/tltv/ui"

	"github.com/labstack/echo/v4"
)

func HomeView(e echo.Context) error {
	return e.Render(http.StatusOK, "home.gohtml", nil)
}

// AudioView returns a handler function that renders the frontend html page to upload a file for mp3 creation
func AudioView(m interfaces.ModelsStore) echo.HandlerFunc {
	return func(e echo.Context) error {
		languageCodes, err := m.GetLanguageCodes(e.Request().Context())
		if err != nil {
			return e.String(http.StatusInternalServerError, "error getting language codes: "+err.Error())
		}
		voices, err := m.GetVoices(e.Request().Context())
		if err != nil {
			return e.String(http.StatusInternalServerError, "error getting voices: "+err.Error())
		}
		return e.Render(http.StatusOK, "audio.gohtml", newTemplateData(languageCodes, voices, ""))
	}
}

// ParseView renders the frontend html page to upload a file to parsefile it
func ParseView(maxPhrases int) echo.HandlerFunc {
	return func(e echo.Context) error {
		return e.Render(http.StatusOK, "parse.gohtml", map[string]interface{}{
			"MaxPhrases": maxPhrases,
		})
	}
}

// AdsView renders the ads.txt file
func AdsView(e echo.Context) error {
	return e.String(http.StatusOK, "placeholder.example.com, placeholder, DIRECT, placeholder")
}

// RobotsView renders the robots.txt file
func RobotsView(e echo.Context) error {
	return e.String(http.StatusOK, "User-agent: *\nDisallow:")
}

// FaviconView renders the favicon.ico file
func FaviconView(e echo.Context) error {
	// Serve static files from the "static" directory
	staticFiles, err := fs.Sub(ui.Files, "static")

	if err != nil {
		return e.String(http.StatusNotFound, "favicon.ico not found")
	}
	file, err := staticFiles.Open("favicon.ico")
	if err != nil {
		return e.String(http.StatusNotFound, "favicon.ico not found")
	}
	return e.Stream(http.StatusOK, "image/x-icon", file)
}

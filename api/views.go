package api

import (
	"github.com/labstack/echo/v4"
	"io/fs"
	"net/http"
	"talkliketv.click/tltv/ui"
)

func homeView(e echo.Context) error {
	return e.Render(http.StatusOK, "home.gohtml", nil)
}

// audioView renders the frontend html page to upload a file for mp3 creation
func (s *Server) audioView(e echo.Context) error {
	languages, err := s.m.GetLanguages(e.Request().Context())
	if err != nil {
		return e.String(http.StatusInternalServerError, "error getting languages: "+err.Error())
	}
	voices, err := s.m.GetVoices(e.Request().Context())
	if err != nil {
		return e.String(http.StatusInternalServerError, "error getting voices: "+err.Error())
	}
	return e.Render(http.StatusOK, "audio.gohtml", newTemplateData(languages, voices, ""))
}

// parseView renders the frontend html page to upload a file to parsefile it
func (s *Server) parseView(e echo.Context) error {
	return e.Render(http.StatusOK, "parse.gohtml", map[string]interface{}{
		"MaxPhrases": s.config.MaxNumPhrases,
	})
}

// adsView renders the ads.txt file
func adsView(e echo.Context) error {
	return e.String(http.StatusOK, "placeholder.example.com, placeholder, DIRECT, placeholder")
}

// robotsView renders the robots.txt file
func robotsView(e echo.Context) error {
	return e.String(http.StatusOK, "User-agent: *\nDisallow:")
}

// faviconView renders the favicon.ico file
func faviconView(e echo.Context) error {
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

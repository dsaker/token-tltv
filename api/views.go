package api

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func homeView(e echo.Context) error {
	return e.Render(http.StatusOK, "home.gohtml", nil)
}

// audioView renders the frontend html page to upload a file for mp3 creation
func audioView(e echo.Context) error {
	return e.Render(http.StatusOK, "audio.gohtml", newTemplateData(""))
}

// parseView renders the frontend html page to upload a file to parse it
func (s *Server) parseView(e echo.Context) error {
	return e.Render(http.StatusOK, "parse.gohtml", map[string]interface{}{
		"MaxPhrases": s.config.MaxNumPhrases,
	})
}

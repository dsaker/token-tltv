package api

import (
	"errors"
	"github.com/labstack/echo/v4"
	"html/template"
	"io"
	"io/fs"
	"path/filepath"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/ui"
)

type templateData struct {
	LanguageCodes  map[string]models.LanguageCode
	Voices         map[string]models.Voice
	Error          string
	PauseDurations []int
}

type TemplateRegistry struct {
	templates map[string]*template.Template
}

// Render Implement e.Renderer interface
func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, ok := t.templates[name]
	if !ok {
		err := errors.New("Template not found -> " + name)
		return err
	}
	return tmpl.ExecuteTemplate(w, "base", data)
}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	// Use fs.Glob() to get a slice of all filepaths in the ui.Files embedded
	// filesystem which match the pattern 'html/pages/*.tmpl'. This essentially
	// gives us a slice of all the 'page' templates for the application.
	pages, err := fs.Glob(ui.Files, "html/pages/*.gohtml")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		// Create a slice containing the filepath patterns for the templates we
		// want to parsefile.
		patterns := []string{
			"html/base.gohtml",
			"html/common/*.gohtml",
			page,
		}

		// Use ParseFS() instead of ParseFiles() to parsefile the template files
		// from the ui.Files embedded filesystem.
		ts, err := template.New(name).ParseFS(ui.Files, patterns...)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}

// newTemplateDatachecks if the user is authenticated and adds the base data needed for the templates
func newTemplateData(l map[string]models.LanguageCode, v map[string]models.Voice, err string) *templateData {
	return &templateData{
		PauseDurations: []int{3, 4, 5, 6, 7, 8, 9, 10},
		LanguageCodes:  l,
		Voices:         v,
		Error:          err,
	}
}

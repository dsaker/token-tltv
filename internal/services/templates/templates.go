package templates

import (
	"errors"
	"html/template"
	"io"
	"io/fs"
	"path/filepath"
	"talkliketv.com/tltv/internal/interfaces"
	"talkliketv.com/tltv/ui"

	"github.com/labstack/echo/v4"
)

type templateData struct {
	LanguageCodes  []interfaces.LanguageCode
	Voices         []interfaces.Voice
	Error          string
	PauseDurations []int
}

type TemplateRegistry struct {
	Templates map[string]*template.Template
}

// Render Implement e.Renderer interface
func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, ok := t.Templates[name]
	if !ok {
		err := errors.New("Template not found -> " + name)
		return err
	}
	return tmpl.ExecuteTemplate(w, "base", data)
}

func NewTemplateCache() (map[string]*template.Template, error) {
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
func newTemplateData(l []interfaces.LanguageCode, v []interfaces.Voice, err string) *templateData {
	return &templateData{
		PauseDurations: []int{3, 4, 5, 6, 7, 8, 9, 10},
		LanguageCodes:  l,
		Voices:         v,
		Error:          err,
	}
}

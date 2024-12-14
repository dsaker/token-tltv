package api

import (
	"github.com/labstack/echo/v4"
	"html/template"
	"io"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/ui"
)

type TemplateData struct {
	Form      any
	Flash     string
	Languages []*models.Language
	Voices    []*models.Voice
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func NewTemplates() *Template {

	templates := template.Must(template.New("").ParseFS(ui.Files, "html/pages/*.gohtml", "html/common/*.gohtml"))
	return &Template{
		templates: templates,
	}
}

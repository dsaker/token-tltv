package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/util"
	"talkliketv.click/tltv/ui"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTemplateCache(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}
	// Test creation of template cache
	cache, err := newTemplateCache()
	require.NoError(t, err)
	require.NotNil(t, cache)

	// Verify expected templates are loaded
	expectedTemplates := []string{
		"home.gohtml",
		"audio.gohtml",
		"parse.gohtml",
	}

	for _, tmplName := range expectedTemplates {
		t.Run("Template_"+tmplName, func(t *testing.T) {
			tmpl, found := cache[tmplName]
			require.True(t, found, "Template %s should exist in cache", tmplName)
			require.NotNil(t, tmpl)
		})
	}
}

func TestTemplateRegistryRender(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}
	// Create a template cache
	cache, err := newTemplateCache()
	require.NoError(t, err)

	// Create a template registry
	registry := &TemplateRegistry{
		templates: cache,
	}

	// Create Echo context
	e := echo.New()

	// Create proper language and voice maps
	langCodes := []models.LanguageCode{
		{Code: "en", Name: "English"},
	}
	voices := []models.Voice{
		{Name: "en-US-Standard-A", SsmlGender: 1},
	}

	testCases := []struct {
		name          string
		templateName  string
		data          interface{}
		expectedError bool
		contains      []string
	}{
		{
			name:          "Valid home template",
			templateName:  "home.gohtml",
			data:          newTemplateData(langCodes, voices, ""),
			expectedError: false,
			contains: []string{
				"<title>Home - TalkLikeTV</title>",
				"What is it?",
				"How to use it",
			},
		},
		{
			name:          "Valid audio template with data",
			templateName:  "audio.gohtml",
			data:          newTemplateData(langCodes, voices, ""),
			expectedError: false,
			contains: []string{
				"<title>Audio - TalkLikeTV</title>",
			},
		},
		{
			name:          "Valid parse template",
			templateName:  "parse.gohtml",
			data:          newTemplateData(langCodes, voices, ""),
			expectedError: false,
			contains: []string{
				"<title>Parse - TalkLikeTV</title>",
			},
		},
		{
			name:          "Non-existent template",
			templateName:  "nonexistent.gohtml",
			data:          nil,
			expectedError: true,
			contains:      []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Create a buffer to capture rendered output
			buf := new(bytes.Buffer)

			// Test the render function
			err := registry.Render(buf, tc.templateName, tc.data, c)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				result := buf.String()
				for _, expected := range tc.contains {
					assert.Contains(t, result, expected)
				}
			}
		})
	}
}

func TestNewTemplateData(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}
	// Test with empty slices
	t.Run("EmptySlices", func(t *testing.T) {
		languages := []models.LanguageCode{}
		voices := []models.Voice{}
		errorMsg := "test error"

		data := newTemplateData(languages, voices, errorMsg)

		assert.NotNil(t, data)
		assert.Equal(t, languages, data.LanguageCodes)
		assert.Equal(t, voices, data.Voices)
		assert.Equal(t, errorMsg, data.Error)
		assert.Equal(t, []int{3, 4, 5, 6, 7, 8, 9, 10}, data.PauseDurations)
	})

	// Test with populated slices
	t.Run("PopulatedSlices", func(t *testing.T) {
		languages := []models.LanguageCode{
			{Code: "en", Name: "English"},
			{Code: "es", Name: "Spanish"},
		}
		voices := []models.Voice{
			{Name: "voice1", SsmlGender: 2},
			{Name: "voice2", SsmlGender: 1},
		}
		errorMsg := ""

		data := newTemplateData(languages, voices, errorMsg)

		assert.NotNil(t, data)
		assert.Equal(t, languages, data.LanguageCodes)
		assert.Equal(t, voices, data.Voices)
		assert.Equal(t, errorMsg, data.Error)
		assert.Len(t, data.LanguageCodes, 2)
		assert.Len(t, data.Voices, 2)
		assert.Equal(t, []int{3, 4, 5, 6, 7, 8, 9, 10}, data.PauseDurations)
	})
}

func TestMockEmbeddedFile(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}
	// This test verifies that we can access files from the embedded filesystem
	// It's important because our template functionality depends on it
	_, err := ui.Files.Open("html/base.gohtml")
	assert.NoError(t, err, "Should be able to open base template from embedded filesystem")

	_, err = ui.Files.Open("html/pages/home.gohtml")
	assert.NoError(t, err, "Should be able to open home template from embedded filesystem")

	// Test common templates
	_, err = ui.Files.Open("html/common/header.gohtml")
	assert.NoError(t, err, "Should be able to open common header template from embedded filesystem")
}

// Package oapi provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.1 DO NOT EDIT.
package oapi

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Error defines model for Error.
type Error struct {
	// Code Error code
	Code int32 `json:"code"`

	// Message Error message
	Message string `json:"message"`
}

// Translates defines model for Translates.
type Translates struct {
	LanguageId int16  `json:"languageId"`
	Phrase     string `json:"phrase"`
	PhraseHint string `json:"phraseHint"`
	PhraseId   int64  `json:"phraseId"`
}

// AudioFromFileMultipartBody defines parameters for AudioFromFile.
type AudioFromFileMultipartBody struct {
	// FileLanguageId the original language of the file you are uploading
	FileLanguageId string             `json:"fileLanguageId"`
	FilePath       openapi_types.File `json:"filePath"`

	// FromVoiceId the language you know
	FromVoiceId string `json:"fromVoiceId"`

	// Pattern pattern is the pattern used to construct the audio files. You have 3 choices:
	// 1 is beginner and repeats closer together --
	// 2 is intermediate --
	// 3 is advanced and repeats phrases less often and should only be used if you are at an advanced level --
	// 4 is review and repeats each phrase one time and can be used to review already learned phrases
	Pattern *string `json:"pattern,omitempty"`

	// Pause the pause in seconds between phrases in the audiofile (default is 5)
	Pause *string `json:"pause,omitempty"`

	// TitleName choose a descriptive title that includes to and from languages
	TitleName string `json:"titleName"`

	// ToVoiceId the language you want to learn
	ToVoiceId string `json:"toVoiceId"`

	// Token tokens are required to be able to successfully request an audio file
	Token string `json:"token"`
}

// AudioFromFileMultipartRequestBody defines body for AudioFromFile for multipart/form-data ContentType.
type AudioFromFileMultipartRequestBody AudioFromFileMultipartBody

// ServerInterface represents all server handlers.
type ServerInterface interface {

	// (POST /audio)
	AudioFromFile(ctx echo.Context) error
}

// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler ServerInterface
}

// AudioFromFile converts echo context to params.
func (w *ServerInterfaceWrapper) AudioFromFile(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.AudioFromFile(ctx)
	return err
}

// This is a simple interface which specifies echo.Route addition functions which
// are present on both echo.Echo and echo.Group, since we want to allow using
// either of them for path registration
type EchoRouter interface {
	CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

// RegisterHandlers adds each server route to the EchoRouter.
func RegisterHandlers(router EchoRouter, si ServerInterface) {
	RegisterHandlersWithBaseURL(router, si, "")
}

// Registers handlers, and prepends BaseURL to the paths, so that the paths
// can be served under a prefix.
func RegisterHandlersWithBaseURL(router EchoRouter, si ServerInterface, baseURL string) {

	wrapper := ServerInterfaceWrapper{
		Handler: si,
	}

	router.POST(baseURL+"/audio", wrapper.AudioFromFile)

}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %w", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	res := make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	resolvePath := PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		pathToFile := url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}

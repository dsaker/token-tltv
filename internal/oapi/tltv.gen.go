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

	"H4sIAAAAAAAC/5RXUXPjuA3+Kxi2D+2MIjvJ3k7rp17bXJuZvZ1MN72ZTr0PCAVJvFCgSkJ2fDv+7x1Q",
	"lm0l3m3vKREBAh8/AB/pL8aGrg9MLMmsvphkW+ow/3sXY4j6Tx9DT1Ec5WUbKtK/FSUbXS8usFmNzpBt",
	"halD7FDMyjiW2xtTGNn1NH5SQ9HsC9NRSth8NdBkPm5NEh03Zr8vTKT/DC5SZVb/NoeEk/vnfWEeI3Ly",
	"KCPaOXaP3AzY0H2lX+cwr99fhNm3EVNG+QrHZPq7Y/mG+W2i9+8uJHp1quPe4hzxEc0s9+e97nZch7E4",
	"LGgzIurQebMy1ZBkt8Ud059s6CwmKZnEFIaxUxB/VTt8wufxxPNqPKJ//uCe6fEncAkQJjjgCSM7bgD7",
	"3juL6g8VJdcwVSABWvI9DIligrChaENHIC1Br6XBAdYcaiEGYhsGFopUwdZJC0FaiqdE2PephHuBUNca",
	"DKGnmAKjd79QdcJBLz1FR2wJ1vy0A/Q+bNUwYpAAtg0hjSBST9bVzsJIZNLFHWyRRR3rYIcEgUv4Z95r",
	"kWHofcAK1owg9CJQO08j3imEY+gxYhOxb0ErXkBgOpgVNHjHVECIQBtiQIZP/3jMgQpA1tgK7ZzPrfMe",
	"GmKKKAQIiZQG+PHhFnCoXMib89lqtM47UbcjI9LGMDQteJeEdKWENa/5X2HIJ7KRclQ+iwVJIrqmFahj",
	"6DJVeRkFHkISWGTXhRp1vYT7OjuJE0/QYoI1dyHSGa/I2aPDlwwfBWzg2jXlj/jycegeJvZkPG0kGSID",
	"wi+u76ka04f6RHqC1Hsn4FjCFHnNPHRPFNXxkLmEe4hkQ9cRV5AEo4ycuARb3EEKsJuIaMk+K4kdPhOk",
	"IWqPoKg95pRr3mLS4iaqwIYYyYrflWvW+XSWeFSIwzx936NtCW7KpSnMEHUEW5E+rRaL7XZbYjaXITaL",
	"w960+HD/l7uPn+6ubspl2UrndQ4zpecTuDGF2VBM42Bel8tyqX6hJ8bemZW5zUuF6VHarHZjtbIKhiRv",
	"pXbqgEtTfWqJ3Alj++tkv4h2cIpjNcq8kP2UyieajUFS1wtDkJlTXc5trhppvteEP8TQ/eC8KpzKISX5",
	"c6h2k67RqLTd4MX1GGWhQ3ZVoeDp4nor+Yrtw0z25yRoC4XoGsfoT0SE+tT82icY6cCBynth6AW7Ptfn",
	"3R/fXlJFzvqA0s7U/8kxxt1F9xi6n4KzX0V4BKZgnjlsZxiu399citqjCEV+G/FgUEnPknz4HNIo3TZw",
	"kjhYydYzpSlBxaPFDcGtyqmzlFaw5muN9ESNY6aYtSxSTygJrA+JIkhoKMv61RWs+Ubd9e6LHVVOWzAv",
	"3+YrptogW6pmUSY18ZQSjNeGmlMbBl9BYL/TzsvwXX2sF0oWtymgpw35MdM7zRRp42g7y0NoJz3PbSuu",
	"o+xw6O2JoGmrj4TVbpwaqiaYubvPinO5NEOiy6XOJh2jRDZwpcTKlojPL5pjXXKD/q6iGgcveqrvfj9L",
	"/p0+jvDFdUNnVtfLwnSOx4/bC6iy5nzMMvZGK8bLE+G4vJlkP4ulY+uHaryMlLGsGlPXpksUSPj/O366",
	"mTPTswPeLP/wPx+Jp2MVr+VgPnrnoM5m+PMxQXj6mayY/OSaQ9Ywp7haiXx9MI5EBW4GFcsw9/n2QUvQ",
	"Bwj87e4RFkcq9Wmhku1Sfgscd7sqjTfS6eQSB8pUpD7oPaNE3yyXr/T07MGx+DkFnoupE+ryxt9Gqs3K",
	"/GZx+r2wOPxYWJw9ufdHrjBG3F2i6iAo2iC5fSd84+szd/KvgvgtZONvmAsgBtYHoxWqgCaf/X7/3wAA",
	"AP//HWM8yg8NAAA=",
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

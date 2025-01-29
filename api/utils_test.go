package api

import (
	"bytes"
	"flag"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/translates"

	"testing"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"talkliketv.click/tltv/internal/config"
	"talkliketv.click/tltv/internal/test"
	"talkliketv.click/tltv/internal/util"
)

var (
	testCfg TestConfig
)

const (
	audioBasePath = "/v1/audio"
	tokenFilePath = "/tokens.json" //nolint:gosec
)

type TestConfig struct {
	config.Config
}

var (
	tokenStrings []string
	tokenCount   int
	mu           sync.RWMutex
)

// testCase struct groups together the fields necessary for running most of the test cases
type testCase struct {
	name          string
	buildStubs    func(stubs test.MockStubs)
	multipartBody func(t *testing.T) (*bytes.Buffer, *multipart.Writer)
	checkResponse func(res *http.Response)
}

func TestMain(m *testing.M) {
	_ = config.SetConfigs(&testCfg.Config)
	flag.BoolVar(&util.Integration, "integration", false, "Run integration tests")
	flag.Parse()
	testCfg.TTSBasePath = test.AudioBasePath
	plaintext, err := test.CreateTokensFile(test.AudioBasePath, tokenFilePath, 100)
	if err != nil {
		log.Fatal(err)
	}
	tokenStrings = plaintext
	testCfg.TokenFilePath = test.AudioBasePath + tokenFilePath

	// create maps of voices and languages depending on platform
	if translates.GlobalPlatform == translates.Google {
		models.MakeGoogleMaps()
	} else {
		models.MakeAmazonMaps()
	}
	os.Exit(m.Run())
}

// readBody reads the http response body and returns it as a string
func readBody(t *testing.T, rs *http.Response) string {
	// Read the checkResponse body from the test server.
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(rs.Body)

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}
	bytes.TrimSpace(body)

	return string(body)
}

// setupServerTest sets up testCase that will include the middleware not included in handler tests
func setupServerTest(ctrl *gomock.Controller, tc testCase) *httptest.Server {
	stubs := test.NewMockStubs(ctrl)
	tc.buildStubs(stubs)

	e := NewServer(testCfg.Config, stubs.TranslateX, stubs.AudioFileX)

	ts := httptest.NewServer(e)

	return ts
}

// createMultiPartBody creates and returns a multipart Writer.
// data is the data you want to write to the file.
// m is the map[string][string] of the fields, values you want to write to the multipart body
func createMultiPartBody(t *testing.T, data []byte, filename string, m map[string]string) (*bytes.Buffer, *multipart.Writer) {
	err := os.WriteFile(filename, data, 0600)
	require.NoError(t, err)
	file, err := os.Open(filename)
	require.NoError(t, err)
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file_path", filename)
	require.NoError(t, err)
	_, err = io.Copy(part, file)
	require.NoError(t, err)
	for key, val := range m {
		err = writer.WriteField(key, val)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return body, writer
}

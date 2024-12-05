package api

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"

	"strings"
	"talkliketv.click/tltv/internal/models"
	"testing"

	"github.com/docker/go-connections/nat"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"go.uber.org/mock/gomock"
	"talkliketv.click/tltv/internal/config"
	mocka "talkliketv.click/tltv/internal/mock/audiofile"
	mockt "talkliketv.click/tltv/internal/mock/translates"
	"talkliketv.click/tltv/internal/test"
	"talkliketv.click/tltv/internal/util"
)

var (
	testCfg    TestConfig
	count      = 0
	mappedPort nat.Port
)

const (
	audioBasePath = "/v1/audio"
)

type MockStubs struct {
	TranslateX       *mockt.MockTranslateX
	TranslateClientX *mockt.MockTranslateClientX
	TtsClientX       *mockt.MockTTSClientX
	AudioFileX       *mocka.MockAudioFileX
}

type TestConfig struct {
	config.Config
	conn      *sql.DB
	container testcontainers.Container
}

// NewMockStubs creates instantiates new instances of all the mock interfaces for testing
func NewMockStubs(ctrl *gomock.Controller) MockStubs {
	return MockStubs{
		TranslateX:       mockt.NewMockTranslateX(ctrl),
		TranslateClientX: mockt.NewMockTranslateClientX(ctrl),
		TtsClientX:       mockt.NewMockTTSClientX(ctrl),
		AudioFileX:       mocka.NewMockAudioFileX(ctrl),
	}
}

// testCase struct groups together the fields necessary for running most of the test cases
type testCase struct {
	name          string
	body          interface{}
	buildStubs    func(stubs MockStubs)
	multipartBody func(t *testing.T) (*bytes.Buffer, *multipart.Writer)
	checkRecorder func(rec *httptest.ResponseRecorder)
	checkResponse func(res *http.Response)
	values        map[string]any
	cleanUp       func(*testing.T)
}

func TestMain(m *testing.M) {
	_ = config.SetConfigs(&testCfg.Config)
	flag.BoolVar(&util.Integration, "integration", false, "Run integration tests")
	flag.Parse()
	testCfg.TTSBasePath = test.AudioBasePath

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

// randomLanguage creates a random db Language for testing
func randomLanguage() (language models.Language) {
	return models.Language{
		ID:       rand.Int(),
		Language: test.RandomString(6),
		Name:     "en",
	}
}

// setupHandlerTest sets up a testCase that will be run through the handler
// these tests will not include the middleware JWT verification or the automated validation
// through openapi
func setupHandlerTest(t *testing.T, ctrl *gomock.Controller, tc testCase, urlBasePath, body, method string) (*Server, echo.Context, *httptest.ResponseRecorder) {
	stubs := NewMockStubs(ctrl)
	tc.buildStubs(stubs)

	e := echo.New()
	srv := NewServer(e, testCfg.Config, stubs.TranslateX, stubs.AudioFileX)

	urlPath := urlBasePath + string(rune(rand.Int()))

	req := httptest.NewRequest(method, urlPath, strings.NewReader(body))

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	return srv, c, rec
}

// setupServerTest sets up testCase that will include the middleware not included in handler tests
func setupServerTest(t *testing.T, ctrl *gomock.Controller, tc testCase) *httptest.Server {
	stubs := NewMockStubs(ctrl)
	tc.buildStubs(stubs)

	e := echo.New()
	_ = NewServer(e, testCfg.Config, stubs.TranslateX, stubs.AudioFileX)

	ts := httptest.NewServer(e)

	return ts
}

// jsonRequest creates a new request which has json as the body and sets the Header content type to
// application/json
func jsonRequest(t *testing.T, json []byte, ts *httptest.Server, urlPath, method string) *http.Request {
	req, err := http.NewRequest(method, ts.URL+urlPath, bytes.NewBuffer(json))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	return req
}

// createMultiPartBody creates and returns a multipart Writer.
// data is the data you want to write to the file.
// m is the map[string][string] of the fields, values you want to write to the multipart body
func createMultiPartBody(t *testing.T, data []byte, filename string, m map[string]string) (*bytes.Buffer, *multipart.Writer) {
	err := os.WriteFile(filename, data, 0600)
	require.NoError(t, err)
	file, err := os.Open(filename)
	require.NoError(t, err)
	fmt.Println(file.Name())
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("filePath", filename)
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

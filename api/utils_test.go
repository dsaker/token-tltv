package api

import (
	"bytes"
	"context"
	"flag"
	"github.com/playwright-community/playwright-go"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"talkliketv.com/tltv/internal/config"
	"talkliketv.com/tltv/internal/models"
	"talkliketv.com/tltv/internal/services/tokens"
	"talkliketv.com/tltv/internal/testflags"
	"talkliketv.com/tltv/internal/testutil"
	"talkliketv.com/tltv/internal/util"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	testCfg  TestConfig
	local    = false
	headless = true
	saFile   string
)

const (
	audioBasePath  = "/v1/audio"
	parseBasePath  = "/v1/parse"
	validSentences = "This is the first sentence.\nThis is the second sentence.\nThis is the third sentence.\nThis is the fourth sentence.\nThis is the fifth sentence.\n"
)

type TestConfig struct {
	config.Config
	browser *playwright.Browser
	url     string
	tc      *testutil.TltvContainer
}

// testCase struct groups together the fields necessary for running most of the test cases
type testCase struct {
	name          string
	mocks         func(stubs testutil.MockStubs)
	multipartBody func(t *testing.T) (*bytes.Buffer, *multipart.Writer)
	checkResponse func(res *http.Response)
}

func TestMain(m *testing.M) {
	err := testCfg.SetConfigs()
	if err != nil {
		log.Fatal(err)
	}

	testflags.ParseFlags()
	flag.Parse()

	util.Test = testflags.TestType
	local = testflags.Local
	headless = testflags.Headless
	saFile = testflags.SAFile

	testCfg.url = "http://localhost:8080"
	if util.Test == "end-to-end" {
		getBrowserContext(headless, saFile)
	}

	testCfg.TTSBasePath = testutil.AudioBasePath

	// Run tests
	exitCode := testflags.RunTests(m)

	os.Exit(exitCode)
}

func addTokenFirestore(ctx context.Context, t *testing.T, client *models.Models) (string, string) {
	// generate new token
	testToken, plaintext, err := tokens.GenerateToken()
	require.NoError(t, err)

	// add test token to the collection
	err = client.AddToken(ctx, *testToken)
	require.NoError(t, err)

	return plaintext, testToken.Hash
}

// getBrowserContext sets up the playwright browser context
func getBrowserContext(headless bool, saFile string) {
	if !local {
		ctx := context.Background()
		container, err := testutil.StartContainer(ctx, testCfg.ProjectId, saFile)
		testCfg.tc = container
		if err != nil {
			log.Fatal(err)
		}
		testCfg.url = testCfg.tc.URI
	}

	runOption := &playwright.RunOptions{
		SkipInstallBrowsers: true,
	}
	err := playwright.Install(runOption)
	if err != nil {
		log.Fatal(err)
	}

	pw, err := playwright.Run()
	if err != nil {
		log.Fatal(err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
	})
	if err != nil {
		log.Fatal(err)
	}

	testCfg.browser = &browser
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
	stubs := testutil.NewMockStubs(ctrl)
	tc.mocks(stubs)

	srv := NewServer(testCfg.Config, stubs.TranslateX, stubs.AudioFileX, stubs.ModelsX)

	e := srv.NewEcho(nil)
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

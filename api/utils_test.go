package api

import (
	"bytes"
	"context"
	"flag"
	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
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
)

var (
	local = false
)

type TestConfig struct {
	config.Config
	browserContext *playwright.BrowserContext
	url            string
}

// testCase struct groups together the fields necessary for running most of the test cases
type testCase struct {
	name          string
	buildStubs    func(stubs test.MockStubs)
	multipartBody func(t *testing.T) (*bytes.Buffer, *multipart.Writer)
	checkResponse func(res *http.Response)
}

func TestMain(m *testing.M) {
	err := testCfg.SetConfigs()
	if err != nil {
		log.Fatal(err)
	}
	flag.StringVar(&util.Test, "test", "unit", "type of tests to run [unit|integration|end-to-end]")
	flag.BoolVar(&local, "local", false, "if true end-to-end tests will be run in local mode")
	flag.Parse()

	testCfg.browserContext, testCfg.url = getBrowserContext()

	testCfg.TTSBasePath = test.AudioBasePath

	// Run tests
	exitCode := m.Run()

	//if util.Integration {
	//	// Code to run after tests
	//	teardown()
	//}

	os.Exit(exitCode)
}

func getBrowserContext() (*playwright.BrowserContext, string) {
	var url = "http://localhost:8080"
	if !local {
		ctx := context.Background()
		container := test.StartContainer(ctx, t, testCfg.ProjectId)
		defer func(container *test.TltvContainer, ctx context.Context, opts ...testcontainers.TerminateOption) {
			if err := container.Terminate(ctx, opts...); err != nil {
				require.NoError(t, err)
			}
		}(container, ctx)
		url = container.URI
	}

	runOption := &playwright.RunOptions{
		SkipInstallBrowsers: true,
	}
	err := playwright.Install(runOption)
	require.NoError(t, err)
	pw, err := playwright.Run()
	assert.NoError(t, err)
	defer func(pw *playwright.Playwright) {
		err = pw.Stop()
		require.NoError(t, err)
	}(pw)

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	assert.NoError(t, err)
	defer func(browser playwright.Browser, options ...playwright.BrowserCloseOptions) {
		err = browser.Close(options...)
		require.NoError(t, err)
	}(browser)

	// Create a new browser context
	browserContext, err := browser.NewContext(playwright.BrowserNewContextOptions{
		AcceptDownloads: playwright.Bool(true), // Ensure downloads are enabled
	})
	assert.NoError(t, err)
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

	e := NewServer(testCfg.Config, stubs.TranslateX, stubs.AudioFileX, stubs.TokensX)

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

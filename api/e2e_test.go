package api

import (
	"context"
	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/util"
	"testing"
	"time"
)

// TestEndToEndParse tests the ParseFile endpoint end-to-end
// Program arguments: -env=dev -project-id=token-tltv-test -test=end-to-end [-local=true] [-headless=false]
func TestEndToEndParse(t *testing.T) {
	if util.Test != "end-to-end" {
		t.Skip("skipping end-to-end test")
	}

	b := *testCfg.browser
	// Create a new browser context
	browserContext, err := b.NewContext(playwright.BrowserNewContextOptions{
		AcceptDownloads: playwright.Bool(true), // Ensure downloads are enabled
	})
	require.NoError(t, err)
	defer func(browserContext playwright.BrowserContext, options ...playwright.BrowserContextCloseOptions) {
		err = browserContext.Close(options...)
		require.NoError(t, err)
	}(browserContext)

	page, err := browserContext.NewPage()
	require.NoError(t, err)

	resp, err := page.Goto(testCfg.url)
	require.NoError(t, err)

	assert.Contains(t, resp.StatusText(), http.StatusText(http.StatusOK))
	// sleep between clicks or echo golang rate limiter gets triggered
	time.Sleep(time.Second * 1)
	// Get element by ID
	err = page.GetByText("ParseFile").Click()
	require.NoError(t, err)

	time.Sleep(time.Second * 1)
	pageTitle, err := page.Title()
	require.Contains(t, pageTitle, "Parse - TalkLikeTV")

	// Trigger the file input, for example, by clicking a button
	fileChooser, err := page.ExpectFileChooser(func() error {
		err = page.Locator("#text-file").Click()
		require.NoError(t, err)
		return nil
	})

	err = fileChooser.SetFiles("../internal/testutil/sample.srt")
	require.NoError(t, err)

	// Wait for the download event
	downloadChan := make(chan playwright.Download)
	page.On("download", func(d playwright.Download) {
		downloadChan <- d
	})

	err = page.Locator("#submit-parse-form").Click()
	require.NoError(t, err)
	download, ok := <-downloadChan
	if !ok {
		t.Fatal("download channel closed")
	}

	dir := filepath.Join("tmp", strings.Split(t.Name(), "/")[0])
	savePath := filepath.Join(dir, download.SuggestedFilename())
	defer os.RemoveAll(dir)
	err = download.SaveAs(savePath)
	defer require.NoError(t, err)
	fileInfo, err := os.Stat(savePath)
	require.NoError(t, err)

	require.Equal(t, fileInfo.Size(), int64(274))
}

// TestEndToEndAudio tests the Audio endpoint end-to-end
// Program arguments: -env=dev -project-id=[test project id] -test=end-to-end [-local=true] [-headless=false]
func TestEndToEndAudio(t *testing.T) {
	if util.Test != "end-to-end" {
		t.Skip("skipping end-to-end test")
	}

	ctx := context.Background()

	// Use the application default credentials
	client, err := testCfg.FirestoreClient()
	require.NoError(t, err)
	defer client.Close()

	// generate new token and add it to the collection
	testToken, plaintext, err := models.GenerateToken()
	require.NoError(t, err)
	tokensColl := client.Collection(util.TokenColl)
	tokens := models.Tokens{Coll: tokensColl}

	// add test token to the collection
	err = tokens.AddToken(ctx, *testToken)
	require.NoError(t, err)

	b := *testCfg.browser
	// Create a new browser context
	browserContext, err := b.NewContext(playwright.BrowserNewContextOptions{
		AcceptDownloads: playwright.Bool(true), // Ensure downloads are enabled
	})
	require.NoError(t, err)
	defer func(browserContext playwright.BrowserContext, options ...playwright.BrowserContextCloseOptions) {
		err = browserContext.Close(options...)
		require.NoError(t, err)
	}(browserContext)

	page, err := browserContext.NewPage()
	require.NoError(t, err)

	resp, err := page.Goto(testCfg.url)
	require.NoError(t, err)
	assert.Contains(t, resp.StatusText(), http.StatusText(http.StatusOK))

	// sleep between clicks or echo golang rate limiter gets triggered
	time.Sleep(time.Second * 1)
	// Get element by ID
	err = page.Locator("#a-audio").Click()
	require.NoError(t, err)

	time.Sleep(time.Second * 1)
	pageTitle, err := page.Title()
	require.Contains(t, pageTitle, "Audio - TalkLikeTV")

	err = page.Locator("#token-input").Fill(plaintext)
	require.NoError(t, err)

	err = page.Locator("#title-input").Fill("Random Title")
	require.NoError(t, err)

	// Create a string slice
	selectsMap := map[string][]string{
		"#file-lang-select":  {"Spanish"},
		"#from-lang-select":  {"English"},
		"#from-voice-select": {"en-US-Standard-A"},
		"#to-lang-select":    {"Spanish"},
		"#to-voice-select":   {"es-ES-Standard-A"},
		"#pause-select":      {"3"},
		"#pattern-select":    {"advanced"},
	}

	order := []string{"#file-lang-select", "#from-lang-select", "#from-voice-select", "#to-lang-select", "#to-voice-select", "#pause-select", "#pattern-select"}

	for k := range order {
		v, ok := selectsMap[order[k]]
		if !ok {
			t.Fatal("key not found in selectsMap")
		}
		_, err = page.Locator(order[k]).SelectOption(playwright.SelectOptionValues{ValuesOrLabels: &v})
		require.NoError(t, err)
	}

	// Trigger the file input, for example, by clicking a button
	fileChooser, err := page.ExpectFileChooser(func() error {
		err = page.Locator("#text-file").Click()
		require.NoError(t, err)
		return nil
	}, playwright.PageExpectFileChooserOptions{
		// Set a timeout for the file chooser after 5 seconds
		Timeout: playwright.Float(5000),
	})

	err = fileChooser.SetFiles("../internal/testutil/sample.srt")
	require.NoError(t, err)

	// Wait for the download event
	downloadChan := make(chan playwright.Download)
	page.On("download", func(d playwright.Download) {
		downloadChan <- d
	})

	err = page.Locator("#submit-audio-form").Click()
	require.NoError(t, err)

	var download playwright.Download
	var ok bool
	// wait for download event.. timeout after 5 seconds if no event received
	select {
	case download, ok = <-downloadChan:
		if !ok {
			t.Fatal("download channel closed")
		}
		return
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout reached, no message received")
	}

	dir := filepath.Join("tmp", strings.Split(t.Name(), "/")[0])
	savePath := filepath.Join(dir, download.SuggestedFilename())
	defer os.RemoveAll(dir)
	err = download.SaveAs(savePath)
	defer require.NoError(t, err)
	fileInfo, err := os.Stat(savePath)
	require.NoError(t, err)

	require.True(t, fileInfo.Size() > 147000 && fileInfo.Size() < 148000)
}

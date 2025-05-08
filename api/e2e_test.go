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

	var download playwright.Download
	var ok bool
	// Add timeout protection like in TestEndToEndAudio
	select {
	case download, ok = <-downloadChan:
		if !ok {
			t.Fatal("download channel closed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for download")
	}

	dir := filepath.Join("tmp", strings.Split(t.Name(), "/")[0])
	savePath := filepath.Join(dir, download.SuggestedFilename())
	defer os.RemoveAll(dir)
	err = download.SaveAs(savePath)
	require.NoError(t, err)
	fileInfo, err := os.Stat(savePath)
	require.NoError(t, err)

	require.Equal(t, fileInfo.Size(), int64(274))
}

// TestEndToEndAudio tests the Audio endpoint end-to-end
// Program arguments: -project-id=[test project id] -test=end-to-end [-local=true] [-headless=false]
// TestEndToEndAudio tests the Audio endpoint end-to-end
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

	// Fill the token and title fields
	err = page.Locator("#token-input").Fill(plaintext)
	require.NoError(t, err)

	err = page.Locator("#title-input").Fill("Random Title")
	require.NoError(t, err)

	// Select English as the "from" language
	err = page.Locator("#from-en").Click()
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 500)

	// Select a voice for the "from" language
	err = page.Locator("#from-voice-input-en-US-Standard-A").Click()
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 500)

	// Select Spanish as the "to" language
	err = page.Locator("#to-es").Click()
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 500)

	// Select a voice for the "to" language
	err = page.Locator("#to-voice-input-es-ES-Standard-A").Click()
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 500)

	// Select pause duration
	err = page.Locator("#pause-3").Click()
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 500)

	// Select pattern
	err = page.Locator("#pattern-advanced").Click()
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 500)

	// Trigger the file input, for example, by clicking a button
	fileChooser, err := page.ExpectFileChooser(func() error {
		err = page.Locator("#text-file").Click()
		require.NoError(t, err)
		return nil
	}, playwright.PageExpectFileChooserOptions{
		// Set a timeout for the file chooser after 5 seconds
		Timeout: playwright.Float(5000),
	})

	err = fileChooser.SetFiles("../internal/testutil/sample.txt")
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
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout reached, no message received")
	}

	dir := filepath.Join("tmp", strings.Split(t.Name(), "/")[0])
	savePath := filepath.Join(dir, download.SuggestedFilename())
	defer os.RemoveAll(dir)
	err = download.SaveAs(savePath)
	require.NoError(t, err)
	fileInfo, err := os.Stat(savePath)
	require.NoError(t, err)

	require.True(t, fileInfo.Size() > 210000 && fileInfo.Size() < 220000)
}

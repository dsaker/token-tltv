package audiofile

import (
	"archive/zip"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"talkliketv.com/tltv/internal/interfaces"
	"talkliketv.com/tltv/internal/mock"
	"talkliketv.com/tltv/internal/services"
	"talkliketv.com/tltv/internal/testutil"
	"talkliketv.com/tltv/internal/util"
	"testing"
)

func TestCreateMp3Zip(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}
	t.Parallel()

	// Create a base temporary directory for all tests
	baseDir, err := os.MkdirTemp("/tmp/", "createzipfile-test-")
	require.NoError(t, err)
	defer os.RemoveAll(baseDir)

	testCases := []audioFileTestCase{
		{
			name: "Success - multiple files",
			createTitle: func(t *testing.T) (interfaces.Title, string) {
				title := testutil.RandomTitle()
				tmpDir := filepath.Join(baseDir, title.Name, "/")
				err := os.MkdirAll(tmpDir, 0777)
				require.NoError(t, err)

				// Create multiple test files
				file := createFile(t, filepath.Join(tmpDir, "file1.mp3"), "test audio content 1")
				createFile(t, filepath.Join(tmpDir, "file2.mp3"), "test audio content 2")
				createFile(t, filepath.Join(tmpDir, "file3.mp3"), "test audio content 3")
				require.FileExists(t, file.Name())

				return title, tmpDir
			},
			buildStubs: func(ma *mock.MockcmdRunnerX) {
				ma.EXPECT().
					CombinedOutput(gomock.Any()).Times(3).
					Return([]byte{}, nil)
			},
			checkReturn: func(t *testing.T, file *os.File, err error) {
				require.NoError(t, err)
				require.FileExists(t, file.Name())
			},
		},
		{
			name: "No files",
			createTitle: func(t *testing.T) (interfaces.Title, string) {
				title := testutil.RandomTitle()
				tmpDir := filepath.Join(baseDir, title.Name)
				err := os.MkdirAll(tmpDir, 0777)
				require.NoError(t, err)
				return title, tmpDir
			},
			buildStubs: func(ma *mock.MockcmdRunnerX) {
				// No calls expected
			},
			checkReturn: func(t *testing.T, file *os.File, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "no files found in CreateMp3Zip")
				require.Nil(t, file)
			},
		},
		{
			name: "Ffmpeg error",
			createTitle: func(t *testing.T) (interfaces.Title, string) {
				title := testutil.RandomTitle()
				tmpDir := filepath.Join(baseDir, title.Name)
				err := os.MkdirAll(tmpDir, 0777)
				require.NoError(t, err)

				// Create a test file
				createFile(t, filepath.Join(tmpDir, "test.mp3"), "test audio content")

				return title, tmpDir
			},
			buildStubs: func(ma *mock.MockcmdRunnerX) {
				ma.EXPECT().
					CombinedOutput(gomock.Any()).Times(1).
					Return([]byte("ffmpeg error output"), fmt.Errorf("ffmpeg failed"))
			},
			checkReturn: func(t *testing.T, file *os.File, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "ffmpeg failed")
				require.Nil(t, file)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			cmdX := mock.NewMockcmdRunnerX(ctrl)
			tc.buildStubs(cmdX)

			audioFile := New(cmdX)
			title, tmpDir := tc.createTitle(t)

			// Ensure cleanup
			//defer os.RemoveAll(tmpDir)

			osFile, err := audioFile.CreateMp3Zip(title, tmpDir)
			tc.checkReturn(t, osFile, err)
		})
	}
}

func TestCreatePhrasesZip(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}
	t.Parallel()

	basePath := "/tmp/TestCreatePhrasesZip/"
	defer os.RemoveAll(basePath)

	stringsSlice := []string{
		"Absolutely! Here's a zany paragraph packed with punctuation:",
		"Wow! Did you see that?! A purple penguin — yes, a purple penguin! —",
		"just roller-skated past my window... (in broad daylight!)",
		"while juggling pineapples, watermelons, and, believe it or not,",
		"rubber chickens?!? Not only that, but it was whistling a tune",
		"(sounded suspiciously like Beethoven's Fifth) and waving a little flag that said,",
		"'Viva Las Veggies!' 🍍🍉🥒.",
		"Now, I've seen some strange things in my life,",
		"but this takes the (gluten-free) cake. I mean... really?!?"}

	testCases := []audioFileTestCase{
		{
			name: "3 files",
			createTitle: func(t *testing.T) (interfaces.Title, string) {
				title := testutil.RandomTitle()
				tmpDir := basePath + title.Name + "/"
				err := os.MkdirAll(tmpDir, 0777)
				require.NoError(t, err)
				return title, tmpDir
			},
			buildStubs: func(ma *mock.MockcmdRunnerX) {
			},
			checkReturn: func(t *testing.T, file *os.File, err error) {
				require.NoError(t, err)
				require.FileExists(t, file.Name())

				// Verify zip contents
				reader, err := zip.OpenReader(file.Name())
				require.NoError(t, err)
				defer reader.Close()

				fileCount := len(reader.File)
				require.Equal(t, 3, fileCount, "Zip should contain exactly 3 files")

				// Check file extensions and naming pattern
				for i, f := range reader.File {
					assert.True(t, strings.HasSuffix(f.Name, ".txt"),
						fmt.Sprintf("File %d should have .txt extension", i))
				}
			},
			values:       map[string]any{"size": 3},
			stringsSlice: stringsSlice,
		},
		{
			name: "5 files",
			createTitle: func(t *testing.T) (interfaces.Title, string) {
				title := testutil.RandomTitle()
				tmpDir := basePath + title.Name + "/"
				err := os.MkdirAll(tmpDir, 0777)
				require.NoError(t, err)
				return title, tmpDir
			},
			buildStubs: func(ma *mock.MockcmdRunnerX) {
			},
			checkReturn: func(t *testing.T, file *os.File, err error) {
				require.NoError(t, err)
				require.FileExists(t, file.Name())

				// Verify zip contents
				reader, err := zip.OpenReader(file.Name())
				require.NoError(t, err)
				defer reader.Close()

				fileCount := len(reader.File)
				require.Equal(t, 5, fileCount, "Zip should contain exactly 5 files")

				// Check file extensions and naming pattern
				for i, f := range reader.File {
					assert.True(t, strings.HasSuffix(f.Name, ".txt"),
						fmt.Sprintf("File %d should have .txt extension", i))
				}
			},
			values:       map[string]any{"size": 2},
			stringsSlice: stringsSlice,
		},
		{
			name: "No One File",
			createTitle: func(t *testing.T) (interfaces.Title, string) {
				title := testutil.RandomTitle()
				tmpDir := basePath + title.Name + "/"
				err := os.MkdirAll(tmpDir, 0777)
				require.NoError(t, err)
				return title, tmpDir
			},
			buildStubs: func(ma *mock.MockcmdRunnerX) {
			},
			checkReturn: func(t *testing.T, file *os.File, err error) {
				require.Error(t, services.ErrOneFile)
			},
			values:       map[string]any{"size": 3},
			stringsSlice: []string{},
		},
		{
			name: "Write error simulation",
			createTitle: func(t *testing.T) (interfaces.Title, string) {
				title := testutil.RandomTitle()
				// Create a path to a directory that's read-only
				tmpDir := basePath + title.Name + "/"
				err := os.MkdirAll(tmpDir, 0444) // Read-only permissions
				require.NoError(t, err)
				return title, tmpDir
			},
			buildStubs: func(ma *mock.MockcmdRunnerX) {
				// No calls expected
			},
			checkReturn: func(t *testing.T, file *os.File, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "permission denied")
				assert.Nil(t, file)
			},
			values:       map[string]any{"size": 3},
			stringsSlice: stringsSlice,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			cmdX := mock.NewMockcmdRunnerX(ctrl)
			tc.buildStubs(cmdX)
			defer ctrl.Finish()

			audioFile := New(cmdX)
			title, tmpDir := tc.createTitle(t)
			chunkedPhrases := slices.Chunk(tc.stringsSlice, tc.values["size"].(int))
			// remove phrasesBasePath after you have sent zipfile
			defer os.RemoveAll(tmpDir)

			osFile, err := audioFile.CreatePhrasesZip(chunkedPhrases, tmpDir, title.Name)

			// If osFile is not nil, ensure it's closed and removed after test
			if osFile != nil {
				defer func() {
					osFile.Close()
					os.Remove(osFile.Name())
				}()
			}

			tc.checkReturn(t, osFile, err)
		})
	}
}

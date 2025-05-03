package audiofile

import (
	"archive/zip"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"talkliketv.click/tltv/internal/mock"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/services"
	"talkliketv.click/tltv/internal/testutil"
	"talkliketv.click/tltv/internal/util"
	"testing"
)

func TestCreateMp3Zip(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}
	t.Parallel()
	testCases := []audioFileTestCase{
		{
			name: "No error",
			createTitle: func(t *testing.T) (models.Title, string) {
				title := testutil.RandomTitle(voicesMap)
				tmpDir := testutil.AudioBasePath + "TestCreateMp3ZipWithFfmpeg/" + title.Name + "/"
				err := os.MkdirAll(tmpDir, 0777)
				require.NoError(t, err)
				file := createFile(
					t,
					tmpDir+"noerror.txt",
					"This is the first sentence.\nThis is the second sentence.\n")
				require.FileExists(t, file.Name())
				return title, tmpDir
			},
			buildStubs: func(ma *mock.MockcmdRunnerX) {
				ma.EXPECT().
					CombinedOutput(gomock.Any()).Times(1).
					Return([]byte{}, nil)
			},
			checkReturn: func(t *testing.T, file *os.File, err error) {
				require.NoError(t, err)
				require.FileExists(t, file.Name())
			},
		},
		{
			name: "No files",
			createTitle: func(t *testing.T) (models.Title, string) {
				title := testutil.RandomTitle(voicesMap)
				tmpDir := testutil.AudioBasePath + "TestCreateMp3ZipWithFfmpeg/" + title.Name + "/"
				err := os.MkdirAll(tmpDir, 0777)
				require.NoError(t, err)
				return title, tmpDir
			},
			buildStubs: func(ma *mock.MockcmdRunnerX) {
			},
			checkReturn: func(t *testing.T, file *os.File, err error) {
				require.Contains(t, err.Error(), "no files found in CreateMp3Zip")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			cmdX := mock.NewMockcmdRunnerX(ctrl)
			tc.buildStubs(cmdX)
			defer ctrl.Finish()

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/fakeurl", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			audioFile := New(cmdX)
			title, tmpDir := tc.createTitle(t)
			osFile, err := audioFile.CreateMp3Zip(c, title, tmpDir)
			tc.checkReturn(t, osFile, err)
		})
	}
}

func TestCreatePhrasesZip(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}
	t.Parallel()

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
			createTitle: func(t *testing.T) (models.Title, string) {
				title := testutil.RandomTitle(voicesMap)
				tmpDir := testutil.AudioBasePath + "TestCreatePhrasesZip/" + title.Name + "/"
				err := os.MkdirAll(tmpDir, 0777)
				require.NoError(t, err)
				return title, tmpDir
			},
			buildStubs: func(ma *mock.MockcmdRunnerX) {
			},
			checkReturn: func(t *testing.T, file *os.File, err error) {
				require.NoError(t, err)
				require.FileExists(t, file.Name())
				zipFilePath := file.Name()

				reader, err := zip.OpenReader(zipFilePath)
				require.NoError(t, err)
				count := 0
				for range reader.File {
					count++
				}
				require.Equal(t, 3, count)
			},
			values:       map[string]any{"size": 3},
			stringsSlice: stringsSlice,
		},
		{
			name: "5 files",
			createTitle: func(t *testing.T) (models.Title, string) {
				title := testutil.RandomTitle(voicesMap)
				tmpDir := testutil.AudioBasePath + "TestCreatePhrasesZip/" + title.Name + "/"
				err := os.MkdirAll(tmpDir, 0777)
				require.NoError(t, err)
				return title, tmpDir
			},
			buildStubs: func(ma *mock.MockcmdRunnerX) {
			},
			checkReturn: func(t *testing.T, file *os.File, err error) {
				require.NoError(t, err)
				require.FileExists(t, file.Name())
				zipFilePath := file.Name()

				reader, err := zip.OpenReader(zipFilePath)
				require.NoError(t, err)
				count := 0
				for range reader.File {
					count++
				}
				require.Equal(t, 5, count)
			},
			values:       map[string]any{"size": 2},
			stringsSlice: stringsSlice,
		},
		{
			name: "No One File",
			createTitle: func(t *testing.T) (models.Title, string) {
				title := testutil.RandomTitle(voicesMap)
				tmpDir := testutil.AudioBasePath + "TestCreatePhrasesZip/" + title.Name + "/"
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			cmdX := mock.NewMockcmdRunnerX(ctrl)
			tc.buildStubs(cmdX)
			defer ctrl.Finish()

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/fakeurl", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			audioFile := New(cmdX)
			title, tmpDir := tc.createTitle(t)
			chunkedPhrases := slices.Chunk(tc.stringsSlice, tc.values["size"].(int))
			// remove phrasesBasePath after you have sent zipfile
			defer func(path string) {
				err := os.RemoveAll(path)
				require.NoError(t, err)
			}(tmpDir)
			osFile, err := audioFile.CreatePhrasesZip(c, chunkedPhrases, tmpDir, title.Name)
			tc.checkReturn(t, osFile, err)
		})
	}
}

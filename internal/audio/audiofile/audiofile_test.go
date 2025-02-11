package audiofile

import (
	"archive/zip"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"talkliketv.click/tltv/internal/mock"
	"talkliketv.click/tltv/internal/models"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"talkliketv.click/tltv/internal/test"
	"talkliketv.click/tltv/internal/util"
)

type audioFileTestCase struct {
	name         string
	values       map[string]any
	stringsSlice []string
	buildFile    func(*testing.T) *os.File
	checkLines   func([]string, error)
	buildStubs   func(*mock.MockcmdRunnerX)
	createTitle  func(*testing.T) (models.Title, string)
	checkReturn  func(*testing.T, *os.File, error)
}

var (
	voicesMap map[int]models.Voice
)

func TestGetLines(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}
	t.Parallel()
	testCases := []audioFileTestCase{
		{
			name: "No error",
			buildFile: func(t *testing.T) *os.File {
				return createTmpFile(
					t,
					"noerror",
					"This is the first sentence.\nThis is the second sentence.\n")
			},
			checkLines: func(lines []string, err error) {
				require.NoError(t, err)
				require.Equal(t, len(lines), 2)
			},
		},
		{
			name: "parse srt",
			buildFile: func(t *testing.T) *os.File {
				srtString := `654
				00:34:22,393 > 00:34:25,271
				¿El camión a Tepatitlán?
				Saliendo, segundo andén.

				655
				00:34:25,354 > 00:34:28,441
				Por favor, nada más debo entregar esto.
					Un segundo, por favor.

				656
				00:34:29,192 > 00:34:31,444
				Déjala pasar, mi Johnny.
					Gracias.`
				return createTmpFile(
					t,
					"parsesrt",
					srtString)
			},
			checkLines: func(lines []string, err error) {
				require.NoError(t, err)
				require.Equal(t, len(lines), 4)
			},
		},
		{
			name: "Multi newline",
			buildFile: func(t *testing.T) *os.File {
				return createTmpFile(
					t,
					"noerror",
					"This is the first sentence.\n\n\n\n\n\n\nThis is the second sentence.\n")
			},
			checkLines: func(lines []string, err error) {
				require.NoError(t, err)
				require.Equal(t, len(lines), 2)
			},
		},
		{
			name: "paragraph",
			buildFile: func(t *testing.T) *os.File {
				return createTmpFile(
					t,
					"noerror",
					"This is the first one. This is the second one. This is the third one. this is the fourth one\nThis is the fifth")
			},
			checkLines: func(lines []string, err error) {
				require.NoError(t, err)
				require.Equal(t, len(lines), 5)
			},
		},
		{
			name: "too short",
			buildFile: func(t *testing.T) *os.File {
				return createTmpFile(
					t,
					"noerror",
					"This is the. This is. This is the. this is the\nThis is the")
			},
			checkLines: func(lines []string, err error) {
				require.Errorf(t, err, "unable to parse file")
			},
		},
		{
			name: "empty file",
			buildFile: func(t *testing.T) *os.File {
				return createTmpFile(
					t,
					"noerror",
					"")
			},
			checkLines: func(lines []string, err error) {
				require.Errorf(t, err, "unable to parse file")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/fakeurl", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			file := tc.buildFile(t)
			audioFile := AudioFile{}
			stringsSlice, err := audioFile.GetLines(c, file)
			tc.checkLines(stringsSlice, err)
		})
	}
}

func TestBuildAudioInputFiles(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}
	t.Parallel()

	title := test.RandomTitle(voicesMap)
	phrase1 := test.RandomPhrase()
	phrase2 := test.RandomPhrase()
	title.TitlePhrases = []models.Phrase{phrase1, phrase2}
	title.ToPhrases = []models.Phrase{phrase1, phrase2}
	pause := test.RandomString(4)
	from := test.RandomString(4)
	to := test.RandomString(4)
	tmpDir := test.AudioBasePath + "TestBuildAudioInputFiles/" + title.Name + "/"
	fromPath := tmpDir + from
	toPath := tmpDir + to
	err := os.MkdirAll(tmpDir, 0777)
	require.NoError(t, err)
	testCases := []audioFileTestCase{
		{
			name: "No error",
			checkLines: func(lines []string, err error) {
				require.NoError(t, err)
				require.Equal(t, len(lines), 2)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/fakeurl", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			audioFile := AudioFile{}
			err := audioFile.BuildAudioInputFiles(
				c,
				title,
				pause,
				fromPath,
				toPath,
				tmpDir,
			)
			require.NoError(t, err)
			filePath := tmpDir + title.Name + "-input-01"
			require.FileExists(t, filePath)
		})
	}
}

func TestCreateMp3Zip(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}
	t.Parallel()
	testCases := []audioFileTestCase{
		{
			name: "No error",
			createTitle: func(t *testing.T) (models.Title, string) {
				title := test.RandomTitle(voicesMap)
				tmpDir := test.AudioBasePath + "TestCreateMp3ZipWithFfmpeg/" + title.Name + "/"
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
				title := test.RandomTitle(voicesMap)
				tmpDir := test.AudioBasePath + "TestCreateMp3ZipWithFfmpeg/" + title.Name + "/"
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
				title := test.RandomTitle(voicesMap)
				tmpDir := test.AudioBasePath + "TestCreatePhrasesZip/" + title.Name + "/"
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
				title := test.RandomTitle(voicesMap)
				tmpDir := test.AudioBasePath + "TestCreatePhrasesZip/" + title.Name + "/"
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
				title := test.RandomTitle(voicesMap)
				tmpDir := test.AudioBasePath + "TestCreatePhrasesZip/" + title.Name + "/"
				err := os.MkdirAll(tmpDir, 0777)
				require.NoError(t, err)
				return title, tmpDir
			},
			buildStubs: func(ma *mock.MockcmdRunnerX) {
			},
			checkReturn: func(t *testing.T, file *os.File, err error) {
				require.Error(t, ErrOneFile)
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

func TestSplitBigPhrases(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}
	t.Parallel()

	type testCase struct {
		line string
		want []string
	}

	tests := map[string]testCase{
		"combine all into one": {
			line: "This is a, sentence that I, want to all, be combined into, two big sentences.",
			want: []string{"This is a, sentence that I,", "want to all, be combined into, two big sentences."},
		},
		"too short": {
			line: "This is a",
			want: []string{},
		},
		"not too long": {
			line: "This is a, sentence that is, not too long",
			want: []string{"This is a, sentence that is, not too long"},
		},
		"too long no punctuation": {
			line: "This is a sentence that is too long but has no punctuation",
			want: []string{"This is a sentence that is too long but has no punctuation"},
		},
		"beginning short": {
			line: "This is a, sentence that I want to all be combined into one big sentences",
			want: []string{"This is a, sentence that I want to all be combined into one big sentences"},
		},
		"beginning short period": {
			line: "This is a, sentence that I want to all be combined into one big sentences.",
			want: []string{"This is a, sentence that I want to all be combined into one big sentences."},
		},
		"end short": {
			line: "This is a sentence that I want to all be combined into, one big sentence.",
			want: []string{"This is a sentence that I want to all be combined into, one big sentence."},
		},
		"middle short": {
			line: "This is a sentence that; I want to all. be combined into two big sentences.",
			want: []string{"This is a sentence that; I want to all.", "be combined into two big sentences."},
		},
		"two long": {
			line: "This is a sentence that I want to all. be combined into two big sentences.",
			want: []string{"This is a sentence that I want to all.", "be combined into two big sentences."},
		},
		"really long": {
			line: "Wow! Did you see that?! A purple penguin - yes, a purple penguin! - just roller-skated past my window... (in broad daylight!) while juggling pineapples, watermelons, and, believe it or not, rubber chickens?!? Not only that, but it was whistling a tune (sounded suspiciously like Beethoven's Fifth) and waving a little flag that said, 'Viva Las Veggies!' 🍍🍉🥒. Now, I've seen some strange things in my life, but this takes the (gluten-free) cake. I mean... really?!?",
			want: []string{"Wow! Did you see that?!", "A purple penguin - yes,", "a purple penguin! -", "just roller-skated past my window... (in broad daylight!)", "while juggling pineapples, watermelons,", "and, believe it or not,", "rubber chickens?!? Not only that,", "but it was whistling a tune (sounded suspiciously like Beethoven's Fifth)", "and waving a little flag that said, 'Viva Las Veggies!'", "🍍🍉🥒. Now,", "I've seen some strange things in my life,", "but this takes the (gluten-free) cake. I mean... really?!?"},
		},
		"no punctuation at end": {
			line: "Oh, this big, beautiful head is full of great ideas",
			want: []string{"Oh, this big, beautiful head is full of great ideas"},
		},
		"no punctuation short end": {
			line: "Oh, this big, beautiful head is full of great, ideas",
			want: []string{"Oh, this big, beautiful head is full of great, ideas"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := splitLongPhrases(tc.line)
			assert.NotNil(t, got)
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("different result: got %v, expected %v", got, tc.want)
				}
			}
		})
	}
}

func createTmpFile(t *testing.T, filename, fileString string) *os.File {
	// Create a new file
	file, err := os.Create(filename)
	require.NoError(t, err)
	defer os.Remove(filename)

	// Write to the file
	_, err = file.WriteString(fileString)
	require.NoError(t, err)
	// Ensure data is written to disk
	err = file.Sync()
	require.NoError(t, err)

	return file
}

func createFile(t *testing.T, filename, fileString string) *os.File {
	// Create a new file
	file, err := os.Create(filename)
	require.NoError(t, err)
	defer file.Close()

	// Write to the file
	_, err = file.WriteString(fileString)
	require.NoError(t, err)
	// Ensure data is written to disk
	err = file.Sync()
	require.NoError(t, err)

	return file
}

func TestMain(m *testing.M) {
	_, voicesMap = models.MakeGoogleMaps()
	flag.StringVar(&util.Test, "test", "test", "type of tests to run [unit|integration|end-to-end]")
	var projectId string
	flag.StringVar(&projectId, "project-id", "", "project id for google cloud platform that contains firestore")
	flag.Parse()
	os.Exit(m.Run())
}

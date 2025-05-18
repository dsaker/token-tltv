package audiofile

import (
	"flag"
	"os"
	"talkliketv.com/tltv/internal/interfaces"
	"talkliketv.com/tltv/internal/mock"
	"talkliketv.com/tltv/internal/testflags"
	"talkliketv.com/tltv/internal/testutil"
	"talkliketv.com/tltv/internal/util"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type audioFileTestCase struct {
	name         string
	values       map[string]any
	stringsSlice []string
	buildFile    func(*testing.T) *os.File
	checkLines   func([]string, error)
	buildStubs   func(*mock.MockcmdRunnerX)
	createTitle  func(*testing.T) (interfaces.Title, string)
	checkReturn  func(*testing.T, *os.File, error)
}

func TestMain(m *testing.M) {
	testflags.ParseFlags()
	flag.Parse()

	util.Test = testflags.TestType

	os.Exit(testflags.RunTests(m))
}

func TestGetLines(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
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
					"This is the first sentence.\nThis is the second sentence.\nThis is the third sentence.\nThis is the fourth sentence.\nThis is the fifth sentence.\n")
			},
			checkLines: func(lines []string, err error) {
				require.NoError(t, err)
				require.Equal(t, len(lines), 5)
			},
		},
		{
			name: "parsefile srt",
			buildFile: func(t *testing.T) *os.File {
				srtString := `1
00:00:01,418 --> 00:00:04,170
A continuación, se muestra
una presentación especial de Fox.

2
00:00:04,170 --> 00:00:09,342
En vivo desde el Teatro Dolby-Mucinex
en Hollywood, California.

3
00:00:09,342 --> 00:00:11,428

4
00:00:11,428 --> 00:00:16,307
Las mayores estrellas del teatro,
el cine, la política y los deportes`
				return createTmpFile(
					t,
					"parsesrt",
					srtString)
			},
			checkLines: func(lines []string, err error) {
				require.NoError(t, err)
				require.Equal(t, len(lines), 5)
			},
		},
		{
			name: "Multi newline",
			buildFile: func(t *testing.T) *os.File {
				return createTmpFile(
					t,
					"noerror",
					"This is the first sentence.\n\n\n\n\n\n\nThis is the second sentence.\nThis is the third sentence.\nThis is the fourth sentence.\nThis is the fifth sentence.\n")
			},
			checkLines: func(lines []string, err error) {
				require.NoError(t, err)
				require.Equal(t, len(lines), 5)
			},
		},
		{
			name: "paragraph",
			buildFile: func(t *testing.T) *os.File {
				return createTmpFile(
					t,
					"noerror",
					"This is the first one. This is the second one. This is the third one. this is the fourth one.\nThis is the fifth")
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
					"This is the. This is. This is the. this is the.\nThis is the")
			},
			checkLines: func(lines []string, err error) {
				require.Errorf(t, err, "unable to parsefile file")
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
				require.Errorf(t, err, "unable to parsefile file")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			file := tc.buildFile(t)
			audioFile := AudioFile{}
			stringsSlice, err := audioFile.GetLines(file)
			tc.checkLines(stringsSlice, err)
		})
	}
}

func TestBuildAudioInputFiles(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}
	t.Parallel()

	title := testutil.RandomTitle()
	phrase1 := testutil.RandomPhrase()
	phrase2 := testutil.RandomPhrase()
	title.TitlePhrases = []interfaces.Phrase{phrase1, phrase2}
	title.ToPhrases = []interfaces.Phrase{phrase1, phrase2}
	pause := testutil.RandomString(4)
	from := testutil.RandomString(4)
	to := testutil.RandomString(4)
	tmpDir := testutil.AudioBasePath + "TestBuildAudioInputFiles/" + title.Name + "/"
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
			audioFile := AudioFile{}
			err := audioFile.BuildAudioInputFiles(
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

func TestSplitBigPhrases(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
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

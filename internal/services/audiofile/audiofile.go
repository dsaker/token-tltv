package audiofile

import (
	"errors"
	"fmt"
	"iter"
	"mime/multipart"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"talkliketv.com/tltv/internal/interfaces"
	audio "talkliketv.com/tltv/internal/services/pattern"
	"talkliketv.com/tltv/internal/util"
)

// AudioPauseFilePath is a map to the silence mp3's of the embedded FS in
// internal/audio/silence/efs.go and after application startup will be stored
// at config.TTSBasePath
var AudioPauseFilePath = map[int]string{
	3:  "silence/3SecSilence.mp3",
	4:  "silence/4SecSilence.mp3",
	5:  "silence/5SecSilence.mp3",
	6:  "silence/6SecSilence.mp3",
	7:  "silence/7SecSilence.mp3",
	8:  "silence/8SecSilence.mp3",
	9:  "silence/9SecSilence.mp3",
	10: "silence/10SecSilence.mp3",
}

type AudioFileX interface {
	GetLines(multipart.File) ([]string, error)
	CreateMp3Zip(interfaces.Title, string) (*os.File, error)
	BuildAudioInputFiles(interfaces.Title, string, string, string, string) error
	CreatePhrasesZip(iter.Seq[[]string], string, string) (*os.File, error)
}

type AudioFile struct {
	cmdX cmdRunnerX
}

// cmdRunnerX creates an interface to allow for unit testing without having ffmpeg installed
type cmdRunnerX interface {
	CombinedOutput(cmd *exec.Cmd) ([]byte, error)
}

func New(cmdX cmdRunnerX) *AudioFile {
	return &AudioFile{cmdX: cmdX}
}

type RealCmdRunner struct{}

// CombinedOutput is a wrapper function for cmd.CombinedOutput() so this function
// can be interfaced for testing (ffmpeg will not have to be installed on machine
// for unit testing)
func (r *RealCmdRunner) CombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
}

// GetLines determines if the uploaded file is an srt, in paragraph form, or one phrase per
// line and then parses the file accordingly, returning a string slice containing the
// phrases to be translated
func (af *AudioFile) GetLines(f multipart.File) ([]string, error) {
	fileType, err := DetectTextFormat(f)
	if err != nil {
		return nil, err
	}

	// Reset file pointer again
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}

	lines, err := parseFileContent(f, fileType)
	if err != nil || len(lines) == 0 {
		return nil, errors.New("unable to parse file")
	}

	return util.RemoveLongStr(util.RemoveDuplicateStr(lines)), nil
}

// BuildAudioInputFiles creates a file with the filepaths of the mp3's used to construct
// the output files with ffmpeg in CreateMp3Zip
func (af *AudioFile) BuildAudioInputFiles(t interfaces.Title, pause, fromLang, toLang, tmpDir string) error {
	maxP := len(t.TitlePhrases) - 1

	pattern := audio.GetPattern(t.Pattern)
	if pattern == nil {
		return errors.New("error getting pattern from audio file")
	}
	// create chunks of []Audio pattern to split up audio files into ~15 minute lengths
	chunkedSlice := slices.Chunk(pattern, 125)
	count := 1
	last := false
	for chunk := range chunkedSlice {
		// you must pad the count for them to be read in the correct order when building mp3 files
		inputString := fmt.Sprintf("%s-input-%02d", t.Name, count)
		count++
		f, err := os.Create(tmpDir + inputString)
		if err != nil {
			return err
		}

		// start audiofile with silence
		_, err = f.WriteString(fmt.Sprintf("file '%s'\n", pause))
		if err != nil {
			return err
		}
		for _, audioTok := range chunk {
			phraseIdKey, nativeLang, err := SplitShortString(strconv.Itoa(int(audioTok)))
			if err != nil {
				return err
			}
			native := false
			if nativeLang == "1" {
				native = true
			}
			// if: we have reached the highest phrase id then this will be the last audio block
			// else if: skip if phraseId does not exist (is greater than maxP)
			// else if: native language then we add filepath for from language audio mp3
			// else: add audio filepath for to language mp3
			phraseId, err := strconv.Atoi(phraseIdKey)
			if err != nil {
				return err
			}
			if phraseId == maxP {
				last = true
			}

			if err = writeStringToFile(native, f, fromLang, toLang, phraseIdKey, pause); err != nil {
				return err
			}
		}
		// end audiofile with silence
		_, err = f.WriteString(fmt.Sprintf("file '%s'\n", pause))
		if err != nil {
			return err
		}
		// Close the file explicitly
		if err = f.Close(); err != nil {
			return err
		}
		if last {
			break
		}
	}

	return nil
}

func writeStringToFile(native bool, f *os.File, fromLang, toLang, phraseIdKey, pause string) error {
	audioString := ""
	if native {
		audioString = fmt.Sprintf("file '%s%s'\n", fromLang, phraseIdKey)
	} else {
		audioString = fmt.Sprintf("file '%s%s'\n", toLang, phraseIdKey)
	}

	pauseString := fmt.Sprintf("file '%s'\n", pause)
	_, err := f.WriteString(audioString + pauseString)
	if err != nil {
		return err
	}

	return nil
}

// SplitShortString splits a string into two parts: the first part contains all but the last character,
// the last character represents a bool indicating whether the audio input should be the native language,
// the first part is the phraseId
func SplitShortString(input string) (string, string, error) {
	if len(input) < 2 {
		return "", "", errors.New("input string must have at least two characters")
	}

	lastDigit := (input)[len(input)-1:]
	remainingDigits := (input)[:len(input)-1]

	return remainingDigits, lastDigit, nil
}

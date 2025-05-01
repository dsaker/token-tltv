package audiofile

import (
	"bufio"
	"errors"
	"fmt"
	"iter"
	"mime/multipart"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/util"

	"github.com/labstack/echo/v4"
	audio "talkliketv.click/tltv/internal/audio/pattern"
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

// endSentenceMap is a map to find the ending punctuation of a sentence
// TODO change endSentenceMap to work for any language
var (
	ErrOneFile           = errors.New("no need to zip one file")
	ErrUnableToParseFile = func(err error) error {
		return fmt.Errorf("unable to parsefile file: %s", err)
	}
	endSentenceMap = map[rune]bool{
		'!': true,
		'.': true,
		'?': true,
	}
	// Use a regular expression to match punctuation characters
	reAlpha = regexp.MustCompile(`[a-zA-Z]`)
)

type AudioFileX interface {
	GetLines(echo.Context, multipart.File) ([]string, error)
	CreateMp3Zip(echo.Context, models.Title, string) (*os.File, error)
	BuildAudioInputFiles(echo.Context, models.Title, string, string, string, string) error
	CreatePhrasesZip(echo.Context, iter.Seq[[]string], string, string) (*os.File, error)
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
func (af *AudioFile) GetLines(e echo.Context, f multipart.File) ([]string, error) {
	// get file type, options are srt, single line text or paragraph
	fileType := ""
	scanner := bufio.NewScanner(f)
	//start at the first line again
	_, err := f.Seek(0, 0)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}
	count := 0
	var line string

	// verify if file is srt
	for scanner.Scan() {
		if fileType != "" || count > 5 {
			break
		}
		line = scanner.Text()
		// if line contains ">" and doesn't contain any letters it is srt file
		if strings.Contains(line, ">") {
			if strings.Contains(line, "<font") || !reAlpha.MatchString(line) {
				fileType = "srt"
			}
		}
		count++
	}
	//start at the first line again
	_, err = f.Seek(0, 0)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}
	count = 0
	scanner = bufio.NewScanner(f)
	// verify if file is in paragraph form
	for scanner.Scan() {
		if fileType != "" || count > 4 {
			break
		}
		line = scanner.Text()
		// Split on punctuation characters
		re := regexp.MustCompile(`[.!?]`)
		result := re.Split(line, -1)
		if len(result) > 3 {
			fileType = "paragraph"
		}
		count++
	}
	// TODO verify single phrase per line form (these can be multiple sentences per line)
	_, err = f.Seek(0, 0)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}
	var stringsSlice []string
	if fileType == "srt" {
		stringsSlice = parseSrt(f)
	}
	if fileType == "paragraph" {
		stringsSlice = parseParagraph(f)
	}
	if fileType == "" {
		stringsSlice = parseSingle(f)
	}
	if len(stringsSlice) == 0 {
		return nil, errors.New("unable to parsefile file")
	}

	// remove strings longer than 150 characters and duplicates from stringsSlice
	return util.RemoveLongStr(util.RemoveDuplicateStr(stringsSlice)), nil
}

// CreateMp3Zip takes the input txt files created with BuildAudioInputFiles and uses ffmpeg
// to build an output mp3's file and the zips them into a single file to be returned to the
// requester
func (af *AudioFile) CreateMp3Zip(e echo.Context, t models.Title, tmpDir string) (*os.File, error) {
	// get a list of files from the temp directory
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}
	if len(files) == 0 {
		return nil, errors.New("no files found in CreateMp3Zip")
	}
	// create outputs folder to hold all the mp3's to zip
	outDirPath := tmpDir + "outputs"
	err = os.MkdirAll(outDirPath, 0777)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}
	for i, f := range files {
		// ffmpeg -f concat -safe 0 -i ffmpeg_input.txt -c copy output.mp3
		outputString := fmt.Sprintf("%s/%s-%d.mp3", outDirPath, t.Name, i)
		cmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", tmpDir+f.Name(), "-c", "copy", outputString) //nolint:gosec

		//Execute the command and get the output
		output, err := af.cmdX.CombinedOutput(cmd)
		if err != nil {
			e.Logger().Error(err)
			e.Logger().Error("combined output: " + string(output))
			return nil, err
		}
	}

	// add a text files of the translated phrases, this is useful studying
	if t.ToPhrases != nil {
		// Create file to write all the translated phrases to
		f, err := os.Create(outDirPath + "/" + t.Name + "-translates.txt")
		if err != nil {
			e.Logger().Error(err)
			return nil, err
		}
		defer f.Close()
		for _, text := range t.ToPhrases {
			_, err = f.WriteString(text.Text + "\n")
			if err != nil {
				e.Logger().Error(err)
				return nil, err
			}
		}
	}

	return util.CreateZipFile(e, tmpDir, t.Name, outDirPath)
}

// BuildAudioInputFiles creates a file with the filepaths of the mp3's used to construct
// the output files with ffmpeg in CreateMp3Zip
func (af *AudioFile) BuildAudioInputFiles(e echo.Context, t models.Title, pause, from, to, tmpDir string) error {
	maxP := len(t.TitlePhrases) - 1

	pattern := audio.GetPattern(t.Pattern)
	if pattern == nil {
		e.Logger().Error("error getting pattern from audio file")
		return errors.New("no pattern")
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
			e.Logger().Error(err)
			return err
		}
		defer f.Close()

		// start audiofile with silence
		_, err = f.WriteString(fmt.Sprintf("file '%s'\n", pause))
		if err != nil {
			e.Logger().Error(err)
			return err
		}
		for _, audioFloat := range chunk {
			// convert float representation of phrase id and whether speech should be native or translated
			// it is represented in /internal/pattern as "phraseId"."native_boolean" as a float32 to save space
			stringFloat := strconv.FormatFloat(float64(audioFloat), 'f', -1, 32)
			phraseNative := strings.Split(stringFloat, ".")
			// if when you split the string the length is 1 that means the float ended in .0 which means audio
			// should be translated; if length is 2 that means float ended in .1 which indicates it should be
			// native
			native := false
			if len(phraseNative) == 2 {
				native = true
			}
			phraseId, err := strconv.Atoi(phraseNative[0])
			if err != nil {
				e.Logger().Error(err)
				return err
			}
			// if: we have reached the highest phrase id then this will be the last audio block
			// else if: skip if phraseId does not exist (is greater than maxP)
			// else if: native language then we add filepath for from audio mp3
			// else: add audio filepath for language you want to learn
			if phraseId == maxP {
				last = true
			}
			if native {
				_, err = f.WriteString(fmt.Sprintf("file '%s%d'\n", from, phraseId))
				if err != nil {
					e.Logger().Error(err)
					return err
				}
				_, err = f.WriteString(fmt.Sprintf("file '%s'\n", pause))
				if err != nil {
					e.Logger().Error(err)
					return err
				}
			} else {
				_, err = f.WriteString(fmt.Sprintf("file '%s%d'\n", to, phraseId))
				if err != nil {
					e.Logger().Error(err)
					return err
				}
				_, err = f.WriteString(fmt.Sprintf("file '%s'\n", pause))
				if err != nil {
					e.Logger().Error(err)
					return err
				}
			}
		}
		// end audiofile with silence
		_, err = f.WriteString(fmt.Sprintf("file '%s'\n", pause))
		if err != nil {
			e.Logger().Error(err)
			return err
		}
		if last {
			break
		}
	}

	return nil
}

// CreatePhrasesZip creates a zipped file of txt files from the file the user uploaded if it contains
// more phrases than the limit of config.MaxNumPhrases. It takes a iter.Seq of strings and outputs them
// to files, each chunk containing config.MaxNumPhrases and than zips them up. Sending them back to the
// user
func (af *AudioFile) CreatePhrasesZip(e echo.Context, chunkedPhrases iter.Seq[[]string], tmpPath, filename string) (*os.File, error) {
	// create outputs folder to hold all the txt files to zip
	err := os.MkdirAll(tmpPath, 0777)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}
	count := 0
	for chunk := range chunkedPhrases {
		file := fmt.Sprintf("%s-phrases-%d.txt", filename, count)
		count++
		f, err := os.Create(tmpPath + file)
		if err != nil {
			e.Logger().Error(err)
			return nil, err
		}
		defer f.Close()

		for _, phrase := range chunk {
			_, err = f.WriteString(phrase + "\n")
			if err != nil {
				return nil, err
			}
		}
	}

	return util.CreateZipFile(e, tmpPath, filename, tmpPath)
}

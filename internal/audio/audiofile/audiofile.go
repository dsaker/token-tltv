package audiofile

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"io"
	"iter"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"talkliketv.click/tltv/internal/models"
	"unicode"

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
// TODO change this to work for any language
var (
	endSentenceMap = map[rune]bool{
		'!': true,
		'.': true,
		'?': true,
	}
	// Use a regular expression to match punctuation characters
	reAlpha = regexp.MustCompile(`[a-zA-Z]`)
)

const (
	minimumPhraseLength = 4
	maximumPhraseLength = 10
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
func (a *AudioFile) GetLines(e echo.Context, f multipart.File) ([]string, error) {
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
			if strings.Contains(line, "<font") {
				fileType = "srt"
			} else {
				if !reAlpha.MatchString(line) {
					fileType = "srt"
				}
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
	// TODO somehow verify single phrase per line form (these can be multiple sentences per line)
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
		return nil, errors.New("unable to parse file")
	}

	return stringsSlice, nil
}

// parseSrt takes a srt multipart file and parses it into a slice of strings
func parseSrt(f multipart.File) []string {
	var stringsSlice []string
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	var line string
	for scanner.Scan() {
		line = strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		} else if line[0] >= '0' && line[0] <= '9' {
			continue
		} else if line[0] == '[' && line[len(line)-1] == ']' {
			continue
		} else if strings.Contains(line, "<font") || strings.Contains(line, "font>") {
			continue
		} else {
			// if the next line following subtitle is not new line it is more dialogue so combine it
			scanner.Scan()
			nextLine := scanner.Text()
			if nextLine != "" {
				line = strings.ReplaceAll(line, "\n", "")
				line = line + " " + nextLine
				line = replaceFmt(line)
			} else {
				line = replaceFmt(line)
			}
		}

		phrases := splitBigPhrases(line)
		stringsSlice = append(stringsSlice, phrases...)
	}

	return stringsSlice
}

func splitBigPhrases(line string) []string {
	var splitString []string

	words := strings.Fields(line)
	// if phrase is too short don't keep it
	if len(words) <= minimumPhraseLength {
		return []string{}
	} else if len(words) < maximumPhraseLength {
		// if phrase isn't too long don't split it
		return []string{line}
	} else {
		// split into an array of strings along punctuation
		last := 0
		for i, word := range words {
			if unicode.IsPunct(rune(word[len(word)-1])) {
				nextString := ""
				for j := last; j <= i; j++ {
					nextString = nextString + words[j] + " "
				}
				splitString = append(splitString, nextString)
				last = i + 1
			}
		}
		// if last word does not end in punctuation add that string
		if last < len(words) {
			splitString = append(splitString, strings.Join(words[last:], " "))
		}
		// if long phrase has punctuation split on punctuation
		if len(splitString) > 1 {
			// combine any strings that are less than the minimumPhraseLength with the string after it
			i := 0
			for i < len(splitString)-1 {
				// if phrase is small combine it with the next one
				wordsInString := strings.Fields(splitString[i])
				if len(wordsInString) < minimumPhraseLength {
					splitString[i] = splitString[i] + " " + splitString[i+1]
					// remove the next index of split string
					splitString = append(splitString[:i+1], splitString[i+2:]...)
				} else {
					// if both combined are less than maximum than concat
					next := splitString[i] + " " + splitString[i+1]
					nextWordCount := strings.Fields(next)
					if len(nextWordCount) <= maximumPhraseLength {
						splitString[i] = next
						splitString = append(splitString[:i+1], splitString[i+2:]...)
					}
				}
				// else continue
				i++
			}

			// now check the last index and pen ultimate of the split string and combine if shorter than minimumPhraseLength
			if len(splitString) > 1 {
				lastElem := len(splitString) - 1
				lastElemCount := len(strings.Fields(splitString[lastElem]))
				penUltimateElemCount := len(strings.Fields(splitString[lastElem-1]))
				if lastElemCount < minimumPhraseLength || penUltimateElemCount < minimumPhraseLength {
					lastString := splitString[lastElem-1] + " " + splitString[lastElem]
					splitString[lastElem] = lastString
					splitString = splitString[:lastElem-1]
				}
			}
		} else {
			return []string{line}
		}
	}

	for i := range splitString {
		splitString[i] = strings.ReplaceAll(splitString[i], "  ", " ")
		splitString[i] = strings.TrimSpace(splitString[i])
	}
	return splitString
}

// parseParagraph takes a txt multipart file in paragraph form and returns a slice of strings
func parseParagraph(f multipart.File) []string {
	var stringsSlice []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Split on punctuation characters
		last := 0
		for i, c := range line {
			if i == len(line)-1 {
				sentence := strings.TrimSpace(line[last : i+1])
				last = i + 1
				words := strings.Fields(sentence)
				if len(words) > 3 {
					stringsSlice = append(stringsSlice, line)
				}
			} else if endSentenceMap[c] {
				sentence := strings.TrimSpace(line[last : i+1])
				last = i + 1
				phrases := splitBigPhrases(sentence)
				stringsSlice = append(stringsSlice, phrases...)
			}
		}
	}

	return stringsSlice
}

// parseSingle takes a txt multipart file with one phrase per line and parses it
// into a slice of strings
func parseSingle(f multipart.File) []string {
	var stringsSlice []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		phrases := splitBigPhrases(line)
		stringsSlice = append(stringsSlice, phrases...)
	}

	return stringsSlice
}

// replaceFmt is a helper function for parseSrt that replaces characters that are not part
// of the phrase like descriptions or tags
func replaceFmt(line string) string {
	// remove any characters between brackets and brackets [...] or {...} or <...>
	re := regexp.MustCompile("\\[.*?]") //nolint:gosimple
	line = re.ReplaceAllString(line, "")
	re = regexp.MustCompile("\\{.*?}") //nolint:gosimple
	line = re.ReplaceAllString(line, "")
	re = regexp.MustCompile("<.*?>")
	line = re.ReplaceAllString(line, "")
	line = strings.ReplaceAll(line, "-", "")
	line = strings.TrimSpace(line)

	return line
}

// CreateMp3Zip takes the input txt files created with BuildAudioInputFiles and uses ffmpeg
// to build an output mp3's file and the zips them into a single file to be returned to the
// requester
func (a *AudioFile) CreateMp3Zip(e echo.Context, t models.Title, tmpDir string) (*os.File, error) {
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
		output, err := a.cmdX.CombinedOutput(cmd)
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

	return createZipFile(e, tmpDir, t.Name, outDirPath)
}

// addFileToZip is a helper function for CreateMp3Zip that adds each file to
// the zip.Writer
func addFileToZip(e echo.Context, zipWriter *zip.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		e.Logger().Error(err)
		return err
	}
	defer file.Close()

	fInfo, err := file.Stat()
	if err != nil {
		e.Logger().Error(err)
		return err
	}

	header, err := zip.FileInfoHeader(fInfo)
	if err != nil {
		e.Logger().Error(err)
		return err
	}

	header.Name = filepath.Base(filename)
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		e.Logger().Error(err)
		return err
	}

	_, err = io.Copy(writer, file)
	e.Logger().Info("wrote file: %s", file.Name())
	return err
}

// BuildAudioInputFiles creates a file with the filepaths of the mp3's used to construct
// the output files with ffmpeg in CreateMp3Zip
func (a *AudioFile) BuildAudioInputFiles(e echo.Context, t models.Title, pause, from, to, tmpDir string) error {
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
	// TODO fix long silences in last file
	for chunk := range chunkedSlice {
		inputString := fmt.Sprintf("%s-input-%d", t.Name, count)
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
		for _, audioStruct := range chunk {
			// if: we have reached the highest phrase id then this will be the last audio block
			// else if: skip if phraseId does not exist (is greater than maxP)
			// else if: native language then we add filepath for from audio mp3
			// else: add audio filepath for language you want to learn
			phraseId := audioStruct.Id
			if phraseId == maxP {
				last = true
			}
			if phraseId == 0 && audioStruct.Id > 0 {
				continue
			} else if audioStruct.Native {
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
func (a *AudioFile) CreatePhrasesZip(e echo.Context, chunkedPhrases iter.Seq[[]string], tmpPath, filename string) (*os.File, error) {
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

	return createZipFile(e, tmpPath, filename, tmpPath)
}

// createZipFile takes a tmpDir which is the directory containing the files you want to zip.
// filename which is the name that you want the zipped files to have as their base name
// and outDirPath which is where the zip file will be stored and zips up the files
func createZipFile(e echo.Context, tmpDir, filename, outDirPath string) (*os.File, error) {
	// TODO add txt file of the phrases
	zipFile, err := os.Create(tmpDir + "/" + filename + ".zip")
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// get a list of files from the output directory
	files, err := os.ReadDir(outDirPath)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".zip") {
			err = addFileToZip(e, zipWriter, outDirPath+"/"+file.Name())
			if err != nil {
				return nil, err
			}
		}
	}

	return zipFile, err
}

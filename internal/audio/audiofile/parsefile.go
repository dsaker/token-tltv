package audiofile

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"mime/multipart"
	"os"
	"regexp"
	"slices"
	"strings"
	"talkliketv.click/tltv/internal/config"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/test"
	"talkliketv.click/tltv/internal/translates"
	"unicode"
)

const (
	minimumPhraseLength = 4
	maximumPhraseLength = 10
)

func ProcessFile(e echo.Context, af AudioFileX, cfg config.Config, titleName string) ([]models.Phrase, *os.File, error) {
	stringsSlice, err := FileParse(e, af, cfg.FileUploadLimit)
	if err != nil {
		return nil, nil, err
	}
	// send back zip of split files of phrase that requester can use if too big
	if len(stringsSlice) > cfg.MaxNumPhrases {
		chunkedPhrases := slices.Chunk(stringsSlice, cfg.MaxNumPhrases)
		phrasesBasePath := cfg.TTSBasePath + titleName + "/"
		// create zip of phrases files of maxNumPhrases for user to use instead of uploaded file
		zipFile, err := af.CreatePhrasesZip(e, chunkedPhrases, phrasesBasePath, titleName)
		if err != nil {
			e.Logger().Error(err)
			return nil, nil, err
		}
		return nil, zipFile, models.ErrTooManyPhrases
	}

	// make an array of phrases with id so we can match all the translates and text-to-speech
	phrases := make([]models.Phrase, len(stringsSlice))
	for i := range stringsSlice {
		phrases[i] = models.Phrase{
			ID:   i,
			Text: stringsSlice[i],
		}
	}
	return phrases, nil, nil
}

func FileParse(e echo.Context, af AudioFileX, fileUploadLimit int64) ([]string, error) {
	// Get file handler for filename, size and headers
	fh, err := e.FormFile("file_path")
	if err != nil {
		e.Logger().Error(err)
		return nil, ErrUnableToParseFile(err)
	}

	// Check if file size is too large 64000 == 8KB ~ approximately 4 pages of text
	if fh.Size > fileUploadLimit {
		rString := fmt.Sprintf("file too large (%d > %d)", fh.Size, fileUploadLimit)
		return nil, ErrUnableToParseFile(errors.New(rString))
	}
	src, err := fh.Open()
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}
	defer src.Close()

	// get an array of all the phrases from the uploaded file
	stringsSlice, err := af.GetLines(e, src)
	if err != nil {
		return nil, ErrUnableToParseFile(err)
	}

	return stringsSlice, nil
}

func ZipStringsSlice(e echo.Context, af AudioFileX, slice []string, max int, path, name string) (*os.File, error) {
	chunkedPhrases := slices.Chunk(slice, max)
	phrasesBasePath := path + name + "/"
	// create zip of phrases files of maxNumPhrases for user to use instead of uploaded file
	zipFile, err := af.CreatePhrasesZip(e, chunkedPhrases, phrasesBasePath, name)
	if err != nil {
		return nil, err
	}
	return zipFile, nil
}

// CreateAudioFromTitle is a helper function that performs the tasks shared by
// AudioFromFile and AudioFromTitle
func CreateAudioFromTitle(e echo.Context, t translates.TranslateX, af AudioFileX, title models.Title, path string) (*os.File, error) {
	// TODO if you don't want these files to persist then you need to defer removing them from calling function
	audioBasePath := path + title.Name

	fromAudioBasePath := fmt.Sprintf("%s/%d/", audioBasePath, title.FromVoiceId)
	toAudioBasePath := fmt.Sprintf("%s/%d/", audioBasePath, title.ToVoiceId)

	_, err := t.CreateTTS(e, title, title.FromVoiceId, fromAudioBasePath)
	if err != nil {
		e.Logger().Error(err)
		// if error remove all the text-to-speech created up to that point
		osErr := os.RemoveAll(audioBasePath)
		if osErr != nil {
			e.Logger().Error(osErr)
		}
		return nil, err
	}

	toPhrases, err := t.CreateTTS(e, title, title.ToVoiceId, toAudioBasePath)
	if err != nil {
		e.Logger().Error(err)
		osErr := os.RemoveAll(audioBasePath)
		if osErr != nil {
			e.Logger().Error(osErr)
		}
		return nil, err
	}
	title.ToPhrases = toPhrases

	// get pause path string to build the full pause file path
	pausePath, ok := AudioPauseFilePath[title.Pause]
	if !ok {
		e.Logger().Error(models.ErrPauseNotFound)
		return nil, models.ErrPauseNotFound
	}
	fullPausePath := path + pausePath

	// create a temporary directory for building all the files
	tmpDirPath := fmt.Sprintf("%s%s-%s/", path, title.Name, test.RandomString(4))
	err = os.MkdirAll(tmpDirPath, 0777)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}

	if err = af.BuildAudioInputFiles(e, title, fullPausePath, fromAudioBasePath, toAudioBasePath, tmpDirPath); err != nil {
		return nil, err
	}

	return af.CreateMp3Zip(e, title, tmpDirPath)
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

		phrases := splitLongPhrases(line)
		stringsSlice = append(stringsSlice, phrases...)
	}

	return stringsSlice
}

func splitLongPhrases(line string) []string {
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
				phrases := splitLongPhrases(sentence)
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
		phrases := splitLongPhrases(line)
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
	line = strings.ReplaceAll(line, "♪", "")
	line = strings.ReplaceAll(line, "\"", "")
	line = strings.TrimSpace(line)

	return line
}

package audiofile

import (
	"bufio"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"regexp"
	"slices"
	"strings"
	"talkliketv.com/tltv/internal/config"
	"talkliketv.com/tltv/internal/interfaces"
	"talkliketv.com/tltv/internal/services"
	"unicode"
)

const (
	minimumPhraseLength = 4
	maximumPhraseLength = 10
)

func parseFileContent(f multipart.File, fileType TextFormat) ([]string, error) {
	switch fileType {
	case Srt:
		return parseSrt(f), nil
	case Paragraph:
		return parseParagraph(f), nil
	case OnePhrasePerLine:
		return parseSingle(f), nil
	default:
		return nil, errors.New("file must be srt, paragraph or one phrase per line")
	}
}

func ProcessFile(fh *multipart.FileHeader, af AudioFileX, cfg config.Config, titleName string) ([]interfaces.Phrase, *os.File, error) {
	// Update FileParse to accept context
	stringsSlice, err := FileParse(fh, af, cfg.FileUploadLimit)
	if err != nil {
		return nil, nil, err
	}

	// send back zip of split files of phrase that requester can use if too big
	if len(stringsSlice) > cfg.MaxNumPhrases {
		// Chunking is CPU-bound and fast, so no need for context check here
		chunkedPhrases := slices.Chunk(stringsSlice, cfg.MaxNumPhrases)
		phrasesBasePath := cfg.TTSBasePath + titleName + "/"

		// Pass context to CreatePhrasesZip
		zipFile, err := af.CreatePhrasesZip(chunkedPhrases, phrasesBasePath, titleName)
		if err != nil {
			return nil, nil, err
		}
		return nil, zipFile, interfaces.ErrTooManyPhrases
	}

	// This operation is fast and CPU-bound, so no need for context check
	phrases := make([]interfaces.Phrase, len(stringsSlice))
	for i := range stringsSlice {
		phrases[i] = interfaces.Phrase{
			ID:   i,
			Text: stringsSlice[i],
		}
	}

	return phrases, nil, nil
}

func FileParse(fh *multipart.FileHeader, af AudioFileX, fileUploadLimit int64) ([]string, error) {
	// Check if file size is too large 64000 == 8KB ~ approximately 4 pages of text
	if fh.Size > fileUploadLimit {
		return nil, services.ErrFileTooLarge(fh.Size, fileUploadLimit)
	}
	src, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	// get an array of all the phrases from the uploaded file
	stringsSlice, err := af.GetLines(src)
	if err != nil {
		return nil, services.ErrUnableToParseFile(err)
	}

	return stringsSlice, nil
}

func ZipStringsSlice(af AudioFileX, slice []string, max int, path, name string) (*os.File, error) {
	chunkedPhrases := slices.Chunk(slice, max)
	phrasesBasePath := path + name + "/"
	// create zip of phrases files of maxNumPhrases for user to use instead of uploaded file
	zipFile, err := af.CreatePhrasesZip(chunkedPhrases, phrasesBasePath, name)
	if err != nil {
		return nil, err
	}
	return zipFile, nil
}

// parseSrt takes a srt multipart file and parses it into a slice of strings
func parseSrt(f multipart.File) []string {
	var stringsSlice []string
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	var line string
	for scanner.Scan() {
		line = strings.TrimSpace(scanner.Text())
		if line == "" ||
			(line[0] >= '0' && line[0] <= '9') ||
			(line[0] == '[' && line[len(line)-1] == ']') ||
			strings.Contains(line, "<font") ||
			strings.Contains(line, "font>") {
			continue
		}
		scanner.Scan()
		nextLine := scanner.Text()
		if nextLine != "" {
			line = strings.ReplaceAll(line, "\n", "") + " " + nextLine
			line = strings.ReplaceAll(line, "\t", "")
		}
		line = replaceFmt(line)

		phrases := splitLongPhrases(line)
		stringsSlice = append(stringsSlice, phrases...)
	}

	return stringsSlice
}

// splitLongPhrases splits a long phrase into smaller phrases based on punctuation
func splitLongPhrases(line string) []string {
	var splitString []string

	words := strings.Fields(line)
	// if phrase is too short don't keep it
	if len(words) < minimumPhraseLength {
		return []string{}
	}
	if len(words) < maximumPhraseLength {
		// if phrase isn't too long don't split it
		return []string{line}
	}
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

	for i := range splitString {
		splitString[i] = strings.ReplaceAll(splitString[i], "  ", " ")
		splitString[i] = strings.TrimSpace(splitString[i])
	}
	return splitString
}

// parseParagraph takes a txt multipart file in paragraph form and returns a slice of strings
func parseParagraph(f multipart.File) []string {
	if f == nil {
		return nil
	}

	content, err := io.ReadAll(f)
	if err != nil {
		return nil
	}

	allText := string(content)
	allLines := splitOnEndingPunctuation(allText)

	var stringsSlice []string
	for _, line := range allLines {
		phrases := splitLongPhrases(line)
		if len(phrases) > 0 {
			stringsSlice = append(stringsSlice, phrases...)
		}
	}

	return stringsSlice
}

// splitOnEndingPunctuation splits the text on sentence-ending punctuation
func splitOnEndingPunctuation(text string) []string {
	// Regular expression to split the text at sentence-ending punctuation
	re := regexp.MustCompile(`[!.?]`)

	// Split the text using the regular expression
	sentences := re.Split(text, -1)

	// Filter out empty strings
	var filtered []string
	for _, s := range sentences {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}

	return filtered
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

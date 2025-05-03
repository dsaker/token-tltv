package audiofile

import (
	"bufio"
	"errors"
	"io"
	"regexp"
	"strings"
)

// TextFormat represents different text file formats
type TextFormat int

const (
	Srt TextFormat = iota
	OnePhrasePerLine
	Paragraph
)

// DetectTextFormat determines the format of the uploaded text file
func DetectTextFormat(fileStream io.ReadSeeker) (TextFormat, error) {
	if fileStream == nil {
		return 0, errors.New("fileStream is nil")
	}

	// Check if the file is in SRT format
	reader := bufio.NewReader(fileStream)
	scanner := bufio.NewScanner(reader)

	for i := 0; i < 15 && scanner.Scan(); i++ {
		line := scanner.Text()
		if srtFormatCheck(line) {
			// Seek back to the beginning of the file
			if _, err := fileStream.Seek(0, io.SeekStart); err != nil {
				return 0, err
			}

			return Srt, nil
		}
	}

	// Seek back to the beginning of the file
	if _, err := fileStream.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}

	// Read all content to analyze line length
	reader = bufio.NewReader(fileStream)
	content, err := io.ReadAll(reader)
	if err != nil {
		return 0, err
	}

	lines := strings.FieldsFunc(string(content), func(r rune) bool {
		return r == '\r' || r == '\n'
	})

	lineCount := len(lines)
	if lineCount == 0 {
		return 0, errors.New("file is empty")
	}

	// Calculate average line length
	totalLength := 0
	for _, line := range lines {
		totalLength += len(line)
	}
	averageLineLength := float64(totalLength) / float64(lineCount)

	// Heuristics for determining format
	if lineCount > 3 && averageLineLength < 80 {
		// Seek back to the beginning of the file
		if _, err := fileStream.Seek(0, io.SeekStart); err != nil {
			return 0, err
		}
		return OnePhrasePerLine, nil
	}

	// Reset file position
	if _, err := fileStream.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}

	return Paragraph, nil
}

// srtTimestampRegex is a compiled regular expression for detecting SRT timestamps
var srtTimestampRegex = regexp.MustCompile(`\d{2}:\d{2}:\d{2},\d{3}\s-->\s\d{2}:\d{2}:\d{2},\d{3}`)

// srtFormatCheck checks if a line matches the SRT timestamp format (00:00:00,000 --> 00:00:00,000)
func srtFormatCheck(line string) bool {

	return srtTimestampRegex.MatchString(line)
}

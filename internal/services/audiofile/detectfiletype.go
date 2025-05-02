package audiofile

import (
	"bufio"
	"mime/multipart"
	"regexp"
	"strings"
)

func detectFileType(f multipart.File) (string, error) {
	// Reset file pointer
	if _, err := f.Seek(0, 0); err != nil {
		return "", err
	}

	// Check for "srt" file type
	scanner := bufio.NewScanner(f)
	for count := 0; scanner.Scan() && count <= 5; count++ {
		line := scanner.Text()
		if strings.Contains(line, ">") && (!reAlpha.MatchString(line) || strings.Contains(line, "<font")) {
			return "srt", nil
		}
	}

	// Reset file pointer again
	if _, err := f.Seek(0, 0); err != nil {
		return "", err
	}

	// Check for "paragraph" file type
	scanner = bufio.NewScanner(f)
	for count := 0; scanner.Scan() && count <= 4; count++ {
		if len(regexp.MustCompile(`[.!?]`).Split(scanner.Text(), -1)) > 3 {
			return "paragraph", nil
		}
	}

	return "", nil
}

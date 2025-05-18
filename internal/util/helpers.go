package util

import (
	"io"
	"net/http"
	"os"
)

var (
	Test = "unit"
)

const (
	StarString  = "*********************************************\n"
	metadataURL = "http://metadata.google.internal/computeMetadata/v1/instance/name"
)

// PathExists returns whether the given file or directory exists
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// RemoveDuplicateStr removes duplicate strings from a slice.
// In this application, it will be particularly useful if someone wants to use
// song lyrics to build the mp3 files.
// You cannot use a sort function because you want to keep the order.
func RemoveDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	var list []string
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

// RemoveLongStr removes strings with more than 150 characters.
func RemoveLongStr(strSlice []string) []string {
	var list []string
	for _, item := range strSlice {
		if len(item) < 150 {
			list = append(list, item)
		}
	}
	return list
}

func GetVMName() (string, error) {
	req, err := http.NewRequest(http.MethodGet, metadataURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

package util

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"google.golang.org/api/iterator"
	"io"
	"net/http"
	"os"
)

var (
	Test = "unit"
)

const (
	StarString  = "*********************************************\n"
	TokenColl   = "tokens"
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

func DeleteFirestoreCollection(ctx context.Context, client *firestore.Client, coll *firestore.CollectionRef) error {
	// delete all documents in test collection
	bulkwriter := client.BulkWriter(ctx)
	for {
		// Get a batch of documents
		iter := coll.Documents(ctx)
		numDeleted := 0

		// Iterate through the documents, adding
		// a delete operation for each one to the BulkWriter.
		for {
			doc, err := iter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return err
			}

			_, err = bulkwriter.Delete(doc.Ref)
			if err != nil {
				return err
			}
			numDeleted++
		}

		// If there are no documents to delete, the process is over.
		if numDeleted == 0 {
			bulkwriter.End()
			break
		}

		bulkwriter.Flush()
	}
	return nil
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

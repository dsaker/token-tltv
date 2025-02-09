package util

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"github.com/go-playground/form/v4"
	"google.golang.org/api/iterator"
	"net/http"
	"os"
)

var (
	Integration = false
)

const (
	StarString = "************************************************************\n"
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

// DecodePostForm  helper method. The second parameter here, dst,
// is the target destination that we want to decode the form data into.
func DecodePostForm(r *http.Request, dst any, fd *form.Decoder) error {
	// Call Decode() on our decoder instance, passing the target destination as
	// the first parameter.
	err := fd.Decode(dst, r.PostForm)
	if err != nil {
		// If we try to use an invalid target destination, the Decode() method
		// will return an error with the type *form.InvalidDecoderError.We use
		// errors.As() to check for this and raise a panic rather than returning
		// the error.
		var invalidDecoderError *form.InvalidDecoderError

		if errors.As(err, &invalidDecoderError) {
			panic(err)
		}

		// For all other errors, we return them as normal.
		return err
	}

	return nil
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

		// If there are no documents to delete,
		// the process is over.
		if numDeleted == 0 {
			bulkwriter.End()
			break
		}

		bulkwriter.Flush()
	}
	return nil
}

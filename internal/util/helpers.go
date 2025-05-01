package util

import (
	"archive/zip"
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"github.com/go-playground/form/v4"
	"github.com/labstack/echo/v4"
	"google.golang.org/api/iterator"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

// CreateZipFile takes a tmpDir which is the directory containing the files you want to zip.
// filename which is the name that you want the zipped files to have as their base name
// and outDirPath which is where the zip file will be stored and zips up the files
func CreateZipFile(e echo.Context, tmpDir, filename, outDirPath string) (*os.File, error) {
	// TODO add txt file of the phrases
	zipFile, err := os.Create(tmpDir + filename + ".zip")
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

package audiofile

import (
	"archive/zip"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"io"
	"iter"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"talkliketv.click/tltv/internal/models"
)

// CreateMp3Zip generates mp3 files from input text files and zips them into a single file.
func (af *AudioFile) CreateMp3Zip(e echo.Context, t models.Title, tmpDir string) (*os.File, error) {
	files, err := os.ReadDir(tmpDir)
	if err != nil || len(files) == 0 {
		e.Logger().Error(err)
		return nil, errors.New("no files found in CreateMp3Zip")
	}

	outDirPath := tmpDir + "outputs"
	if err := os.MkdirAll(outDirPath, 0777); err != nil {
		e.Logger().Error(err)
		return nil, err
	}

	for i, f := range files {
		outputPath := fmt.Sprintf("%s/%s-%d.mp3", outDirPath, t.Name, i)
		cmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", tmpDir+f.Name(), "-c", "copy", outputPath) //nolint:gosec
		if output, err := af.cmdX.CombinedOutput(cmd); err != nil {
			e.Logger().Error(err)
			e.Logger().Error("combined output: " + string(output))
			return nil, err
		}
	}

	if t.ToPhrases != nil {
		if err := writeTranslatedPhrases(outDirPath, t.Name, t.ToPhrases, e); err != nil {
			return nil, err
		}
	}

	return createZipFile(e, tmpDir, t.Name, outDirPath)
}

// writeTranslatedPhrases writes translated phrases to a text file.
func writeTranslatedPhrases(outDirPath, title string, phrases []models.Phrase, e echo.Context) error {
	file, err := os.Create(fmt.Sprintf("%s/%s-translates.txt", outDirPath, title))
	if err != nil {
		e.Logger().Error(err)
		return err
	}
	defer file.Close()

	for _, phrase := range phrases {
		if _, err := file.WriteString(phrase.Text + "\n"); err != nil {
			e.Logger().Error(err)
			return err
		}
	}
	return nil
}

// CreatePhrasesZip creates a zipped file of txt files from the file the user uploaded if it contains
// more phrases than the limit of config.MaxNumPhrases. It takes a iter.Seq of strings and outputs them
// to files, each chunk containing config.MaxNumPhrases and than zips them up.
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

		for _, phrase := range chunk {
			_, err = f.WriteString(phrase + "\n")
			if err != nil {
				return nil, err
			}
		}

		// Close the file explicitly
		if err = f.Close(); err != nil {
			e.Logger().Error("failed to close file: %v", err)
			return nil, err
		}
	}

	return createZipFile(e, tmpPath, filename, tmpPath)
}

// CreateZipFile takes a tmpDir which is the directory containing the files you want to zip.
// filename which is the name that you want the zipped files to have as their base name
// and outDirPath which is where the zip file will be stored and zips up the files
func createZipFile(e echo.Context, tmpDir, filename, outDirPath string) (*os.File, error) {
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

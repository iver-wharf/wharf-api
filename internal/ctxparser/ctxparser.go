package ctxparser

import (
	"errors"
	"io/ioutil"
	"mime/multipart"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-core/pkg/logger"
)

var log = logger.NewScoped("CTX-PARSER")

// File represents a file parsed from multipart form data.
type File struct {
	// Name is the key value of the file in the multipart form data.
	Name string
	// FileName is the name of the file.
	FileName string
	// Data is the binary data of the file.
	Data []byte
}

// ParseMultipartFormDataFiles parses one or more files from a gin.Context's
// multipart form data's specified File field entry.
func ParseMultipartFormDataFiles(c *gin.Context, formFileFieldKey string) ([]File, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, err
	}

	var files []File
	if fhs, ok := form.File[formFileFieldKey]; ok {
		for _, fh := range fhs {
			data, err := readMultipartFileData(fh)
			if err != nil {
				return nil, err
			}

			files = append(files, File{
				Name:     formFileFieldKey,
				FileName: fh.Filename,
				Data:     data,
			})
		}
	}

	return files, nil
}

func readMultipartFileData(fh *multipart.FileHeader) ([]byte, error) {
	if fh == nil {
		return nil, errors.New("fh argument was nil")
	}

	f, err := fh.Open()
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := f.Close(); err != nil {
			log.Error().
				WithError(err).
				Message("Failed to close multipart form request body file handle.")
		}
	}()

	data, err := ioutil.ReadAll(f)
	return data, err
}

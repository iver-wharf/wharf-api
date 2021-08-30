package ctxparser

import (
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"github.com/iver-wharf/wharf-core/pkg/logger"
)

var log = logger.NewScoped("REQUEST")

// File represents a file parsed from multipart form data.
type File struct {
	// Name is the key value of the file in the multipart form data.
	Name string
	// FileName is the name of the file.
	FileName string
	// Data is the binary data of the file.
	Data []byte
}

// ParseMultipartFormData parses multipart form data files from a gin.Context.
func ParseMultipartFormData(c *gin.Context, buildID uint) ([]File, bool) {
	form, err := c.MultipartForm()
	if err != nil {
		ginutil.WriteMultipartFormReadError(c, err,
			fmt.Sprintf("Failed reading multipart-form content from request body when uploading new"+
				" artifact for build with ID %d.", buildID))
		return nil, false
	}

	var files []File
	for k := range form.File {
		if fhs := form.File[k]; len(fhs) > 0 {
			data, err := readMultipartFileData(fhs[0])
			if err != nil {
				ginutil.WriteMultipartFormReadError(c, err,
					fmt.Sprintf("Failed reading multipart-form's file data from request body when uploading"+
						" new artifact for build with ID %d.", buildID))
				return nil, false
			}

			files = append(files, File{
				Name:     k,
				FileName: fhs[0].Filename,
				Data:     data,
			})
		}
	}

	return files, true
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

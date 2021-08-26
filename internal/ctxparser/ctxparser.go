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

// ParamArtifactAndBuildID is a helper function that parses the params
// artifactId and buildid from the gin.Context.
func ParamArtifactAndBuildID(c *gin.Context) (artifactID, buildID uint, ok bool) {
	if artifactID, ok = ginutil.ParseParamUint(c, "artifactId"); ok {
		buildID, ok = ginutil.ParseParamUint(c, "buildid")
	}
	return
}

// ParamBuildIDAndFiles is a helper function that parses the param
// buildid and multipart form data files from the gin.Context.
func ParamBuildIDAndFiles(c *gin.Context) (buildID uint, files []File, ok bool) {
	if buildID, ok = ginutil.ParseParamUint(c, "buildid"); ok {
		files, ok = parseMultipartFormData(c, buildID)
	}
	return
}

func parseMultipartFormData(c *gin.Context, buildID uint) ([]File, bool) {
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

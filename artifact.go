package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"gorm.io/gorm"
)

type artifactModule struct {
	Database *gorm.DB
}

type testRun struct {
	XMLName       xml.Name      `xml:"TestRun"`
	ResultSummary resultSummary `xml:"ResultSummary"`
}

type resultSummary struct {
	XMLName  xml.Name `xml:"ResultSummary"`
	Counters counters `xml:"Counters"`
}

type counters struct {
	XMLName xml.Name `xml:"Counters"`
	Passed  int      `xml:"passed,attr"`
	Failed  int      `xml:"failed,attr"`
}

// TestStatus is an enum of different states a test run or test summary can be
// in.
type TestStatus string

const (
	// TestStatusSuccess means the test run or test summary passed, or in the
	// case that there are multiple tests then that there are no failing tests
	// and at least one successful test.
	TestStatusSuccess TestStatus = "Success"

	// TestStatusFailed means the test run or test summary failed, or in the
	// case that there are multiple tests then that at least one test failed.
	TestStatusFailed TestStatus = "Failed"

	// TestStatusNoTests means the test run or test summary is inconclusive,
	// where there are neither any passing nor failing tests.
	TestStatusNoTests TestStatus = "No tests"
)

// TestsResults holds how many builds has passed and failed. A test result has
// the status of "Failed" if there are any failed tests, "Success" if there are
// any passing tests and no failed tests, and "No tests" if there are no failed
// nor passing tests.
type TestsResults struct {
	Passed int        `json:"passed"`
	Failed int        `json:"failed"`
	Status TestStatus `json:"status" enums:"Success,Failed,No tests"`
}

func (m artifactModule) Register(g *gin.RouterGroup) {
	g.GET("/artifacts", m.getBuildArtifactsHandler)
	g.GET("/artifact/:artifactId", m.getBuildArtifactHandler)
	g.GET("/tests-results", m.getBuildTestsResultsHandler)
	g.POST("/artifact", m.postBuildArtifactHandler)
}

// getBuildArtifactsHandler godoc
// @summary Get list of build artifacts
// @tags artifact
// @param buildid path int true "Build ID"
// @success 200 {array} Artifact
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/artifacts [get]
func (m artifactModule) getBuildArtifactsHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}

	var artifacts []Artifact
	err := m.Database.
		Where(&Artifact{BuildID: uint(buildID)}).
		Find(&artifacts).
		Error
	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching artifacts for build with ID %d from database.",
			buildID))
		return
	}

	c.JSON(http.StatusOK, artifacts)
}

// getBuildArtifactHandler godoc
// @summary Get build artifact
// @tags artifact
// @param buildid path int true "Build ID"
// @param artifactId path int true "Artifact ID"
// @success 200 {file} string "OK"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Artifact not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/artifact/{artifactId} [get]
func (m artifactModule) getBuildArtifactHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}
	artifactID, ok := ginutil.ParseParamUint(c, "artifactId")
	if !ok {
		return
	}

	var artifact Artifact
	err := m.Database.
		Where(&Artifact{
			BuildID:    buildID,
			ArtifactID: artifactID}).
		Order(artifactColumnArtifactID + " DESC").
		First(&artifact).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ginutil.WriteDBNotFound(c, fmt.Sprintf(
			"Artifact with ID %d was not found on build with ID %d.",
			artifactID, buildID))
		return
	} else if err != nil {
		ginutil.WriteBodyReadError(c, err, fmt.Sprintf(
			"Failed fetching artifact with ID %d on build with ID %d.",
			artifactID, buildID))
		return
	}

	extension := filepath.Ext(artifact.FileName)
	mimeType := mime.TypeByExtension(extension)
	disposition := fmt.Sprintf("attachment; filename=\"%s\"", artifact.FileName)

	c.Header("Content-Disposition", disposition)
	c.Data(http.StatusOK, mimeType, artifact.Data)
}

// getBuildTestsResultsHandler godoc
// @summary Get build tests results from .trx files
// @tags artifact
// @param buildid path int true "Build ID"
// @success 200 {object} TestsResults "OK"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/tests-results [get]
func (m artifactModule) getBuildTestsResultsHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}

	var testRunFiles []Artifact

	err := m.Database.
		Where(&Artifact{BuildID: uint(buildID)}).
		Where(artifactColumnFileName+" LIKE ?", "%.trx").
		Find(&testRunFiles).
		Error
	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching test result artifacts for build with ID %d from database.",
			buildID))
		return
	}

	var results TestsResults
	var run testRun

	for _, testRunFile := range testRunFiles {
		xml.Unmarshal(testRunFile.Data, &run)
		results.Passed += run.ResultSummary.Counters.Passed
		results.Failed += run.ResultSummary.Counters.Failed
	}

	if results.Failed == 0 && results.Passed == 0 {
		results.Status = TestStatusNoTests
	} else if results.Failed == 0 {
		results.Status = TestStatusSuccess
	} else {
		results.Status = TestStatusFailed
	}

	c.JSON(http.StatusOK, results)
}

// postBuildArtifactHandler godoc
// @summary Post build artifact
// @tags artifact
// @accept multipart/form-data
// @param buildid path int true "Build ID"
// @param file formData file true "build artifact file"
// @success 200 "OK"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Artifact not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/artifact [post]
func (m artifactModule) postBuildArtifactHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		ginutil.WriteMultipartFormReadError(c, err, fmt.Sprintf(
			"Failed reading multipart-form content from request body when uploading new artifact for build with ID %d.",
			buildID))
		return
	}

	for k := range form.File {
		if fhs := form.File[k]; len(fhs) > 0 {
			f, err := fhs[0].Open()
			if err != nil {
				ginutil.WriteMultipartFormReadError(c, err, fmt.Sprintf(
					"Failed with starting to read file content from multipart form request body when uploading new artifact for build with ID %d.",
					buildID))
				return
			}

			data, err := ioutil.ReadAll(f)
			if err != nil {
				ginutil.WriteMultipartFormReadError(c, err, fmt.Sprintf(
					"Failed reading file content from multipart form request body when uploading new artifact for build with ID %d.",
					buildID))
				return
			}

			artifact := Artifact{
				Data:     data,
				Name:     k,
				FileName: fhs[0].Filename,
				BuildID:  buildID,
			}

			err = m.Database.Create(&artifact).Error
			if err != nil {
				ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
					"Failed saving artifact with name %q for build with ID %d in database.",
					artifact.FileName, buildID))
				return
			}

			log.Info().
				WithString("filename", artifact.Name).
				WithUint("build", buildID).
				WithUint("artifact", artifact.ArtifactID).
				Message("File saved as artifact")
		}
	}

	c.Status(http.StatusCreated)
}

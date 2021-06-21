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
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ArtifactModule struct {
	Database *gorm.DB
}

type TestRun struct {
	XMLName       xml.Name      `xml:"TestRun"`
	ResultSummary ResultSummary `xml:"ResultSummary"`
}

type ResultSummary struct {
	XMLName  xml.Name `xml:"ResultSummary"`
	Counters Counters `xml:"Counters"`
}

type Counters struct {
	XMLName xml.Name `xml:"Counters"`
	Passed  int      `xml:"passed,attr"`
	Failed  int      `xml:"failed,attr"`
}

type TestStatus string

const (
	TestStatusSuccess TestStatus = "Success"
	TestStatusFailed  TestStatus = "Failed"
	TestStatusNoTests TestStatus = "No tests"
)

type TestsResults struct {
	Passed int        `json:"passed"`
	Failed int        `json:"failed"`
	Status TestStatus `json:"status" enums:"Success,Failed,No tests"`
}

func (m ArtifactModule) Register(g *gin.RouterGroup) {
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
func (m ArtifactModule) getBuildArtifactsHandler(c *gin.Context) {
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
func (m ArtifactModule) getBuildArtifactHandler(c *gin.Context) {
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
func (m ArtifactModule) getBuildTestsResultsHandler(c *gin.Context) {
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

	var testsResults TestsResults
	var testRun TestRun

	for _, testRunFile := range testRunFiles {
		xml.Unmarshal(testRunFile.Data, &testRun)
		testsResults.Passed += testRun.ResultSummary.Counters.Passed
		testsResults.Failed += testRun.ResultSummary.Counters.Failed
	}

	if testsResults.Failed == 0 && testsResults.Passed == 0 {
		testsResults.Status = TestStatusNoTests
	} else if testsResults.Failed == 0 {
		testsResults.Status = TestStatusSuccess
	} else {
		testsResults.Status = TestStatusFailed
	}

	c.JSON(http.StatusOK, testsResults)
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
func (m ArtifactModule) postBuildArtifactHandler(c *gin.Context) {
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

			log.
				WithFields(log.Fields{"filename": artifact.Name, "build": buildID, "artifact": artifact.ArtifactID}).
				Infoln("File saved as artifact")
		}
	}

	c.Status(http.StatusCreated)
}

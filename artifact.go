package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/internal/ctxparser"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
	"github.com/iver-wharf/wharf-api/pkg/modelconv"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"gorm.io/gorm"
)

type artifactModule struct {
	Database *gorm.DB
}

func (m artifactModule) Register(g *gin.RouterGroup) {
	g.GET("/artifacts", m.getBuildArtifactListHandler)
	g.GET("/artifact/:artifactId", m.getBuildArtifactHandler)
	g.POST("/artifact", m.createBuildArtifactHandler)
	// deprecated
	g.GET("/tests-results", m.getBuildTestResultListHandler)
}

// getBuildArtifactListHandler godoc
// @id getBuildArtifactList
// @summary Get list of build artifacts
// @tags artifact
// @param buildId path uint true "Build ID"
// @success 200 {object} []response.Artifact
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildId}/artifacts [get]
func (m artifactModule) getBuildArtifactListHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildId")
	if !ok {
		return
	}

	dbArtifacts := []database.Artifact{}
	err := m.Database.
		Where(&database.Artifact{BuildID: buildID}).
		Find(&dbArtifacts).
		Error
	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching artifacts for build with ID %d from database.",
			buildID))
		return
	}

	resArtifacts := make([]response.Artifact, len(dbArtifacts))
	for i, dbArtifact := range dbArtifacts {
		resArtifacts[i] = modelconv.DBArtifactToResponse(dbArtifact)
	}

	c.JSON(http.StatusOK, resArtifacts)
}

// getBuildArtifactHandler godoc
// @id getBuildArtifact
// @summary Get build artifact
// @tags artifact
// @param buildId path uint true "Build ID"
// @param artifactId path uint true "Artifact ID"
// @success 200 {file} string "OK"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Artifact not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildId}/artifact/{artifactId} [get]
func (m artifactModule) getBuildArtifactHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildId")
	if !ok {
		return
	}

	artifactID, ok := ginutil.ParseParamUint(c, "artifactId")
	if !ok {
		return
	}

	var dbArtifact database.Artifact
	err := m.Database.
		Where(&database.Artifact{
			BuildID:    buildID,
			ArtifactID: artifactID}).
		Order(database.ArtifactColumns.ArtifactID + " DESC").
		First(&dbArtifact).
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

	extension := filepath.Ext(dbArtifact.FileName)
	mimeType := mime.TypeByExtension(extension)
	disposition := fmt.Sprintf("attachment; filename=\"%s\"", dbArtifact.FileName)

	c.Header("Content-Disposition", disposition)
	c.Data(http.StatusOK, mimeType, dbArtifact.Data)
}

// createBuildArtifactHandler godoc
// @id createBuildArtifact
// @summary Post build artifact
// @tags artifact
// @accept multipart/form-data
// @param buildId path uint true "Build ID"
// @param files formData file true "Build artifact file"
// @success 201 "Added new artifacts"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Artifact not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildId}/artifact [post]
func (m artifactModule) createBuildArtifactHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildId")
	if !ok {
		return
	}

	files, err := ctxparser.ParseMultipartFormDataFiles(c, "files")
	if err != nil {
		ginutil.WriteMultipartFormReadError(c, err,
			fmt.Sprintf("Failed reading multipart-form's file data from request body when uploading"+
				" new artifact for build with ID %d.", buildID))
		return
	}

	_, ok = createArtifacts(c, m.Database, files, buildID)
	if !ok {
		return
	}

	c.Status(http.StatusCreated)
}

// getBuildTestResultListHandler godoc
// @id getBuildTestResultList
// @deprecated
// @summary Get build tests results from .trx files.
// @description Deprecated, /build/{buildid}/test-result/list-summary should be used instead.
// @tags artifact
// @param buildId path uint true "Build ID"
// @success 200 {object} response.TestsResults
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildId}/tests-results [get]
func (m artifactModule) getBuildTestResultListHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildId")
	if !ok {
		return
	}

	testRunFiles := []database.Artifact{}

	err := m.Database.
		Where(&database.Artifact{BuildID: buildID}).
		Where(database.ArtifactColumns.FileName+" LIKE ?", "%.trx").
		Find(&testRunFiles).
		Error
	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching test result artifacts for build with ID %d from database.",
			buildID))
		return
	}

	var resResults response.TestsResults
	var run trxTestRun

	for _, testRunFile := range testRunFiles {
		xml.Unmarshal(testRunFile.Data, &run)
		resResults.Passed += run.ResultSummary.Counters.Passed
		resResults.Failed += run.ResultSummary.Counters.Failed
	}

	if resResults.Failed == 0 && resResults.Passed == 0 {
		resResults.Status = response.TestStatusNoTests
	} else if resResults.Failed == 0 {
		resResults.Status = response.TestStatusSuccess
	} else {
		resResults.Status = response.TestStatusFailed
	}

	c.JSON(http.StatusOK, resResults)
}

func createArtifacts(c *gin.Context, db *gorm.DB, files []ctxparser.File, buildID uint) ([]database.Artifact, bool) {
	dbArtifacts := make([]database.Artifact, len(files))
	for idx, f := range files {
		artifactPtr := &dbArtifacts[idx]
		artifactPtr.Data = f.Data
		artifactPtr.Name = f.Name
		artifactPtr.FileName = f.FileName
		artifactPtr.BuildID = buildID

		err := db.Create(artifactPtr).Error
		if err != nil {
			ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
				"Failed saving artifact with name %q for build with ID %d in database.",
				artifactPtr.FileName, buildID))
			return dbArtifacts, false
		}

		log.Debug().
			WithString("filename", artifactPtr.FileName).
			WithUint("build", buildID).
			WithUint("artifact", artifactPtr.ArtifactID).
			Message("File saved as artifact")
	}
	return dbArtifacts, true
}

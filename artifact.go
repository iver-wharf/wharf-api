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
	"github.com/iver-wharf/wharf-api/internal/wherefields"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
	"github.com/iver-wharf/wharf-api/pkg/modelconv"
	"github.com/iver-wharf/wharf-api/pkg/orderby"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"gorm.io/gorm"
)

type artifactModule struct {
	Database *gorm.DB
}

func (m artifactModule) Register(g *gin.RouterGroup) {
	g.GET("/artifact", m.getBuildArtifactListHandler)
	g.GET("/artifact/:artifactId", m.getBuildArtifactHandler)
	g.POST("/artifact", m.createBuildArtifactHandler)
	// deprecated
	g.GET("/tests-results", m.getBuildTestResultListHandler)
}

var artifactJSONToColumns = map[string]database.SafeSQLName{
	response.ArtifactJSONFields.ArtifactID: database.ArtifactColumns.ArtifactID,
	response.ArtifactJSONFields.Name:       database.ArtifactColumns.Name,
	response.ArtifactJSONFields.FileName:   database.ArtifactColumns.FileName,
}

var defaultGetArtifactsOrderBy = orderby.Column{Name: database.ArtifactColumns.ArtifactID, Direction: orderby.Desc}

// getBuildArtifactListHandler godoc
// @id getBuildArtifactList
// @summary Get list of build artifacts
// @description List all build artifacts, or a window of build artifacts using the `limit` and `offset` query parameters. Allows optional filtering parameters.
// @description Verbatim filters will match on the entire string used to find exact matches,
// @description while the matching filters are meant for searches by humans where it tries to find soft matches and is therefore inaccurate by nature.
// @description Added in TODO.
// @tags artifact
// @param buildId path uint true "Build ID" minimum(0)
// @param limit query int false "Number of results to return. No limiting is applied if empty (`?limit=`) or non-positive (`?limit=0`). Required if `offset` is used." default(100)
// @param offset query int false "Skipped results, where 0 means from the start." minimum(0) default(0)
// @param orderby query []string false "Sorting orders. Takes the property name followed by either 'asc' or 'desc'. Can be specified multiple times for more granular sorting. Defaults to `?orderby=artifactId desc`"
// @param name query string false "Filter by verbatim artifact name."
// @param fileName query string false "Filter by verbatim artifact file name."
// @param nameMatch query string false "Filter by matching artifact name. Cannot be used with `name`."
// @param fileNameMatch query string false "Filter by matching artifact file name. Cannot be used with `fileName`."
// @param match query string false "Filter by matching on any supported fields."
// @success 200 {object} response.PaginatedArtifacts
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildId}/artifact [get]
func (m artifactModule) getBuildArtifactListHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildId")
	if !ok {
		return
	}
	var params = struct {
		commonGetQueryParams

		Name     *string `form:"name"`
		FileName *string `form:"fileName"`

		NameMatch     *string `form:"nameMatch" binding:"excluded_with=Name"`
		FileNameMatch *string `form:"fileNameMatch" binding:"excluded_with=FileName"`

		Match *string `form:"match"`
	}{
		commonGetQueryParams: defaultCommonGetQueryParams,
	}
	if !bindCommonGetQueryParams(c, &params) {
		return
	}
	orderBySlice, ok := parseCommonOrderBySlice(c, params.OrderBy, artifactJSONToColumns)
	if !ok {
		return
	}

	var where wherefields.Collection
	where.AddFieldName(database.ArtifactFields.BuildID)

	query := m.Database.
		Clauses(orderBySlice.ClauseIfNone(defaultGetArtifactsOrderBy)).
		Where(&database.Artifact{
			BuildID:  buildID,
			Name:     where.String(database.ArtifactFields.Name, params.Name),
			FileName: where.String(database.ArtifactFields.FileName, params.FileName),
		}, where.NonNilFieldNames()...).
		Scopes(
			whereLikeScope(map[database.SafeSQLName]*string{
				database.ArtifactColumns.Name:     params.NameMatch,
				database.ArtifactColumns.FileName: params.FileNameMatch,
			}),
			whereAnyLikeScope(
				params.Match,
				database.ArtifactColumns.Name,
				database.ArtifactColumns.FileName,
			),
		)

	var dbArtifacts []database.Artifact
	var totalCount int64
	err := findDBPaginatedSliceAndTotalCount(query, params.Limit, params.Offset, &dbArtifacts, &totalCount)
	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching list of artifacts for build with ID %d from database.",
			buildID,
		))
		return
	}

	c.JSON(http.StatusOK, response.PaginatedArtifacts{
		List:       modelconv.DBArtifactsToResponses(dbArtifacts),
		TotalCount: totalCount,
	})
}

// getBuildArtifactHandler godoc
// @id getBuildArtifact
// @summary Get build artifact
// @description Added in TODO.
// @tags artifact
// @param buildId path uint true "Build ID" minimum(0)
// @param artifactId path uint true "Artifact ID" minimum(0)
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
// @description Added in TODO.
// @tags artifact
// @accept multipart/form-data
// @param buildId path uint true "Build ID" minimum(0)
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
// @description Added in TODO.
// @tags artifact
// @param buildId path uint true "Build ID" minimum(0)
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

package main

import (
	"errors"
	"fmt"
	"io"
	"time"

	"net/http"

	"github.com/dustin/go-broadcast"
	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/internal/wherefields"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/request"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
	"github.com/iver-wharf/wharf-api/pkg/modelconv"
	"github.com/iver-wharf/wharf-api/pkg/orderby"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"gorm.io/gorm"
)

type buildModule struct {
	Database *gorm.DB
}

func (m buildModule) Register(g *gin.RouterGroup) {
	build := g.Group("/build")
	{
		build.GET("", m.getBuildListHandler)

		buildByID := build.Group("/:buildId")
		{
			buildByID.GET("", m.getBuildHandler)
			buildByID.PUT("", m.updateBuildHandler)
			buildByID.POST("/log", m.createBuildLogHandler)
			buildByID.GET("/log", m.getBuildLogListHandler)
			buildByID.GET("/stream", m.streamBuildLogHandler)

			artifacts := artifactModule{m.Database}
			artifacts.Register(buildByID)

			buildTestResults := buildTestResultModule{m.Database}
			buildTestResults.Register(buildByID)
		}
	}
}

var buildChannels = make(map[uint]broadcast.Broadcaster)

func openListener(buildID uint) chan interface{} {
	listener := make(chan interface{})
	build(buildID).Register(listener)
	return listener
}

func closeListener(buildID uint, listener chan interface{}) {
	build(buildID).Unregister(listener)
	close(listener)
}

func build(buildID uint) broadcast.Broadcaster {
	b, ok := buildChannels[buildID]
	if !ok {
		b = broadcast.NewBroadcaster(10)
		buildChannels[buildID] = b
	}
	return b
}

// getBuildHandler godoc
// @id getBuild
// @summary Finds build by build ID
// @tags build
// @param buildId path uint true "build id" minimum(0)
// @success 200 {object} response.Build
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Build not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildId} [get]
func (m buildModule) getBuildHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildId")
	if !ok {
		return
	}

	dbBuild, err := m.getBuild(buildID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ginutil.WriteDBNotFound(c, fmt.Sprintf(
			"Build with ID %d was not found.",
			buildID))
		return
	} else if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching build with ID %d from database.",
			buildID))
		return
	}

	resBuild := modelconv.DBBuildToResponse(dbBuild)
	c.JSON(http.StatusOK, resBuild)
}

var buildJSONToColumns = map[string]string{
	response.BuildJSONFields.BuildID:     database.BuildColumns.BuildID,
	response.BuildJSONFields.Environment: database.BuildColumns.Environment,
	response.BuildJSONFields.CompletedOn: database.BuildColumns.CompletedOn,
	response.BuildJSONFields.ScheduledOn: database.BuildColumns.ScheduledOn,
	response.BuildJSONFields.StartedOn:   database.BuildColumns.StartedOn,
	response.BuildJSONFields.Stage:       database.BuildColumns.Stage,
	response.BuildJSONFields.StatusID:    database.BuildColumns.StatusID,
	response.BuildJSONFields.IsInvalid:   database.BuildColumns.IsInvalid,
}

var defaultGetBuildsOrderBy = orderby.Column{Name: database.BuildColumns.BuildID, Direction: orderby.Desc}

// getBuildListHandler godoc
// @id getBuildList
// @summary Get slice of builds.
// @description List all builds, or a window of builds using the `limit` and `offset` query parameters. Allows optional filtering parameters.
// @description Verbatim filters will match on the entire string used to find exact matches,
// @description while the matching filters are meant for searches by humans where it tries to find soft matches and is therefore inaccurate by nature.
// @tags build
// @param limit query int false "Number of results to return. No limit if unset or non-positive. Required if `offset` is used." default(100)
// @param offset query int false "Skipped results, where 0 means from the start." minimum(0) default(0)
// @param orderby query []string false "Sorting orders. Takes the property name followed by either 'asc' or 'desc'. Can be specified multiple times for more granular sorting. Defaults to `?orderby=buildId desc`"
// @param projectId query uint false "Filter by project ID."
// @param scheduledAfter query string false "Filter by builds with scheduled date later than value." format(date-time)
// @param scheduledBefore query string false "Filter by builds with scheduled date earlier than value." format(date-time)
// @param finishedAfter query string false "Filter by builds with finished date later than value." format(date-time)
// @param finishedBefore query string false "Filter by builds with finished date earlier than value." format(date-time)
// @param environment query string false "Filter by verbatim build environment."
// @param gitBranch query string false "Filter by verbatim build Git branch."
// @param stage query string false "Filter by verbatim build stage."
// @param isInvalid query bool false "Filter by build's valid/invalid state."
// @param status query string false "Filter by build status name" enums(Scheduling,Running,Completed,Failed)
// @param statusId query int false "Filter by build status ID. Cannot be used with `status`." enums(0,1,2,3)
// @param environmentMatch query string false "Filter by matching build environment. Cannot be used with `environment`."
// @param gitBranchMatch query string false "Filter by matching build Git branch. Cannot be used with `gitBranch`."
// @param stageMatch query string false "Filter by matching build stage. Cannot be used with `stage`."
// @param match query string false "Filter by matching on any supported fields."
// @success 200 {object} response.PaginatedBuilds
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build [get]
func (m buildModule) getBuildListHandler(c *gin.Context) {
	var params = struct {
		commonGetQueryParams

		ScheduledAfter  *time.Time `form:"scheduledAfter"`
		ScheduledBefore *time.Time `form:"scheduledBefore"`
		FinishedAfter   *time.Time `form:"finishedAfter"`
		FinishedBefore  *time.Time `form:"finishedBefore"`

		ProjectID   *uint   `form:"projectId"`
		Environment *string `form:"environment"`
		GitBranch   *string `form:"gitBranch"`
		Stage       *string `form:"stage"`

		IsInvalid *bool `form:"isInvalid"`

		Status   *string `form:"status"`
		StatusID *int    `form:"statusId" binding:"excluded_with=Status"`

		EnvironmentMatch *string `form:"environmentMatch" binding:"excluded_with=Environment"`
		GitBranchMatch   *string `form:"gitBranchMatch" binding:"excluded_with=GitBranch"`
		StageMatch       *string `form:"stageMatch" binding:"excluded_with=Stage"`

		Match *string `form:"match"`
	}{
		commonGetQueryParams: defaultCommonGetQueryParams,
	}
	if !bindCommonGetQueryParams(c, &params) {
		return
	}
	orderBySlice, ok := parseCommonOrderBySlice(c, params.OrderBy, buildJSONToColumns)
	if !ok {
		return
	}

	var where wherefields.Collection

	var statusID database.BuildStatus
	if params.StatusID != nil {
		statusID = database.BuildStatus(*params.StatusID)
		if !statusID.IsValid() {
			err := fmt.Errorf("invalid database build status: %v", statusID)
			ginutil.WriteInvalidParamError(c, err, "statusId", fmt.Sprintf("Invalid build status ID: %d", *params.StatusID))
			return
		}
		where.AddFieldName(database.BuildColumns.StatusID)
	} else if params.Status != nil {
		reqStatusID := request.BuildStatus(*params.Status)
		statusID, ok = modelconv.ReqBuildStatusToDatabase(reqStatusID)
		if !ok {
			err := fmt.Errorf("invalid request build status: %v", reqStatusID)
			ginutil.WriteInvalidParamError(c, err, "status", fmt.Sprintf("Invalid build status: %q", *params.Status))
			return
		}
		where.AddFieldName(database.BuildColumns.StatusID)
	}

	query := m.Database.
		Clauses(orderBySlice.ClauseIfNone(defaultGetBuildsOrderBy)).
		Where(&database.Build{
			ProjectID:   where.Uint(database.BuildFields.ProjectID, params.ProjectID),
			Environment: where.NullStringEmptyNull(database.BuildFields.Environment, params.Environment),
			GitBranch:   where.String(database.BuildFields.GitBranch, params.GitBranch),
			IsInvalid:   where.Bool(database.BuildFields.IsInvalid, params.IsInvalid),
			Stage:       where.String(database.BuildFields.Stage, params.Stage),
			StatusID:    statusID,
		}, where.NonNilFieldNames()...).
		Scopes(
			optionalTimeRangeScope(database.BuildColumns.ScheduledOn, params.ScheduledAfter, params.ScheduledBefore),
			optionalTimeRangeScope(database.BuildColumns.CompletedOn, params.FinishedAfter, params.FinishedBefore),
			whereLikeScope(map[string]*string{
				database.BuildColumns.Environment: params.EnvironmentMatch,
				database.BuildColumns.GitBranch:   params.GitBranchMatch,
				database.BuildColumns.Stage:       params.StageMatch,
			}),
			whereAnyLikeScope(
				params.Match,
				database.BuildColumns.Environment,
				database.BuildColumns.GitBranch,
				database.BuildColumns.Stage,
			),
		)

	var dbBuilds []database.Build
	var totalCount int64
	err := findDBPaginatedSliceAndTotalCount(query, params.Limit, params.Offset, &dbBuilds, &totalCount)
	if err != nil {
		ginutil.WriteDBReadError(c, err, "Failed fetching list of builds from database.")
		return
	}

	c.JSON(http.StatusOK, response.PaginatedBuilds{
		Builds:     modelconv.DBBuildsToResponses(dbBuilds),
		TotalCount: totalCount,
	})
}

// getBuildLogListHandler godoc
// @id getBuildLogList
// @summary Finds logs for build with selected build ID
// @tags build
// @param buildId path uint true "build id" minimum(0)
// @success 200 {object} []response.Log "logs from selected build"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildId}/log [get]
func (m buildModule) getBuildLogListHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildId")
	if !ok {
		return
	}

	dbLogs, err := m.getLogs(buildID)
	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching logs for build with ID %d.",
			buildID))
		return
	}

	resLogs := make([]response.Log, len(dbLogs))
	for i, dbLog := range dbLogs {
		resLogs[i] = response.Log{
			LogID:     dbLog.LogID,
			BuildID:   dbLog.BuildID,
			Message:   dbLog.Message,
			Timestamp: dbLog.Timestamp,
		}
	}

	c.JSON(http.StatusOK, resLogs)
}

// streamBuildLogHandler godoc
// @id streamBuildLog
// @summary Opens stream listener
// @tags build
// @param buildId path uint true "build id" minimum(0)
// @success 200 "Open stream"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @router /build/{buildId}/stream [get]
func (m buildModule) streamBuildLogHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildId")
	if !ok {
		return
	}

	listener := openListener(buildID)
	defer closeListener(buildID, listener)

	clientGone := c.Writer.CloseNotify()
	c.Stream(func(w io.Writer) bool {
		select {
		case <-clientGone:
			return false
		case message := <-listener:
			c.SSEvent("message", message)
			return true
		}
	})
}

// createBuildLogHandler godoc
// @id createBuildLog
// @summary Post a log to selected build
// @tags build
// @param buildId path uint true "build id" minimum(0)
// @param data body request.LogOrStatusUpdate true "data"
// @success 201 "Created"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildId}/log [post]
func (m buildModule) createBuildLogHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildId")
	if !ok {
		return
	}

	var reqLogOrStatusUpdate request.LogOrStatusUpdate
	if err := c.ShouldBindJSON(&reqLogOrStatusUpdate); err != nil {
		ginutil.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for log object to post.")
		return
	}

	if dbBuildStatus, ok := modelconv.ReqBuildStatusToDatabase(reqLogOrStatusUpdate.Status); ok {
		_, err := m.updateBuildStatus(buildID, dbBuildStatus)
		if err != nil {
			ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
				"Failed updating status on build with ID %d to status with ID %d.",
				buildID, dbBuildStatus))
			return
		}
	} else {
		dbLog, err := m.saveLog(buildID, reqLogOrStatusUpdate.Message, reqLogOrStatusUpdate.Timestamp)
		if err != nil {
			ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
				"Failed adding log message to build with ID %d.",
				buildID))
			return
		}
		resLog := response.Log{
			LogID:     dbLog.LogID,
			BuildID:   dbLog.BuildID,
			Message:   dbLog.Message,
			Timestamp: dbLog.Timestamp,
		}
		build(buildID).Submit(resLog)
	}

	c.Status(http.StatusCreated)
}

// updateBuildHandler godoc
// @id updateBuild
// @summary Partially update specific build
// @tags build
// @param buildId path uint true "build id" minimum(0)
// @param status query string true "Build status term" Enums(Scheduling, Running, Completed, Failed)
// @success 200 {object} response.Build
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Build not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildId} [put]
func (m buildModule) updateBuildHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildId")
	if !ok {
		return
	}

	status, ok := ginutil.RequireQueryString(c, "status")
	if !ok {
		return
	}

	statusID, ok := modelconv.ReqBuildStatusToDatabase(request.BuildStatus(status))
	if !ok {
		ginutil.WriteInvalidParamError(c, nil, "status", fmt.Sprintf(
			"Unable to parse build status from %q", status))
		return
	}

	dbBuild, err := m.updateBuildStatus(buildID, statusID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ginutil.WriteDBNotFound(c, fmt.Sprintf(
			"Build with ID %d was not found when trying to update status to %q.",
			buildID, status))
		return
	} else if err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed updating build status to %q on build with ID %d in the database.",
			status, buildID))
		return
	}

	resBuild := modelconv.DBBuildToResponse(dbBuild)
	c.JSON(http.StatusOK, resBuild)
}

func (m buildModule) updateBuildStatus(buildID uint, statusID database.BuildStatus) (database.Build, error) {
	if !statusID.IsValid() {
		return database.Build{}, fmt.Errorf("invalid status ID: %+v", statusID)
	}

	dbBuild, err := m.getBuild(buildID)
	if err != nil {
		return database.Build{}, err
	}

	message := struct {
		StatusBefore database.BuildStatus
		StatusAfter  database.BuildStatus
		Build        database.Build
	}{
		StatusBefore: dbBuild.StatusID,
		StatusAfter:  statusID,
	}

	dbBuild.StatusID = statusID
	setStatusDate(&dbBuild, statusID)

	message.Build = dbBuild

	if err := m.Database.Save(&dbBuild).Error; err != nil {
		return database.Build{}, err
	}

	return dbBuild, nil
}

func (m buildModule) saveLog(buildID uint, message string, timestamp time.Time) (database.Log, error) {
	dbLog := database.Log{
		BuildID:   buildID,
		Message:   message,
		Timestamp: timestamp,
	}
	if err := m.Database.Save(&dbLog).Error; err != nil {
		return database.Log{}, err
	}
	return dbLog, nil
}

func setStatusDate(build *database.Build, statusID database.BuildStatus) {
	now := time.Now().UTC()
	switch statusID {
	case database.BuildRunning:
		build.StartedOn.SetValid(now)
	case database.BuildCompleted, database.BuildFailed:
		build.CompletedOn.SetValid(now)
	}
}

func (m buildModule) getBuild(buildID uint) (database.Build, error) {
	var dbBuild database.Build
	if err := m.Database.
		Where(&database.Build{BuildID: buildID}).
		Preload(database.BuildFields.TestResultSummaries).
		Preload(database.BuildFields.Params).
		First(&dbBuild).
		Error; err != nil {
		return database.Build{}, err
	}
	return dbBuild, nil
}

func (m buildModule) getLogs(buildID uint) ([]database.Log, error) {
	var dbLogs []database.Log
	if err := m.Database.
		Where(&database.Build{BuildID: buildID}).
		Find(&dbLogs).
		Error; err != nil {
		return []database.Log{}, err
	}
	return dbLogs, nil
}

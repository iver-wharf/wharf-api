package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"net/http"
	"net/url"

	"github.com/dustin/go-broadcast"
	"github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/v5/internal/coalesce"
	"github.com/iver-wharf/wharf-api/v5/internal/wherefields"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/request"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/response"
	"github.com/iver-wharf/wharf-api/v5/pkg/modelconv"
	"github.com/iver-wharf/wharf-api/v5/pkg/orderby"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"github.com/iver-wharf/wharf-core/pkg/problem"
	"gopkg.in/guregu/null.v4"
	"gorm.io/gorm"
)

type buildModule struct {
	Database *gorm.DB
	Config   *Config
}

func (m buildModule) Register(g *gin.RouterGroup) {
	build := g.Group("/build")
	{
		build.GET("", m.getBuildListHandler)

		buildByID := build.Group("/:buildId")
		{
			buildByID.GET("", m.getBuildHandler)
			buildByID.PUT("/status", m.updateBuildStatusHandler)
			buildByID.POST("/log", m.createBuildLogHandler)
			buildByID.GET("/log", m.getBuildLogListHandler)
			buildByID.GET("/stream", m.streamBuildLogHandler)

			artifacts := artifactModule{m.Database}
			artifacts.Register(buildByID)

			buildTestResults := buildTestResultModule{m.Database}
			buildTestResults.Register(buildByID)
		}
	}
	projectByID := g.Group("/project/:projectId")
	{
		projectByID.POST("/build", m.startProjectBuildHandler)
		// Deprecated:
		projectByID.POST("/:stage/run", m.oldStartProjectBuildHandler)
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
// @description Added in v0.3.5.
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

	resBuild := modelconv.DBBuildToResponse(dbBuild, m.engineLookup)
	c.JSON(http.StatusOK, resBuild)
}

var buildJSONToColumns = map[string]database.SafeSQLName{
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
// @description Added in v5.0.0.
// @tags build
// @param limit query int false "Number of results to return. No limiting is applied if empty (`?limit=`) or non-positive (`?limit=0`). Required if `offset` is used." default(100)
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
		where.AddFieldName(database.BuildFields.StatusID)
	} else if params.Status != nil {
		reqStatusID := request.BuildStatus(*params.Status)
		statusID, ok = modelconv.ReqBuildStatusToDatabase(reqStatusID)
		if !ok {
			err := fmt.Errorf("invalid request build status: %v", reqStatusID)
			ginutil.WriteInvalidParamError(c, err, "status", fmt.Sprintf("Invalid build status: %q", *params.Status))
			return
		}
		where.AddFieldName(database.BuildFields.StatusID)
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
			whereLikeScope(map[database.SafeSQLName]*string{
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
		List:       modelconv.DBBuildsToResponses(dbBuilds, m.engineLookup),
		TotalCount: totalCount,
	})
}

// getBuildLogListHandler godoc
// @id getBuildLogList
// @summary Finds logs for build with selected build ID
// @description Added in v0.3.8.
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
// @description Added in v0.3.8.
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
// @description Added in v0.1.0.
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

// updateBuildStatusHandler godoc
// @id updateBuildStatus
// @summary Update a build's status.
// @description Added in v5.0.0.
// @tags build
// @param buildId path uint true "Build ID" minimum(0)
// @param data body request.BuildStatusUpdate true "Status update"
// @success 204 "Updated"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Build not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildId}/status [put]
func (m buildModule) updateBuildStatusHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildId")
	if !ok {
		return
	}
	var reqStatusUpdate request.BuildStatusUpdate
	if err := c.ShouldBindJSON(&reqStatusUpdate); err != nil {
		ginutil.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for build status update.")
		return
	}
	if !validateBuildExistsByID(c, m.Database, buildID, "when updating build status") {
		return
	}
	dbBuildStatus, ok := modelconv.ReqBuildStatusToDatabase(reqStatusUpdate.Status)
	if !ok {
		err := errors.New("invalid build status value")
		ginutil.WriteInvalidParamError(c, err, "status", fmt.Sprintf(
			"The new build status %q is not a valid build status value.",
			reqStatusUpdate.Status,
		))
	}
	_, err := m.updateBuildStatus(buildID, dbBuildStatus)
	if err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed updating status on build with ID %d to status with ID %d.",
			buildID, dbBuildStatus))
		return
	}
	c.Status(http.StatusNoContent)
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

// oldStartProjectBuildHandler godoc
// @id oldStartProjectBuild
// @deprecated
// @summary Responsible for run stage environment for selected project
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use `POST /project/{projectId}/build` instead.
// @description Added in v0.2.4.
// @tags project
// @accept json
// @param projectId path uint true "project ID" minimum(0)
// @param stage path string true "name of stage to run, or specify ALL to run everything"
// @param branch query string false "branch name, uses default branch if omitted"
// @param environment query string false "environment name"
// @param inputs body string _ "user inputs" example(foo:bar)
// @success 200 {object} response.BuildReferenceWrapper "Build scheduled"
// @failure 400 {object} problem.Response "Bad request, such as invalid body JSON"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Project was not found"
// @failure 502 {object} problem.Response "Database or code execution engine is unreachable"
// @router /project/{projectId}/{stage}/run [post]
func (m buildModule) oldStartProjectBuildHandler(c *gin.Context) {
	// not moved to `internal/deprecated` package as it's too much
	// code duplication
	projectID, ok := ginutil.ParseParamUint(c, "projectId")
	if !ok {
		return
	}
	stageName := c.Param("stage")
	engineID := ""
	m.startBuildHandler(c, projectID, stageName, engineID)
}

// startProjectBuildHandler godoc
// @id startProjectBuild
// @summary Start a new build for the given project, with optional build stage, build environment, or repo branch filters.
// @description Added in v5.0.0.
// @tags build
// @accept json
// @param projectId path uint true "Project ID" minimum(0)
// @param stage query string false "Name of stage to run, or specify `ALL` to run all stages." default(ALL)
// @param branch query string false "Branch name. Uses project's default branch if omitted"
// @param environment query string false "Environment name filter. If left empty it will run all stages without any environment filters."
// @param engine query string false "Execution engine ID"
// @param inputs body request.BuildInputs _ "Input variable values. Map of variable names (as defined in the project's `.wharf-ci.yml` file) as keys paired with their string, boolean, or numeric value."
// @success 200 {object} response.BuildReferenceWrapper "Build scheduled"
// @failure 400 {object} problem.Response "Bad request, such as invalid body JSON"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Project was not found"
// @failure 502 {object} problem.Response "Database or code execution engine is unreachable"
// @router /project/{projectId}/build [post]
func (m buildModule) startProjectBuildHandler(c *gin.Context) {
	projectID, ok := ginutil.ParseParamUint(c, "projectId")
	if !ok {
		return
	}
	stageName, hasStageName := c.GetQuery("stage")
	if !hasStageName {
		stageName = "ALL"
	}
	engineID := c.Query("engine")
	m.startBuildHandler(c, projectID, stageName, engineID)
}

func (m buildModule) startBuildHandler(c *gin.Context, projectID uint, stageName string, engineID string) {
	engine, ok := lookupEngineOrDefaultFromConfig(m.Config.CI, engineID)
	if !ok {
		if engineID == "" {
			ginutil.WriteProblem(c, problem.Response{
				Type:   "/prob/api/engine/no-default",
				Title:  "No default execution engine configured.",
				Status: http.StatusInternalServerError,
				Detail: "The wharf-api does not have any default execution engine configured, meaning it doesn't know where to run your Wharf build.",
			})
			return
		}
		err := fmt.Errorf("unknown engine by ID: %q", engineID)
		ginutil.WriteInvalidParamError(c, err, "engine", fmt.Sprintf(
			"No execution engine was found by ID %q. You can skip to specify the engine ID to use the default execution engine.",
			engineID))
		return
	}

	dbProject, ok := fetchProjectByID(c, m.Database, projectID, "when starting a new build")
	if !ok {
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		ginutil.WriteBodyReadError(c, err, fmt.Sprintf(
			"Failed to read the input variables body when starting a new build for project with ID %d.",
			projectID))
		return
	}

	env, hasEnv := c.GetQuery("environment")
	branch, hasBranch := c.GetQuery("branch")

	if !hasBranch {
		b, ok := findDefaultBranch(dbProject.Branches)
		if !ok {
			ginutil.WriteDBNotFound(c, fmt.Sprintf(
				"No branch to build for project with ID %d was specified, and no default branch was found on the project.",
				projectID))
			return
		}
		branch = b.Name
	}

	now := time.Now().UTC()
	dbBuild := database.Build{
		ProjectID:   dbProject.ProjectID,
		ScheduledOn: null.TimeFrom(now),
		GitBranch:   branch,
		Environment: null.NewString(env, hasEnv),
		Stage:       stageName,
		EngineID:    engine.ID,
	}
	if err := m.Database.Create(&dbBuild).Error; err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed creating build on stage %q and branch %q for project with ID %d in database.",
			stageName, branch, projectID))
		return
	}

	dbBuildParams, err := parseDBBuildParams(dbBuild.BuildID, []byte(dbProject.BuildDefinition), body)
	if err != nil {
		dbBuild.IsInvalid = true
		if saveErr := m.Database.Save(&dbBuild).Error; saveErr != nil {
			c.Error(saveErr)
		}
		ginutil.WriteProblemError(c, err, problem.Response{
			Type:   "/prob/api/project/run/params-deserialize",
			Title:  "Parsing build parameters failed.",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf(
				"Failed to deserialize build parameters from request body for build on stage %q and branch %q for project with ID %d.",
				stageName, branch, projectID),
		})
		return
	}

	err = m.SaveBuildParams(dbBuildParams)
	if err != nil {
		dbBuild.IsInvalid = true
		if saveErr := m.Database.Save(&dbBuild).Error; saveErr != nil {
			c.Error(saveErr)
		}
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed saving build parameters for build on stage %q and branch %q for project with ID %d in database.",
			stageName, branch, projectID))
		return
	}

	dbJobParams, err := getDBJobParams(dbProject, dbBuild, dbBuildParams, m.Config.InstanceID)
	if err != nil {
		dbBuild.IsInvalid = true
		if saveErr := m.Database.Save(&dbBuild).Error; saveErr != nil {
			c.Error(saveErr)
		}
		ginutil.WriteProblemError(c, err, problem.Response{
			Type:   "/prob/api/project/run/params-serialize",
			Title:  "Serializing build parameters failed.",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf(
				"Failed to serialize build parameters before sending them onwards to Wharfs execution engine for build on stage %q and branch %q for project with ID %d.",
				stageName, branch, projectID),
		})
		return
	}

	if m.Config.CI.MockTriggerResponse {
		log.Info().Message("Setting for mocking build triggers was true, mocking CI response.")
		c.JSON(http.StatusOK, modelconv.DBBuildToResponseBuildReferenceWrapper(dbBuild))
		return
	}

	_, err = triggerBuild(dbJobParams, engine)
	if err != nil {
		dbBuild.IsInvalid = true
		if saveErr := m.Database.Save(&dbBuild).Error; saveErr != nil {
			c.Error(saveErr)
		}

		ginutil.WriteProblemError(c, err, problem.Response{
			Type:   "/prob/api/project/run/trigger",
			Title:  "Triggering build failed.",
			Status: http.StatusBadGateway,
			Detail: fmt.Sprintf(
				"Failed to trigger code execution engine to schedule the build with ID %d on stage %q on branch %q for project with ID %d.",
				dbBuild.BuildID, stageName, branch, projectID),
		})
		return
	}

	c.JSON(http.StatusOK, modelconv.DBBuildToResponseBuildReferenceWrapper(dbBuild))
}

func (m buildModule) SaveBuildParams(dbParams []database.BuildParam) error {
	for _, dbParam := range dbParams {
		if err := m.Database.Create(&dbParam).Error; err != nil {
			return err
		}
	}
	return nil
}

func (m buildModule) engineLookup(id string) *response.Engine {
	return lookupResponseEngineFromConfig(m.Config.CI, id)
}

func parseDBBuildParams(buildID uint, buildDef []byte, vars []byte) ([]database.BuildParam, error) {
	type BuildDefinition struct {
		Inputs []struct {
			Name    string
			Type    string
			Default string
		}
	}

	var def BuildDefinition
	err := yaml.Unmarshal(buildDef, &def)
	if err != nil {
		log.Error().WithError(err).Message("Failed unmarshaling build-def.")
		return nil, err
	}

	log.Info().
		WithInt("inputs", len(def.Inputs)).
		Message("Unmarshaled build-def.")

	m := make(request.BuildInputs)
	err = json.Unmarshal(vars, &m)
	if err != nil {
		log.Error().WithError(err).Message("Failed unmarshaling input variables JSON.")
		return nil, err
	}

	var params []database.BuildParam
	for _, input := range def.Inputs {
		param := database.BuildParam{
			Name:    input.Name,
			BuildID: buildID,
		}

		if m[input.Name] == nil {
			param.Value = input.Default
		} else {
			param.Value = fmt.Sprintf("%v", m[input.Name])
		}

		params = append(params, param)
	}

	return params, nil
}

func triggerBuild(dbJobParams []database.Param, engine CIEngineConfig) (string, error) {
	q := ""
	for _, dbJobParam := range dbJobParams {
		if dbJobParam.Value != "" {
			q = fmt.Sprintf("%s&%s=%s", q, url.QueryEscape(dbJobParam.Name), url.QueryEscape(dbJobParam.Value))
		}
	}

	tokenStr := fmt.Sprintf("?token=%s", engine.Token)

	url := fmt.Sprintf("%s%s%s", engine.URL, tokenStr, q)
	fmt.Printf("POSTing to url: %v\n", url)
	log.Info().
		WithString("method", "POST").
		WithString("url", fmt.Sprintf("%s?token=%s%s", engine.URL, "*****", q)).
		Message("Triggering build.")

	var resp, err = http.Post(url, "", nil)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	var body, err2 = ioutil.ReadAll(resp.Body)
	if err2 != nil {
		return "", err2
	}

	var strBody = string(body)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return "", fmt.Errorf(strBody)
	}

	return strBody, err2
}

func getDBJobParams(
	dbProject database.Project,
	dbBuild database.Build,
	dbBuildParams []database.BuildParam,
	wharfInstanceID string,
) ([]database.Param, error) {
	var err error
	var v []byte
	if len(dbBuildParams) > 0 {
		m := make(map[string]interface{})

		for _, input := range dbBuildParams {
			m[input.Name] = input.Value
		}

		v, err = yaml.Marshal(m)
		if err != nil {
			log.Error().WithError(err).Message("Failed to marshal input variables YAML for build.")
			return nil, err
		}
	} else {
		log.Debug().Message("Skipping input variables, nothing in body.")
	}

	token := ""
	if dbProject.Token != nil {
		token = dbProject.Token.Token
	}

	dbJobParams := []database.Param{
		{Type: "string", Name: "REPO_NAME", Value: dbProject.Name},
		{Type: "string", Name: "REPO_GROUP", Value: strings.ToLower(dbProject.GroupName)},
		{Type: "string", Name: "REPO_BRANCH", Value: dbBuild.GitBranch},
		{Type: "string", Name: "GIT_BRANCH", Value: dbBuild.GitBranch},
		{Type: "string", Name: "RUN_STAGES", Value: dbBuild.Stage},
		{Type: "string", Name: "BUILD_REF", Value: strconv.FormatUint(uint64(dbBuild.BuildID), 10)},
		{Type: "string", Name: "VARS", Value: string(v)},
		{Type: "string", Name: "GIT_FULLURL", Value: coalesce.String(dbProject.Overrides.GitURL, dbProject.GitURL)},
		{Type: "string", Name: "GIT_TOKEN", Value: token},
		{Type: "string", Name: "WHARF_PROJECT_ID", Value: strconv.FormatUint(uint64(dbProject.ProjectID), 10)},
		{Type: "string", Name: "WHARF_INSTANCE", Value: wharfInstanceID},
	}

	if dbBuild.Environment.Valid {
		dbJobParams = append(dbJobParams, database.Param{
			Type:  "string",
			Name:  "ENVIRONMENT",
			Value: dbBuild.Environment.String,
		})
	}

	return dbJobParams, nil
}

func validateBuildExistsByID(c *gin.Context, db *gorm.DB, buildID uint, whenMsg string) bool {
	return validateDatabaseObjExistsByID(c, db, &database.Build{}, buildID, "build", whenMsg)
}

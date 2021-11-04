package main

import (
	"errors"
	"fmt"
	"io"
	"time"

	"net/http"

	"github.com/dustin/go-broadcast"
	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/request"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
	"github.com/iver-wharf/wharf-api/pkg/modelconv"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"gorm.io/gorm"
)

type buildModule struct {
	Database *gorm.DB
}

func (m buildModule) Register(g *gin.RouterGroup) {
	builds := g.Group("/builds")
	{
		builds.POST("/search", m.searchBuildListHandler)
	}

	build := g.Group("/build/:buildId")
	{
		build.GET("", m.getBuildHandler)
		build.PUT("", m.updateBuildHandler)
		build.POST("/log", m.createBuildLogHandler)
		build.GET("/log", m.getBuildLogListHandler)
		build.GET("/stream", m.streamBuildLogHandler)

		artifacts := artifactModule{m.Database}
		artifacts.Register(build)

		buildTestResults := buildTestResultModule{m.Database}
		buildTestResults.Register(build)
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

// searchBuildListHandler godoc
// @id searchBuildList
// @summary NOT IMPLEMENTED YET
// @tags build
// @accept json
// @produce json
// @success 501 "Not Implemented"
// @router /builds/search [post]
func (m buildModule) searchBuildListHandler(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
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

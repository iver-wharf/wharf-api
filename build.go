package main

import (
	"errors"
	"fmt"
	"io"
	"time"

	"encoding/json"
	"net/http"

	"github.com/dustin/go-broadcast"
	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"gorm.io/gorm"

	"github.com/iver-wharf/messagebus-go"
)

// BuildLog is a single log line, together with its timestamp of when it was
// logged.
type BuildLog struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
	// StatusID is populated when unmarshalled via UnmarshalJSON
	StatusID BuildStatus `json:"-"`
}

// UnmarshalJSON implements Unmarshaler interface from encoding/json.
func (bl *BuildLog) UnmarshalJSON(data []byte) error {
	type antiInfiniteLoop BuildLog
	if err := json.Unmarshal(data, (*antiInfiniteLoop)(bl)); err != nil {
		return err
	}
	if bl.Status == "" {
		bl.StatusID = -1
	} else {
		if statusID, ok := parseBuildStatus(bl.Status); ok {
			bl.StatusID = statusID
		} else {
			return fmt.Errorf("invalid build status: %s", bl.Status)
		}
	}
	return nil
}

type buildModule struct {
	Database     *gorm.DB
	MessageQueue *messagebus.MQConnection
}

func (m buildModule) Register(g *gin.RouterGroup) {
	builds := g.Group("/builds")
	{
		builds.POST("/search", m.postBuildSearchHandler)
	}

	build := g.Group("/build/:buildid")
	{
		build.GET("", m.getBuildHandler)
		build.PUT("", m.putBuildStatus)
		build.POST("/log", m.postBuildLogHandler)
		build.GET("/log", m.getLogHandler)
		build.GET("/stream", m.streamBuildLogHandler)

		artifacts := artifactModule{m.Database}
		artifacts.Register(build)
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
// @summary Finds build by build ID
// @tags build
// @param buildid path int true "build id"
// @success 200 {object} Build
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Build not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid} [get]
func (m buildModule) getBuildHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}

	build, err := m.getBuild(buildID)
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

	c.JSON(http.StatusOK, &build)
}

func (m buildModule) getBuild(buildID uint) (Build, error) {
	var build Build
	if err := m.Database.
		Where(&Build{BuildID: buildID}).
		Preload(buildAssocParams).
		First(&build).
		Error; err != nil {
		return Build{}, err
	}
	return build, nil
}

func (m buildModule) getLogs(buildID uint) ([]Log, error) {
	var logs []Log
	if err := m.Database.
		Where(&Build{BuildID: buildID}).
		Find(&logs).
		Error; err != nil {
		return []Log{}, err
	}
	return logs, nil
}

// postBuildSearchHandler godoc
// @summary NOT IMPLEMENTED YET
// @tags build
// @accept json
// @produce json
// @success 501 "Not Implemented"
// @router /builds/search [post]
func (m buildModule) postBuildSearchHandler(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

// getLogHandler godoc
// @summary Finds logs for build with selected build ID
// @tags build
// @param buildid path int true "build id"
// @success 200 {object} []Log "logs from selected build"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/log [get]
func (m buildModule) getLogHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}

	logs, err := m.getLogs(buildID)

	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching logs for build with ID %d.",
			buildID))
		return
	}

	c.JSON(http.StatusOK, logs)
}

// streamBuildLogHandler godoc
// @summary Opens stream listener
// @tags build
// @param buildid path int true "build id"
// @success 200 "Open stream"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @router /build/{buildid}/stream [get]
func (m buildModule) streamBuildLogHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
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

// postBuildLogHandler godoc
// @summary Post a log to selected build
// @tags build
// @param buildid path int true "build id"
// @param data body BuildLog true "data"
// @success 201 "Created"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/log [post]
func (m buildModule) postBuildLogHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}

	var log BuildLog
	if err := c.ShouldBindJSON(&log); err != nil {
		ginutil.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for log object to post.")
		return
	}

	if log.StatusID >= 0 {
		_, err := m.updateBuildStatus(buildID, log.StatusID)
		if err != nil {
			ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
				"Failed updating status on build with ID %d to status with ID %d.",
				buildID, log.StatusID))
			return
		}
	} else {
		if err := m.saveLog(buildID, log.Message, log.Timestamp); err != nil {
			ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
				"Failed adding log message to build with ID %d.",
				buildID))
			return
		}
	}

	build(buildID).Submit(log)
	c.Status(http.StatusCreated)
}

// putBuildStatus godoc
// @summary Partially update specific build
// @tags build
// @param buildid path uint true "build id"
// @param status query string true "Build status term" Enums(Scheduling, Running, Completed, Failed)
// @success 200 {object} Build
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Build not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid} [put]
func (m buildModule) putBuildStatus(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}

	status, ok := ginutil.RequireQueryString(c, "status")
	if !ok {
		return
	}

	statusID, ok := parseBuildStatus(status)
	if !ok {
		ginutil.WriteInvalidParamError(c, nil, "status", fmt.Sprintf(
			"Unable to parse build status from %q", status))
		return
	}

	build, err := m.updateBuildStatus(buildID, statusID)
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

	c.JSON(http.StatusOK, build)
}

func (m buildModule) updateBuildStatus(buildID uint, statusID BuildStatus) (Build, error) {
	if statusID < BuildScheduling && statusID > BuildFailed {
		return Build{}, fmt.Errorf("invalid status ID: %+v", statusID)
	}

	build, err := m.getBuild(buildID)
	if err != nil {
		return Build{}, err
	}

	message := struct {
		StatusBefore BuildStatus
		StatusAfter  BuildStatus
		Build        Build
	}{
		StatusBefore: build.StatusID,
		StatusAfter:  statusID,
	}

	build.StatusID = statusID
	setStatusDate(&build, statusID)

	message.Build = build

	if err := m.Database.Save(&build).Error; err != nil {
		return Build{}, err
	}

	if m.MessageQueue != nil {
		if err := m.MessageQueue.PublishMessage(message); err != nil {
			log.Error().WithError(err).Message("Failed sending build-status update message.")
		}
	}

	return build, nil
}

func (m buildModule) saveLog(buildID uint, message string, timestamp time.Time) error {
	return m.Database.Save(&Log{
		BuildID:   buildID,
		Message:   message,
		Timestamp: timestamp,
	}).Error
}

func setStatusDate(build *Build, statusID BuildStatus) {
	now := time.Now().UTC()
	switch statusID {
	case BuildRunning:
		build.StartedOn = &now
	case BuildCompleted, BuildFailed:
		build.CompletedOn = &now
	}
}

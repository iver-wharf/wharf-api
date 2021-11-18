package deprecated

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/request"
	"github.com/iver-wharf/wharf-api/pkg/modelconv"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"gorm.io/gorm"
)

// BuildModule holds deprecated endpoint handlers for /build
type BuildModule struct {
	Database *gorm.DB
}

// Register adds all deprecated endpoints to a given Gin router group.
func (m BuildModule) Register(g *gin.RouterGroup) {
	builds := g.Group("/builds")
	{
		builds.POST("/search", m.searchBuildListHandler)
	}

	build := g.Group("/build")
	{
		buildByID := build.Group("/:buildId")
		{
			buildByID.PUT("", m.updateBuildHandler)
			artifacts := artifactModule{m.Database}
			artifacts.Register(buildByID)
		}
	}
}

// searchBuildListHandler godoc
// @id oldSearchBuildList
// @deprecated
// @description This endpoint was never implemented!
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use `GET /build` instead.
// @summary NOT IMPLEMENTED YET
// @tags build
// @accept json
// @produce json
// @success 501 "Not Implemented"
// @router /builds/search [post]
func (m BuildModule) searchBuildListHandler(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

// updateBuildHandler godoc
// @id oldUpdateBuild
// @deprecated
// @summary Partially update specific build
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use `PUT /build/{buildId}/status` instead.
// @tags build
// @param buildId path uint true "build id" minimum(0)
// @param status query string true "Build status term" Enums(Scheduling, Running, Completed, Failed)
// @success 200 {object} response.Build
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Build not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildId} [put]
func (m BuildModule) updateBuildHandler(c *gin.Context) {
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

func (m BuildModule) updateBuildStatus(buildID uint, statusID database.BuildStatus) (database.Build, error) {
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

func (m BuildModule) getBuild(buildID uint) (database.Build, error) {
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

func setStatusDate(build *database.Build, statusID database.BuildStatus) {
	now := time.Now().UTC()
	switch statusID {
	case database.BuildRunning:
		build.StartedOn.SetValid(now)
	case database.BuildCompleted, database.BuildFailed:
		build.CompletedOn.SetValid(now)
	}
}

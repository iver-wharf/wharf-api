package deprecated

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
	"github.com/iver-wharf/wharf-api/pkg/modelconv"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"gorm.io/gorm"
)

// ArtifactModule holds deprecated endpoint handlers for /build/{buildId}/artifact
type artifactModule struct {
	Database *gorm.DB
}

// Register adds all deprecated endpoints to a given Gin router group.
func (m artifactModule) Register(g *gin.RouterGroup) {
	g.GET("/artifacts", m.getBuildArtifactListHandler)
}

// getBuildArtifactListHandler godoc
// @id oldGetBuildArtifactList
// @deprecated
// @summary Get list of build artifacts
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use `GET /build/{buildId}/artifact` instead.
// @description Added in TODO.
// @tags artifact
// @param buildId path uint true "Build ID" minimum(0)
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

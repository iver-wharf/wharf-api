package deprecated

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
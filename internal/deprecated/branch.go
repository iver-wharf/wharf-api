package deprecated

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BranchModule holds deprecated endpoint handlers for /branch
type BranchModule struct{}

// Register adds all deprecated endpoints to a given Gin router group.
func (m BranchModule) Register(g *gin.RouterGroup) {
	branch := g.Group("/branch")
	{
		branch.GET("/:branchid", m.GetBranchHandler)
	}
}

// GetBranchHandler godoc
// @summary Get a branch by ID
// @description This endpoint has not been implemented!
// @description Deprecated since v4.2.1. Planned for removal in v6.0.0.
// @deprecated
// @tags branch
// @param branchid path int true "branch ID"
// @success 501 "Not Implemented"
// @router /branch/{branchid} [get]
func (m BranchModule) GetBranchHandler(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

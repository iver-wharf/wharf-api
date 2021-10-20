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
		branch.GET("/:branchId", m.GetBranchHandler)
	}
}

// GetBranchHandler godoc
// @deprecated
// @id oldGetBranch
// @summary Get a branch by ID
// @description This endpoint has not been implemented!
// @description Deprecated since v4.3.0. Planned for removal in v6.0.0.
// @tags branch
// @param branchId path int true "branch ID"
// @success 501 "Not Implemented"
// @router /branch/{branchId} [get]
func (m BranchModule) GetBranchHandler(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

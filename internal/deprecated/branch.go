package deprecated

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/request"
	"github.com/iver-wharf/wharf-api/pkg/modelconv"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"gorm.io/gorm"
)

// BranchModule holds deprecated endpoint handlers for /branch
type BranchModule struct {
	Database *gorm.DB
}

// Register adds all deprecated endpoints to a given Gin router group.
func (m BranchModule) Register(g *gin.RouterGroup) {
	branch := g.Group("/branch")
	{
		branch.GET("/:branchId", m.GetBranchHandler)
	}
}

// GetBranchHandler godoc
// @summary Get a branch by ID
// @description This endpoint has not been implemented!
// @description Deprecated since v4.3.0. Planned for removal in v6.0.0.
// @deprecated
// @tags branch
// @param branchId path int true "branch ID"
// @success 501 "Not Implemented"
// @router /branch/{branchId} [get]
func (m BranchModule) GetBranchHandler(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

// updateProjectBranchListHandler godoc
// @deprecated
// @id oldUpdateProjectBranchList
// @summary Resets branches for a project
// @description Alters the database by removing, adding and updating until the stored branches equals the input branches.
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use PUT /project/{projectId}/branch instead.
// @tags branches
// @accept json
// @produce json
// @param branches body []request.Branch true "branch array"
// @success 200 {object} []response.Branch "Updated branches"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /branches [put]
func (m BranchModule) updateProjectBranchListHandler(c *gin.Context) {
	var reqBranches []request.Branch
	if err := c.ShouldBindJSON(&reqBranches); err != nil {
		ginutil.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for branch object array to update.")
		return
	}
	dbBranches, err := m.replaceBranchList(reqBranches)
	if err != nil {
		ginutil.WriteDBWriteError(c, err, "Failed to update branches in database.")
		return
	}
	resBranches := modelconv.DBBranchesToResponses(dbBranches)
	c.JSON(http.StatusOK, resBranches)
}

func (m BranchModule) replaceBranchList(reqBranches []request.Branch) ([]database.Branch, error) {
	var dbNewBranches []database.Branch

	err := m.Database.Transaction(func(tx *gorm.DB) error {
		var projectID uint
		var defaultBranchName string

		var dbOldBranches []database.Branch
		if len(reqBranches) > 0 {
			var reqFirstBranch = reqBranches[0]
			if err := tx.
				Where(&database.Branch{ProjectID: reqFirstBranch.ProjectID}, database.BranchFields.ProjectID).
				Find(&dbOldBranches).Error; err != nil {
				return err
			}
			projectID = reqFirstBranch.ProjectID
			defaultBranchName = reqFirstBranch.Name
		}

		branchNamesSet := map[string]struct{}{}

		for _, reqBranch := range reqBranches {
			if reqBranch.Default {
				defaultBranchName = reqBranch.Name
			}

			branchNamesSet[reqBranch.Name] = struct{}{}
			var count int64
			err := m.Database.
				Model(&database.Branch{}).
				Where(&database.Branch{
					ProjectID: reqBranch.ProjectID,
					TokenID:   reqBranch.TokenID,
					Name:      reqBranch.Name,
				}, database.BranchFields.ProjectID, database.BranchFields.TokenID, database.BranchFields.Name).
				Count(&count).
				Error
			if err != nil {
				return err
			}
			if count == 0 {
				if err := tx.Create(&database.Branch{
					ProjectID: reqBranch.ProjectID,
					TokenID:   reqBranch.TokenID,
					Name:      reqBranch.Name,
				}).Error; err != nil {
					return err
				}
			}
		}

		//set single default branch in project
		if err := tx.
			Model(&database.Branch{}).
			Where(&database.Branch{ProjectID: projectID, Default: true}).
			Where(tx.Not(&database.Branch{Name: defaultBranchName}, database.BranchFields.Name)).
			Select(database.BranchFields.Default).
			Updates(&database.Branch{Default: false}).Error; err != nil {
			return err
		}

		for _, dbOldBranch := range dbOldBranches {
			if _, ok := branchNamesSet[dbOldBranch.Name]; !ok {
				//remove old branch
				if err := tx.
					Where(&dbOldBranch, database.BranchFields.ProjectID, database.BranchFields.TokenID, database.BranchFields.Name).
					Delete(&dbOldBranch).Error; err != nil {
					return err
				}
				log.Info().
					WithString("branch", dbOldBranch.Name).
					WithUint("project", dbOldBranch.ProjectID).
					Message("Deleted branch from project.")
			}
		}

		return m.Database.Find(&dbNewBranches).Error
	})

	if err != nil {
		log.Error().
			WithError(err).
			Message("Failed to replace branch list. Transaction rolled back.")
		return nil, err
	}

	return dbNewBranches, nil
}

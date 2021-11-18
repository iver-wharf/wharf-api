package deprecated

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
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
	branches := g.Group("/branches")
	{
		branches.GET("", m.getBranchListHandler)
		branches.PUT("", m.updateProjectBranchListHandler)
	}

	branch := g.Group("/branch")
	{
		branch.GET("/:branchId", m.GetBranchHandler)
		branch.POST("", m.createBranchHandler)
	}
}

// GetBranchHandler godoc
// @deprecated
// @id oldGetBranch
// @summary Get a branch by ID
// @description This endpoint has not been implemented!
// @description Deprecated since v4.3.0. Planned for removal in v6.0.0.
// @description Added in v0.2.0.
// @tags branch
// @param branchId path uint true "branch ID" minimum(0)
// @success 501 "Not Implemented"
// @router /branch/{branchId} [get]
func (m BranchModule) GetBranchHandler(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

// getBranchListHandler godoc
// @deprecated
// @id oldGetBranchList
// @description This endpoint was never implemented!
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use `GET /project/{projectId}/branch` instead.
// @description Added in v0.2.0.
// @summary NOT IMPLEMENTED
// @tags branch
// @success 501 "Not Implemented"
// @router /branches [get]
func (m BranchModule) getBranchListHandler(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

// Branch specifies fields when adding a new branch to a project.
type Branch struct {
	BranchID  uint   `json:"branchId"`
	ProjectID uint   `json:"projectId"`
	Name      string `json:"name"`
	Default   bool   `json:"default"`
	TokenID   uint   `json:"tokenId"`
}

// createBranchHandler godoc
// @deprecated
// @id oldCreateBranch
// @summary Create or update branch.
// @description It finds branch by project ID, token ID and name.
// @description First found branch will have updated default flag.
// @description If not existing new branch will be created.
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use `PUT /project/{projectId}/branch` instead.
// @description Added in v0.2.0.
// @tags branch
// @accept json
// @produce json
// @param branch body deprecated.Branch true "branch object"
// @success 200 {object} response.Branch "Updated branch"
// @success 201 {object} response.Branch "Added new branch"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /branch [post]
func (m BranchModule) createBranchHandler(c *gin.Context) {
	var reqBranch Branch
	if err := c.ShouldBindJSON(&reqBranch); err != nil {
		ginutil.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for branch object to update.")
		return
	}

	dbBranch := database.Branch{
		ProjectID: reqBranch.ProjectID,
		TokenID:   reqBranch.TokenID,
		Name:      reqBranch.Name,
		Default:   reqBranch.Default,
	}

	var dbExistingBranch database.Branch
	err := m.Database.
		Where(&dbBranch, database.BranchFields.ProjectID, database.BranchFields.TokenID, database.BranchFields.Name).
		First(&dbExistingBranch).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := m.Database.Create(&dbBranch).Error; err != nil {
			ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
				"Failed creating branch with name %q for token with ID %d and for project with ID %d in database.",
				dbBranch.Name, dbBranch.TokenID, dbBranch.ProjectID))
			return
		}
		c.JSON(http.StatusCreated, modelconv.DBBranchToResponse(dbBranch))
		return
	} else if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching branch with name %q for token with ID %d and for project with ID %d in database.",
			reqBranch.Name, reqBranch.TokenID, reqBranch.ProjectID))
		return
	}

	dbExistingBranch.Default = reqBranch.Default
	m.Database.Save(dbExistingBranch)
	c.JSON(http.StatusOK, modelconv.DBBranchToResponse(dbExistingBranch))
}

// updateProjectBranchListHandler godoc
// @deprecated
// @id oldUpdateProjectBranchList
// @summary Resets branches for a project
// @description Alters the database by removing, adding and updating until the stored branches equals the input branches.
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use PUT /project/{projectId}/branch instead.
// @description Added in v1.0.0.
// @tags branches
// @accept json
// @produce json
// @param branches body []deprecated.Branch true "branch array"
// @success 200 {object} []response.Branch "Updated branches"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /branches [put]
func (m BranchModule) updateProjectBranchListHandler(c *gin.Context) {
	var reqBranches []Branch
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

func (m BranchModule) replaceBranchList(reqBranches []Branch) ([]database.Branch, error) {
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

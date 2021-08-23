package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/iver-wharf/wharf-core/pkg/ginutil"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type branchModule struct {
	Database *gorm.DB
}

func (m branchModule) Register(g *gin.RouterGroup) {
	branches := g.Group("/branches")
	{
		branches.GET("", m.GetBranchesHandler)
		branches.PUT("", m.PutBranchesHandler)
	}

	branch := g.Group("/branch")
	{
		branch.GET("/:branchid", m.GetBranchHandler)
		branch.POST("", m.PostBranchHandler)
	}
}

// GetBranchesHandler godoc
// @summary NOT IMPLEMENTED YET
// @tags branch
// @success 501 "Not Implemented"
// @router /branches [get]
func (m branchModule) GetBranchesHandler(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

// GetBranchHandler godoc
// @summary NOT IMPLEMENTED YET
// @tags branch
// @param branchid path int true "branch ID"
// @success 501 "Not Implemented"
// @router /branch/{branchid} [get]
func (m branchModule) GetBranchHandler(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

// PostBranchHandler godoc
// @summary Create or update branch.
// @description It finds branch by project ID, token ID and name.
// @description First found branch will have updated default flag.
// @description If not existing new branch will be created.
// @tags branch
// @accept json
// @produce json
// @param branch body Branch true "branch object"
// @success 200 {object} Branch "Updated branch"
// @success 201 {object} Branch "Added new branch"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /branch [post]
func (m branchModule) PostBranchHandler(c *gin.Context) {
	var branch Branch
	if err := c.ShouldBindJSON(&branch); err != nil {
		ginutil.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for branch object to update.")
		return
	}

	var existingBranch Branch
	err := m.Database.
		Where(&branch, branchFieldProjectID, branchFieldTokenID, branchFieldName).
		First(&existingBranch).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := m.Database.Create(&branch).Error; err != nil {
			ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
				"Failed creating branch with name %q for token with ID %d and for project with ID %d in database.",
				branch.Name, branch.TokenID, branch.ProjectID))
			return
		}
		c.JSON(http.StatusCreated, branch)
		return
	} else if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching branch with name %q for token with ID %d and for project with ID %d in database.",
			branch.Name, branch.TokenID, branch.ProjectID))
		return
	}

	existingBranch.Default = branch.Default
	m.Database.Save(existingBranch)
	c.JSON(http.StatusOK, existingBranch)
}

// PutBranchesHandler godoc
// @summary Resets branches for a project
// @description Alters the database by removing, adding and updating until the stored branches equals the input branches.
// @tags branches
// @accept json
// @produce json
// @param branches body []Branch true "branch array"
// @success 200 {object} []Branch "Updated branches"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /branches [put]
func (m branchModule) PutBranchesHandler(c *gin.Context) {
	var branches []Branch
	if err := c.ShouldBindJSON(&branches); err != nil {
		ginutil.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for branch object array to update.")
		return
	}
	if err := m.PutBranches(branches); err != nil {
		ginutil.WriteDBWriteError(c, err, "Failed to update branches in database.")
		return
	}
	c.JSON(http.StatusOK, branches)
}

func (m branchModule) PutBranches(branches []Branch) error {
	return m.Database.Transaction(func(tx *gorm.DB) error {
		var defaultBranch Branch
		var oldDbBranches []Branch
		if len(branches) > 0 {
			var firstBranch = branches[0]
			if err := tx.Where(&firstBranch, branchFieldProjectID).Find(&oldDbBranches).Error; err != nil {
				return err
			}
			defaultBranch = firstBranch
		}

		var branchNames []string
		for _, branch := range branches {
			if branch.Default {
				defaultBranch = branch
			}

			branchNames = append(branchNames, branch.Name)
			result := m.Database.
				Where(&branch, branchFieldProjectID, branchFieldTokenID, branchFieldName).
				First(&branch)
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				if err := tx.Create(&branch).Error; err != nil {
					return err
				}
			}
		}

		//set single default branch in project
		if err := tx.
			Model(&Branch{}).
			Where(&Branch{ProjectID: defaultBranch.ProjectID, Default: true}).
			Where(tx.Not(&defaultBranch, branchFieldName)).
			Select(branchFieldDefault).
			Updates(&Branch{Default: false}).Error; err != nil {
			return err
		}

		for _, oldDbBranch := range oldDbBranches {
			if !contains(branchNames, oldDbBranch.Name) {
				//remove old branch
				if err := tx.
					Where(&oldDbBranch, branchFieldProjectID, branchFieldTokenID, branchFieldName).
					Delete(&oldDbBranch).Error; err != nil {
					return err
				}
				log.Info().
					WithString("branch", oldDbBranch.Name).
					WithUint("project", oldDbBranch.ProjectID).
					Message("Deleted branch from project.")
			}
		}

		return nil
	})
}

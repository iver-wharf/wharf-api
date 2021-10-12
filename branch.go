package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/iver-wharf/wharf-api/internal/set"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/request"
	"github.com/iver-wharf/wharf-api/pkg/modelconv"
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
		branches.GET("", m.getBranchListHandler)
	}

	branch := g.Group("/branch")
	{
		branch.POST("", m.createBranchHandler)
	}

	projectBranch := g.Group("/project/:projectId/branch")
	{
		projectBranch.PUT("", m.updateProjectBranchListHandler)
	}
}

// getBranchListHandler godoc
// @id getBranchList
// @summary NOT IMPLEMENTED YET
// @tags branch
// @success 501 "Not Implemented"
// @router /branches [get]
func (m branchModule) getBranchListHandler(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

// createBranchHandler godoc
// @id createBranch
// @summary Create or update branch.
// @description It finds branch by project ID, token ID and name.
// @description First found branch will have updated default flag.
// @description If not existing new branch will be created.
// @tags branch
// @accept json
// @produce json
// @param branch body request.Branch true "branch object"
// @success 200 {object} response.Branch "Updated branch"
// @success 201 {object} response.Branch "Added new branch"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /branch [post]
func (m branchModule) createBranchHandler(c *gin.Context) {
	var reqBranch request.Branch
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
// @id updateProjectBranchList
// @summary Resets branches for a project
// @description Alters the database by removing, adding and updating until the stored branches equals the input branches.
// @tags branch
// @accept json
// @produce json
// @param projectId path int true "project ID"
// @param branches body request.BranchListUpdate true "Branch update"
// @success 200 {object} response.BranchList "Updated branches"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Project not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /project/{projectId}/branch [put]
func (m branchModule) updateProjectBranchListHandler(c *gin.Context) {
	projectID, ok := ginutil.ParseParamUint(c, "projectId")
	if !ok {
		return
	}
	var reqBranchListUpdate request.BranchListUpdate
	if err := c.ShouldBindJSON(&reqBranchListUpdate); err != nil {
		ginutil.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for branch object array to update.")
		return
	}
	dbProject, ok := fetchProjectByIDSlim(c, m.Database, projectID, "when updating branches")
	if !ok {
		return
	}
	dbBranchList, err := replaceBranchList(m.Database, projectID, dbProject.TokenID, reqBranchListUpdate)
	if err != nil {
		ginutil.WriteDBWriteError(c, err, "Failed to update branches in database.")
		return
	}
	resBranchList := modelconv.DBBranchListToResponse(dbBranchList.branches, dbBranchList.defaultBranch)
	c.JSON(http.StatusOK, resBranchList)
}

type databaseBranchList struct {
	defaultBranch *database.Branch
	branches      []database.Branch
}

func replaceBranchList(db *gorm.DB, projectID uint, tokenID uint, reqUpdate request.BranchListUpdate) (databaseBranchList, error) {
	err := db.Transaction(func(tx *gorm.DB) error {
		var dbOldBranches []database.Branch
		if err := tx.
			Where(&database.Branch{ProjectID: projectID}, database.BranchFields.ProjectID).
			Find(&dbOldBranches).Error; err != nil {
			return err
		}

		wantBranchNamesSet := set.String{}
		for _, reqBranchUpdate := range reqUpdate.Branches {
			wantBranchNamesSet.Set(reqBranchUpdate.Name)
		}

		hasBranchNamesSet := set.String{}
		for _, dbOldBranch := range dbOldBranches {
			hasBranchNamesSet.Set(dbOldBranch.Name)
		}

		branchNamesToDelete := hasBranchNamesSet.Difference(wantBranchNamesSet)
		branchNamesToAdd := wantBranchNamesSet.Difference(hasBranchNamesSet)

		var dbBranchesToAdd []database.Branch
		for branchName := range branchNamesToAdd {
			dbBranchesToAdd = append(dbBranchesToAdd, database.Branch{
				ProjectID: projectID,
				TokenID:   tokenID,
				Name:      branchName,
				Default:   branchName == reqUpdate.DefaultBranch,
			})
		}
		if len(dbBranchesToAdd) > 0 {
			if err := tx.Create(dbBranchesToAdd).Error; err != nil {
				return err
			}
			log.Info().
				WithInt("branchesAdded", len(dbBranchesToAdd)).
				WithUint("project", projectID).
				Message("Added branches to project when updating branches.")
		}

		var dbBranchIDsToDelete []uint
		for _, dbOldBranch := range dbOldBranches {
			if branchNamesToDelete.Has(dbOldBranch.Name) {
				dbBranchIDsToDelete = append(dbBranchIDsToDelete, dbOldBranch.BranchID)
			}
		}
		if len(dbBranchIDsToDelete) > 0 {
			if err := tx.
				Delete(&database.Branch{}, dbBranchIDsToDelete).Error; err != nil {
				return err
			}
			log.Info().
				WithInt("branchesDeleted", len(dbBranchIDsToDelete)).
				WithUint("project", projectID).
				Message("Deleted branches from project when updating branches.")
		}

		return setDefaultBranch(tx, projectID, reqUpdate.DefaultBranch)
	})

	if err != nil {
		log.Error().
			WithError(err).
			Message("Failed to replace branch list. Transaction rolled back.")
		return databaseBranchList{}, err
	}

	var dbNewBranches []database.Branch
	var dbNewDefaultBranch *database.Branch

	if err := db.
		Where(&database.Branch{ProjectID: projectID}, database.BranchFields.ProjectID).
		Find(&dbNewBranches).Error; err != nil {
		return databaseBranchList{}, err
	}

	for _, dbNewBranch := range dbNewBranches {
		if dbNewBranch.Default {
			dbNewDefaultBranch = &dbNewBranch
			break
		}
	}

	return databaseBranchList{dbNewDefaultBranch, dbNewBranches}, nil
}

func setDefaultBranch(db *gorm.DB, projectID uint, defaultBranchName string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// ensure "default=false" on all other branches
		if err := tx.
			Model(&database.Branch{}).
			Where(&database.Branch{ProjectID: projectID, Default: true},
				database.BranchFields.ProjectID,
				database.BranchFields.Default).
			Where(tx.Not(&database.Branch{Name: defaultBranchName},
				database.BranchFields.Name)).
			Select(database.BranchFields.Default).
			Updates(&database.Branch{Default: false}).Error; err != nil {
			return err
		}
		// ensure "default=true" on default branch
		return tx.
			Model(&database.Branch{}).
			Where(&database.Branch{ProjectID: projectID, Name: defaultBranchName},
				database.BranchFields.ProjectID,
				database.BranchFields.Name).
			Select(database.BranchFields.Default).
			Updates(&database.Branch{Default: true}).Error
	})
}

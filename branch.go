package main

import (
	"fmt"
	"net/http"

	"github.com/iver-wharf/wharf-api/internal/ptrconv"
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
	projectBranch := g.Group("/project/:projectId/branch")
	{
		projectBranch.GET("", m.getProjectBranchListHandler)
		projectBranch.PUT("", m.updateProjectBranchListHandler)
		projectBranch.POST("", m.createProjectBranchHandler)
	}
}

// getProjectBranchListHandler godoc
// @id getProjectBranchList
// @summary Get list of branches.
// @tags branch
// @param projectId path uint true "project ID" minimum(0)
// @success 200 {object} response.PaginatedBranches "Branches"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Project not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /project/{projectId}/branch [get]
func (m branchModule) getProjectBranchListHandler(c *gin.Context) {
	projectID, ok := ginutil.ParseParamUint(c, "projectId")
	if !ok {
		return
	}
	if !checkProjectExistsByID(c, m.Database, projectID, "when fetching list of branches for project") {
		return
	}
	var dbBranches []database.Branch
	err := m.Database.
		Where(&database.Branch{ProjectID: projectID}).
		Find(&dbBranches).Error
	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching list of branches for project with ID %d.",
			projectID))
		return
	}
	dbDefaultBranch := findDefaultDBBranch(dbBranches)
	c.JSON(http.StatusOK, modelconv.DBBranchListToPaginatedResponse(dbBranches, int64(len(dbBranches)), dbDefaultBranch))
}

// createProjectBranchHandler godoc
// @id createProjectBranch
// @summary Add branch to project.
// @description Adds a branch to the project, and allows you to set this new branch to be the default branch.
// @description Will ignore name collisions, and treat them as if the branch was just created anyway.
// @tags branch
// @accept json
// @produce json
// @param projectId path uint true "project ID" minimum(0)
// @param branch body request.Branch true "Branch object"
// @success 201 {object} response.Branch "Created branch"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Project not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /project/{projectId}/branch [post]
func (m branchModule) createProjectBranchHandler(c *gin.Context) {
	projectID, ok := ginutil.ParseParamUint(c, "projectId")
	if !ok {
		return
	}
	var reqBranch request.Branch
	if err := c.ShouldBindJSON(&reqBranch); err != nil {
		ginutil.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for branch object to create.")
		return
	}
	dbProject, ok := fetchProjectByIDSlim(c, m.Database, projectID, "when creating branch for project")
	if !ok {
		return
	}
	tokenID := ptrconv.UintPtr(dbProject.TokenID)
	var dbBranch database.Branch
	err := m.Database.Transaction(func(tx *gorm.DB) error {
		dbBranch = database.Branch{
			ProjectID: projectID,
			Default:   reqBranch.Default,
			Name:      reqBranch.Name,
			TokenID:   tokenID,
		}
		if err := tx.Where(&database.Branch{
			ProjectID: projectID,
			Name:      reqBranch.Name,
		}).FirstOrCreate(&dbBranch).Error; err != nil {
			return err
		}
		if reqBranch.Default {
			if err := setDefaultBranchByName(tx, projectID, reqBranch.Name); err != nil {
				return err
			}
		}
		return tx.First(&dbBranch, dbBranch.BranchID).Error
	})
	if err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed creating branch for project with ID %d.",
			projectID))
		return
	}
	c.JSON(http.StatusCreated, modelconv.DBBranchToResponse(dbBranch))
}

// updateProjectBranchListHandler godoc
// @id updateProjectBranchList
// @summary Resets branches for a project
// @description For a given project, it will remove all branches unlisted in the request,
// @description and add all branches from the request that are missing in the database.
// @description Effectively resetting the list of branches to the list from the HTTP request body.
// @description Useful when used via the provider APIs when importing a project.
// @tags branch
// @accept json
// @produce json
// @param projectId path uint true "project ID" minimum(0)
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
	dbBranchList, err := updateBranchList(m.Database, projectID, ptrconv.UintPtr(dbProject.TokenID), reqBranchListUpdate)
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

func updateBranchList(db *gorm.DB, projectID uint, tokenID uint, reqUpdate request.BranchListUpdate) (databaseBranchList, error) {
	if err := ensureOnlyRequestedBranchesExist(db, projectID, tokenID, reqUpdate); err != nil {
		log.Error().
			WithError(err).
			Message("Failed to replace branch list. Transaction rolled back.")
		return databaseBranchList{}, err
	}

	var dbNewBranches []database.Branch
	if err := db.
		Where(&database.Branch{ProjectID: projectID}, database.BranchFields.ProjectID).
		Find(&dbNewBranches).Error; err != nil {
		return databaseBranchList{}, err
	}

	return databaseBranchList{
		defaultBranch: findDefaultDBBranch(dbNewBranches),
		branches:      dbNewBranches,
	}, nil
}

func ensureOnlyRequestedBranchesExist(db *gorm.DB, projectID uint, tokenID uint, reqUpdate request.BranchListUpdate) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var dbOldBranches []database.Branch
		if err := tx.
			Where(&database.Branch{ProjectID: projectID}, database.BranchFields.ProjectID).
			Find(&dbOldBranches).Error; err != nil {
			return err
		}

		wantBranchNamesSet := modelconv.ReqBranchUpdatesToSetOfNames(reqUpdate.Branches)
		hasBranchNamesSet := modelconv.DBBranchesToSetOfNames(dbOldBranches)

		branchNamesToDelete := hasBranchNamesSet.Difference(wantBranchNamesSet)
		branchNamesToAdd := wantBranchNamesSet.Difference(hasBranchNamesSet)

		if len(branchNamesToAdd) > 0 {
			if err := createBranchesWithNames(tx, projectID, tokenID, branchNamesToAdd.Slice()); err != nil {
				return err
			}
			log.Info().
				WithInt("branchesAdded", len(branchNamesToAdd)).
				WithUint("project", projectID).
				Message("Added branches to project when updating branches.")
		}

		if len(branchNamesToDelete) > 0 {
			if err := deleteBranchesByNames(tx, projectID, branchNamesToDelete.Slice()); err != nil {
				return err
			}
			log.Info().
				WithInt("branchesDeleted", len(branchNamesToDelete)).
				WithUint("project", projectID).
				Message("Deleted branches from project when updating branches.")
		}

		return setDefaultBranchByName(tx, projectID, reqUpdate.DefaultBranch)
	})
}

func findDefaultDBBranch(dbBranches []database.Branch) *database.Branch {
	for _, dbNewBranch := range dbBranches {
		if dbNewBranch.Default {
			return &dbNewBranch
		}
	}
	return nil
}

func createBranchesWithNames(db *gorm.DB, projectID, tokenID uint, branchNames []string) error {
	var dbBranches []database.Branch
	for _, branchName := range branchNames {
		dbBranches = append(dbBranches, database.Branch{
			ProjectID: projectID,
			TokenID:   tokenID,
			Name:      branchName,
		})
	}
	return db.Create(dbBranches).Error
}

func deleteBranchesByNames(db *gorm.DB, projectID uint, branchNames []string) error {
	return db.
		Where(&database.Branch{ProjectID: projectID}, database.BranchFields.ProjectID).
		Where(database.BranchColumns.Name+" IN ?", stringSliceToInterfaces(branchNames)).
		Delete(&database.Branch{}).
		Error
}

func setDefaultBranchByName(db *gorm.DB, projectID uint, defaultBranchName string) error {
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

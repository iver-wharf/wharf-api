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

// ProjectUpdate specifies fields when creating a new token using the deprecated
// endpoint
// 	PUT /project
type ProjectUpdate struct {
	ProjectID       uint   `json:"projectId"`
	Name            string `json:"name" validate:"required" binding:"required"`
	GroupName       string `json:"groupName"`
	Description     string `json:"description"`
	AvatarURL       string `json:"avatarUrl"`
	TokenID         uint   `json:"tokenId"`
	ProviderID      uint   `json:"providerId"`
	BuildDefinition string `json:"buildDefinition"`
	GitURL          string `json:"gitUrl"`
}

// ProjectModule holds deprecated endpoint handlers for /project
type ProjectModule struct {
	Database *gorm.DB
}

// Register adds all deprecated endpoints to a given Gin router group.
func (m ProjectModule) Register(g *gin.RouterGroup) {
	project := g.Group("/project")
	{
		project.PUT("", m.updateProjectHandler)
	}
}

// updateProjectHandler godoc
// @id updateProject
// @deprecated
// @summary Adds project when not exists.
// @description It finds project by ID or if ID is set to 0 it takes group name.
// @description First found project will be returned. If not found project will be added into database.
// @description It ignores branches array, build history and provider params.
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use POST /project to create, or PUT /project/{projectId} to update instead.
// @tags project
// @accept json
// @produce json
// @param project body request.ProjectUpdate _ "project object"
// @success 200 {object} response.Project "Project was updated"
// @success 201 {object} response.Project "A new project was created"
// @failure 400 {object} problem.Response "Bad request, such as invalid body JSON"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Project to update was not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /project [put]
func (m ProjectModule) updateProjectHandler(c *gin.Context) {
	var reqProjectUpdate ProjectUpdate
	err := c.ShouldBindJSON(&reqProjectUpdate)
	if err != nil {
		ginutil.WriteInvalidBindError(c, err, "One or more parameters failed to parse when reading the request body.")
		return
	}

	var dbExistingProject database.Project
	if reqProjectUpdate.ProjectID != 0 {
		dbExistingProject, err = m.findProjectByID(reqProjectUpdate.ProjectID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ginutil.WriteDBNotFound(c, fmt.Sprintf(
				"Project with ID %d was not found in the database.",
				reqProjectUpdate.ProjectID))
			return
		} else if err != nil {
			ginutil.WriteDBReadError(c, err, fmt.Sprintf(
				"Failed fetching project by ID %d from database.",
				reqProjectUpdate.ProjectID))
			return
		}
	} else {
		err := m.Database.
			Where(&database.Project{
				Name:      reqProjectUpdate.Name,
				GroupName: reqProjectUpdate.GroupName,
			}, database.ProjectFields.Name, database.ProjectFields.GroupName).
			First(&dbExistingProject).
			Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dbNewProject := database.Project{
				ProjectID:       reqProjectUpdate.ProjectID,
				Name:            reqProjectUpdate.Name,
				GroupName:       reqProjectUpdate.GroupName,
				Description:     reqProjectUpdate.Description,
				AvatarURL:       reqProjectUpdate.AvatarURL,
				TokenID:         reqProjectUpdate.TokenID,
				ProviderID:      reqProjectUpdate.ProjectID,
				BuildDefinition: reqProjectUpdate.BuildDefinition,
				GitURL:          reqProjectUpdate.GitURL,
			}
			if err := m.Database.Create(&dbNewProject).Error; err != nil {
				ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
					"Failed creating new project with group %q, token ID %d, and name %q in database.",
					reqProjectUpdate.GroupName, reqProjectUpdate.TokenID, reqProjectUpdate.Name))
			} else {
				resProject := modelconv.DBProjectToResponse(dbNewProject)
				c.JSON(http.StatusCreated, resProject)
			}
			return
		} else if err != nil {
			ginutil.WriteDBReadError(c, err, fmt.Sprintf(
				"Failed fetching project with name %q and group name %q from the database.",
				reqProjectUpdate.Name, reqProjectUpdate.GroupName))
			return
		}
	}

	dbExistingProject.Name = reqProjectUpdate.Name
	dbExistingProject.GroupName = reqProjectUpdate.GroupName
	dbExistingProject.Description = reqProjectUpdate.Description
	dbExistingProject.AvatarURL = reqProjectUpdate.AvatarURL
	dbExistingProject.TokenID = reqProjectUpdate.TokenID
	dbExistingProject.ProviderID = reqProjectUpdate.ProviderID
	dbExistingProject.BuildDefinition = reqProjectUpdate.BuildDefinition
	dbExistingProject.GitURL = reqProjectUpdate.GitURL

	if err := m.Database.Save(&dbExistingProject).Error; err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed writing project with name %q and group name %q to database.",
			reqProjectUpdate.Name, reqProjectUpdate.GroupName))
		return
	}

	resProject := modelconv.DBProjectToResponse(dbExistingProject)
	c.JSON(http.StatusOK, resProject)
}

func (m ProjectModule) findProjectByID(id uint) (database.Project, error) {
	var dbProject database.Project
	err := m.databaseProjectPreloaded().
		Where(&database.Project{ProjectID: id}).
		First(&dbProject).
		Error
	return dbProject, err
}

func (m ProjectModule) databaseProjectPreloaded() *gorm.DB {
	return m.Database.Set("gorm:auto_preload", false).
		Preload(database.ProjectFields.Provider).
		Preload(database.ProjectFields.Branches, func(db *gorm.DB) *gorm.DB {
			return db.Order(database.BranchColumns.BranchID)
		}).
		Preload(database.ProjectFields.Token)
}

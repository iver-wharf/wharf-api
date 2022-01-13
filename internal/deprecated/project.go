package deprecated

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/v5/internal/ptrconv"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/response"
	"github.com/iver-wharf/wharf-api/v5/pkg/modelconv"
	"github.com/iver-wharf/wharf-api/v5/pkg/orderby"
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
	TokenID         uint   `json:"tokenId" extensions:"x-nullable"`
	ProviderID      uint   `json:"providerId" extensions:"x-nullable"`
	BuildDefinition string `json:"buildDefinition"`
	GitURL          string `json:"gitUrl"`
}

// ProjectSearch holds values used in verbatim searches for projects using the
// deprecated endpoint
// 	POST /projects/search
type ProjectSearch struct {
	Name            string `json:"name"`
	GroupName       string `json:"groupName"`
	Description     string `json:"description"`
	AvatarURL       string `json:"avatarUrl"`
	TokenID         uint   `json:"tokenId" minimum:"0" extensions:"x-nullable"`
	ProviderID      uint   `json:"providerId" minimum:"0" extensions:"x-nullable"`
	BuildDefinition string `json:"buildDefinition"`
	GitURL          string `json:"gitUrl"`
}

// ProjectModule holds deprecated endpoint handlers for /project
type ProjectModule struct {
	Database *gorm.DB
}

// Register adds all deprecated endpoints to a given Gin router group.
func (m ProjectModule) Register(g *gin.RouterGroup) {
	projects := g.Group("/projects")
	{
		projects.GET("", m.getProjectListHandler)
		projects.POST("/search", m.searchProjectListHandler)
		projects.GET("/:projectId/builds", m.getProjectBuildListHandler)
	}

	project := g.Group("/project")
	{
		project.PUT("", m.updateProjectHandler)
	}
}

// getProjectListHandler godoc
// @id oldGetProjectList
// @deprecated
// @summary Returns all projects from database
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use `GET /project` instead.
// @description Added in v0.1.8.
// @tags project
// @success 200 {object} []response.Project
// @failure 502 {object} problem.Response "Database is unreachable"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @router /projects [get]
func (m ProjectModule) getProjectListHandler(c *gin.Context) {
	var dbProjects []database.Project
	err := m.databaseProjectPreloaded().
		Find(&dbProjects).
		Error
	if err != nil {
		ginutil.WriteDBReadError(c, err, "Failed fetching list of projects from database.")
		return
	}
	resProjects := modelconv.DBProjectsToResponses(dbProjects)
	c.JSON(http.StatusOK, resProjects)
}

// searchProjectListHandler godoc
// @id oldSearchProjectList
// @deprecated
// @summary Search for projects from database
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use `GET /project` instead.
// @description Added in v0.5.5.
// @tags project
// @param project body ProjectSearch _ "Project search criteria"
// @success 200 {object} []response.Project
// @failure 502 {object} problem.Response "Database is unreachable"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @router /projects/search [post]
func (m ProjectModule) searchProjectListHandler(c *gin.Context) {
	var reqProjectSearch ProjectSearch
	if err := c.ShouldBindJSON(&reqProjectSearch); err != nil {
		ginutil.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for the project object to search with.")
		return
	}
	var dbProjects []database.Project
	err := m.Database.
		Where(&database.Project{
			Name:            reqProjectSearch.Name,
			GroupName:       reqProjectSearch.GroupName,
			Description:     reqProjectSearch.Description,
			AvatarURL:       reqProjectSearch.AvatarURL,
			TokenID:         ptrconv.UintZeroNil(reqProjectSearch.TokenID),
			ProviderID:      ptrconv.UintZeroNil(reqProjectSearch.ProviderID),
			BuildDefinition: reqProjectSearch.BuildDefinition,
			GitURL:          reqProjectSearch.GitURL,
		}).
		Preload(database.ProjectFields.Provider).
		Find(&dbProjects).
		Error
	if err != nil {
		ginutil.WriteDBReadError(c, err, "Failed searching for projects in database.")
		return
	}
	resProjects := modelconv.DBProjectsToResponses(dbProjects)
	c.JSON(http.StatusOK, resProjects)
}

var buildJSONToColumns = map[string]database.SafeSQLName{
	response.BuildJSONFields.BuildID:     database.BuildColumns.BuildID,
	response.BuildJSONFields.Environment: database.BuildColumns.Environment,
	response.BuildJSONFields.CompletedOn: database.BuildColumns.CompletedOn,
	response.BuildJSONFields.ScheduledOn: database.BuildColumns.ScheduledOn,
	response.BuildJSONFields.StartedOn:   database.BuildColumns.StartedOn,
	response.BuildJSONFields.Stage:       database.BuildColumns.Stage,
	response.BuildJSONFields.StatusID:    database.BuildColumns.StatusID,
	response.BuildJSONFields.IsInvalid:   database.BuildColumns.IsInvalid,
}

// PaginatedBuilds is a list of builds as well as an explicit total count field.
type PaginatedBuilds struct {
	Builds     []response.Build `json:"builds"`
	TotalCount int64            `json:"totalCount"`
}

// getProjectBuildListHandler godoc
// @id oldGetProjectBuildList
// @deprecated
// @summary Get slice of builds.
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use `GET /build?projectId=123` instead.
// @description Added in v1.0.1.
// @tags project
// @param projectId path uint true "project ID" minimum(0)
// @param limit query string true "number of fetched branches"
// @param offset query string true "PK of last branch taken"
// @param orderby query []string false "Sorting orders. Takes the property name followed by either 'asc' or 'desc'. Can be specified multiple times for more granular sorting. Defaults to `?orderby=buildId desc`"
// @success 200 {object} PaginatedBuilds
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /projects/{projectId}/builds [get]
func (m ProjectModule) getProjectBuildListHandler(c *gin.Context) {
	projectID, ok := ginutil.ParseParamUint(c, "projectId")
	if !ok {
		return
	}
	limit, ok := ginutil.ParseQueryInt(c, "limit")
	if !ok {
		return
	}
	offset, ok := ginutil.ParseQueryInt(c, "offset")
	if !ok {
		return
	}
	orderByQueryParams := c.QueryArray("orderby")
	orderBySlice, err := orderby.ParseSlice(orderByQueryParams, buildJSONToColumns)
	if err != nil {
		joinedOrders := strings.Join(orderByQueryParams, ", ")
		ginutil.WriteInvalidParamError(c, err, "orderby", fmt.Sprintf(
			"Failed parsing the %d sort ordering values: %s",
			len(orderByQueryParams),
			joinedOrders))
		return
	}

	dbBuilds, err := m.getBuilds(projectID, limit, offset, orderBySlice)
	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching builds for project with ID %d from database.",
			projectID))
		return
	}

	count, err := m.getBuildsCount(projectID)
	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching build count for project with ID %d from database.",
			projectID))
		return
	}

	resPaginated := PaginatedBuilds{
		Builds:     modelconv.DBBuildsToResponses(dbBuilds),
		TotalCount: count,
	}
	c.JSON(http.StatusOK, resPaginated)
}

// updateProjectHandler godoc
// @id oldUpdateProject
// @deprecated
// @summary Adds project when not exists.
// @description It finds project by ID or if ID is set to 0 it takes group name.
// @description First found project will be returned. If not found project will be added into database.
// @description It ignores branches array, build history and provider params.
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use POST /project to create, or PUT /project/{projectId} to update instead.
// @description Added in v0.2.13.
// @tags project
// @accept json
// @produce json
// @param project body deprecated.ProjectUpdate _ "project object"
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
				TokenID:         ptrconv.UintZeroNil(reqProjectUpdate.TokenID),
				ProviderID:      ptrconv.UintZeroNil(reqProjectUpdate.ProviderID),
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
	dbExistingProject.TokenID = ptrconv.UintZeroNil(reqProjectUpdate.TokenID)
	dbExistingProject.ProviderID = ptrconv.UintZeroNil(reqProjectUpdate.ProviderID)
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

var defaultGetBuildsOrderBy = orderby.Column{Name: database.BuildColumns.BuildID, Direction: orderby.Desc}

func (m ProjectModule) getBuilds(projectID uint, limit int, offset int, orderBySlice orderby.Slice) ([]database.Build, error) {
	var dbBuilds []database.Build
	var query = m.Database.
		Where(&database.Build{ProjectID: projectID}).
		Preload(database.BuildFields.TestResultSummaries).
		Limit(limit).
		Offset(offset).
		Clauses(orderBySlice.ClauseIfNone(defaultGetBuildsOrderBy))
	if err := query.Find(&dbBuilds).Error; err != nil {
		return nil, err
	}

	return dbBuilds, nil
}

func (m ProjectModule) getBuildsCount(projectID uint) (int64, error) {
	var count int64
	if err := m.Database.
		Model(&database.Build{}).
		Where(&database.Build{ProjectID: projectID}).
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return count, nil
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

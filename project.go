package main

import (
	"errors"
	"fmt"

	"github.com/iver-wharf/wharf-core/pkg/ginutil"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/internal/ptrconv"
	"github.com/iver-wharf/wharf-api/internal/wherefields"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/request"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
	"github.com/iver-wharf/wharf-api/pkg/modelconv"
	"github.com/iver-wharf/wharf-api/pkg/orderby"
	"gorm.io/gorm"
)

type projectModule struct {
	Database *gorm.DB
}

func (m projectModule) Register(g *gin.RouterGroup) {
	project := g.Group("/project")
	{
		project.GET("", m.getProjectListHandler)
		project.POST("", m.createProjectHandler)

		projectByID := project.Group("/:projectId")
		{
			projectByID.GET("", m.getProjectHandler)
			projectByID.DELETE("", m.deleteProjectHandler)
			projectByID.PUT("", m.updateProjectHandler)

			override := projectByID.Group("/override")
			{
				override.GET("", m.getProjectOverridesHandler)
				override.PUT("", m.updateProjectOverridesHandler)
				override.DELETE("", m.deleteProjectOverridesHandler)
			}
		}
	}
}

var projectJSONToColumns = map[string]database.SafeSQLName{
	response.ProjectJSONFields.ProjectID:   database.ProjectColumns.ProjectID,
	response.ProjectJSONFields.Name:        database.ProjectColumns.Name,
	response.ProjectJSONFields.GroupName:   database.ProjectColumns.GroupName,
	response.ProjectJSONFields.Description: database.ProjectColumns.Description,
	response.ProjectJSONFields.GitURL:      database.ProjectColumns.GitURL,
}

var defaultGetProjectsOrderBy = orderby.Column{Name: database.ProjectColumns.ProjectID, Direction: orderby.Desc}

// getProjectListHandler godoc
// @id getProjectList
// @summary Returns all projects from database
// @description List all projects, or a window of projects using the `limit` and `offset` query parameters. Allows optional filtering parameters.
// @description Verbatim filters will match on the entire string used to find exact matches,
// @description while the matching filters are meant for searches by humans where it tries to find soft matches and is therefore inaccurate by nature.
// @tags project
// @param orderby query []string false "Sorting orders. Takes the property name followed by either 'asc' or 'desc'. Can be specified multiple times for more granular sorting. Defaults to `?orderby=projectId desc`"
// @param limit query int false "Number of results to return. No limiting is applied if empty (`?limit=`) or non-positive (`?limit=0`). Required if `offset` is used." default(100)
// @param offset query int false "Skipped results, where 0 means from the start." minimum(0) default(0)
// @param name query string false "Filter by verbatim project name."
// @param groupName query string false "Filter by verbatim project group."
// @param description query string false "Filter by verbatim description."
// @param tokenId query uint false "Filter by token ID. Zero (0) will search for null values." minimum(0)
// @param providerId query uint false "Filter by provider ID. Zero (0) will search for null values." minimum(0)
// @param gitUrl query string false "Filter by verbatim Git URL."
// @param nameMatch query string false "Filter by matching project name. Cannot be used with `name`."
// @param groupNameMatch query string false "Filter by matching project group. Cannot be used with `groupName`."
// @param descriptionMatch query string false "Filter by matching description. Cannot be used with `description`."
// @param gitUrlMatch query string false "Filter by matching Git URL. Cannot be used with `gitUrl`."
// @param match query string false "Filter by matching on any supported fields."
// @success 200 {object} response.PaginatedProjects
// @failure 502 {object} problem.Response "Database is unreachable"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @router /project [get]
func (m projectModule) getProjectListHandler(c *gin.Context) {
	var params = struct {
		commonGetQueryParams

		Name        *string `form:"name"`
		GroupName   *string `form:"groupName"`
		Description *string `form:"description"`
		TokenID     *uint   `form:"tokenId"`
		ProviderID  *uint   `form:"providerId"`
		GitURL      *string `form:"gitUrl"`

		NameMatch        *string `form:"nameMatch" binding:"excluded_with=Name"`
		GroupNameMatch   *string `form:"groupNameMatch" binding:"excluded_with=GroupName"`
		DescriptionMatch *string `form:"descriptionMatch" binding:"excluded_with=Description"`
		GitURLMatch      *string `form:"gitUrlMatch" binding:"excluded_with=GitURL"`

		Match *string `form:"match"`
	}{
		commonGetQueryParams: defaultCommonGetQueryParams,
	}
	if !bindCommonGetQueryParams(c, &params) {
		return
	}
	orderBySlice, ok := parseCommonOrderBySlice(c, params.OrderBy, providerJSONToColumns)
	if !ok {
		return
	}

	var where wherefields.Collection
	query := databaseProjectPreloaded(m.Database).
		Clauses(orderBySlice.ClauseIfNone(defaultGetProjectsOrderBy)).
		Where(&database.Project{
			Name:       where.String(database.ProjectFields.Name, params.Name),
			GroupName:  where.String(database.ProjectFields.GroupName, params.GroupName),
			TokenID:    where.UintPtrZeroNil(database.ProjectFields.TokenID, params.TokenID),
			ProviderID: where.UintPtrZeroNil(database.ProjectFields.ProviderID, params.ProviderID),
			GitURL:     where.String(database.ProjectFields.GitURL, params.GitURL),
		}, where.NonNilFieldNames()...).
		Scopes(
			whereLikeScope(map[database.SafeSQLName]*string{
				database.ProjectColumns.Name:        params.NameMatch,
				database.ProjectColumns.GroupName:   params.GroupNameMatch,
				database.ProjectColumns.Description: params.DescriptionMatch,
				database.ProjectColumns.GitURL:      params.GitURLMatch,
			}),
			whereAnyLikeScope(
				params.Match,
				database.ProjectColumns.Name,
				database.ProjectColumns.GroupName,
				database.ProjectColumns.Description,
				database.ProjectColumns.GitURL,
			),
		)

	var dbProjects []database.Project
	var totalCount int64
	err := findDBPaginatedSliceAndTotalCount(query, params.Limit, params.Offset, &dbProjects, &totalCount)
	if err != nil {
		ginutil.WriteDBReadError(c, err, "Failed fetching list of projects from database.")
		return
	}

	c.JSON(http.StatusOK, response.PaginatedProjects{
		List:       modelconv.DBProjectsToResponses(dbProjects),
		TotalCount: totalCount,
	})
}

// getProjectHandler godoc
// @id getProject
// @summary Returns project with selected project ID
// @tags project
// @param projectId path uint true "project ID" minimum(0)
// @success 200 {object} response.Project
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Project not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /project/{projectId} [get]
func (m projectModule) getProjectHandler(c *gin.Context) {
	projectID, ok := ginutil.ParseParamUint(c, "projectId")
	if !ok {
		return
	}
	dbProject, ok := fetchProjectByID(c, m.Database, projectID, "")
	if !ok {
		return
	}
	resProject := modelconv.DBProjectToResponse(dbProject)
	c.JSON(http.StatusOK, resProject)
}

// createProjectHandler godoc
// @id createProject
// @summary Creates project
// @description Add project to database.
// @tags project
// @accept json
// @produce json
// @param project body request.Project true "Project to create"
// @success 201 {object} response.Project
// @failure 400 {object} problem.Response "Bad request"
// @failure 404 {object} problem.Response "Project to update is not found"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /project [post]
func (m projectModule) createProjectHandler(c *gin.Context) {
	var reqProject request.Project
	if err := c.ShouldBindJSON(&reqProject); err != nil {
		ginutil.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for the project object to update.")
		return
	}

	dbProject := modelconv.ReqProjectToDatabase(reqProject)
	if err := m.Database.Create(&dbProject).Error; err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed creating new project with group %q, token ID %d, and name %q in database.",
			reqProject.GroupName, reqProject.TokenID, reqProject.Name))
		return
	}

	resProject := modelconv.DBProjectToResponse(dbProject)
	c.JSON(http.StatusCreated, resProject)
}

// deleteProjectHandler godoc
// @id deleteProject
// @summary Delete project with selected project ID
// @tags project
// @param projectId path uint true "project ID" minimum(0)
// @success 204 "Deleted"
// @failure 502 {object} problem.Response "Database is unreachable"
// @failure 400 {object} problem.Response "Bad request"
// @failure 404 {object} problem.Response "Project to delete is not found"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @router /project/{projectId} [delete]
func (m projectModule) deleteProjectHandler(c *gin.Context) {
	projectID, ok := ginutil.ParseParamUint(c, "projectId")
	if !ok {
		return
	}
	dbProject, ok := fetchProjectByID(c, m.Database, projectID, "when deleting project")
	if !ok {
		return
	}
	if err := m.Database.Delete(&dbProject).Error; err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf("Failed deleting project with ID %d from database.", projectID))
		return
	}

	c.Status(http.StatusNoContent)
}

// updateProjectHandler godoc
// @id updateProject
// @summary Update project in database
// @description Updates a project by replacing all of its fields.
// @tags project
// @accept json
// @produce json
// @param projectId path uint true "project ID" minimum(0)
// @param project body request.ProjectUpdate _ "New project values"
// @success 200 {object} response.Project
// @failure 400 {object} problem.Response "Bad request, such as invalid body JSON"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Project to update was not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /project/{projectId} [put]
func (m projectModule) updateProjectHandler(c *gin.Context) {
	projectID, ok := ginutil.ParseParamUint(c, "projectId")
	if !ok {
		return
	}
	var reqProjectUpdate request.ProjectUpdate
	err := c.ShouldBindJSON(&reqProjectUpdate)
	if err != nil {
		ginutil.WriteInvalidBindError(c, err, "One or more parameters failed to parse when reading the request body.")
		return
	}
	dbProject, ok := fetchProjectByID(c, m.Database, projectID, "when updating project")
	if !ok {
		return
	}

	dbProject.Name = reqProjectUpdate.Name
	dbProject.GroupName = reqProjectUpdate.GroupName
	dbProject.Description = reqProjectUpdate.Description
	dbProject.AvatarURL = reqProjectUpdate.AvatarURL
	dbProject.TokenID = ptrconv.UintZeroNil(reqProjectUpdate.TokenID)
	dbProject.ProviderID = ptrconv.UintZeroNil(reqProjectUpdate.ProviderID)
	dbProject.BuildDefinition = reqProjectUpdate.BuildDefinition
	dbProject.GitURL = reqProjectUpdate.GitURL

	if err := m.Database.Save(&dbProject).Error; err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed writing project with name %q and group name %q to database.",
			reqProjectUpdate.Name, reqProjectUpdate.GroupName))
		return
	}

	resProject := modelconv.DBProjectToResponse(dbProject)
	c.JSON(http.StatusOK, resProject)
}

// getProjectOverridesHandler godoc
// @id getProjectOverrides
// @summary Get project overrides
// @description Get values for a project's overridable fields.
// @description Meant for manual overrides.
// @description Overridden field will take precedence when retreiving the project or in newly started builds,
// @description but will stay unaffected by regular project updates.
// @tags project
// @produce json
// @param projectId path uint true "project ID" minimum(0)
// @success 200 {object} response.ProjectOverrides
// @failure 400 {object} problem.Response "Bad request, such as invalid body JSON"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Project to update was not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /project/{projectId}/override [get]
func (m projectModule) getProjectOverridesHandler(c *gin.Context) {
	projectID, ok := ginutil.ParseParamUint(c, "projectId")
	if !ok {
		return
	}

	var dbProjectOverrides database.ProjectOverrides
	err := m.Database.
		Where(&database.ProjectOverrides{
			ProjectID: projectID,
		}).
		First(&dbProjectOverrides).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// fake that it exists
		dbProjectOverrides.ProjectID = projectID
	} else if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed reading project overrides for project with ID %d to database.",
			projectID))
		return
	}

	resProject := modelconv.DBProjectOverridesToResponse(dbProjectOverrides)
	c.JSON(http.StatusOK, resProject)
}

// updateProjectOverridesHandler godoc
// @id updateProjectOverrides
// @summary Update project overrides in database
// @description Updates a project by replacing all of its overridable fields.
// @description Meant for manual overrides.
// @description Overridden field will take precedence when retreiving the project or in newly started builds,
// @description but will stay unaffected by regular project updates.
// @tags project
// @accept json
// @produce json
// @param projectId path uint true "project ID" minimum(0)
// @param overrides body request.ProjectOverridesUpdate _ "New project overrides"
// @success 200 {object} response.ProjectOverrides
// @failure 400 {object} problem.Response "Bad request, such as invalid body JSON"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Project to update was not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /project/{projectId}/override [put]
func (m projectModule) updateProjectOverridesHandler(c *gin.Context) {
	projectID, ok := ginutil.ParseParamUint(c, "projectId")
	if !ok {
		return
	}
	var reqOverridesUpdate request.ProjectOverridesUpdate
	err := c.ShouldBindJSON(&reqOverridesUpdate)
	if err != nil {
		ginutil.WriteInvalidBindError(c, err, "One or more parameters failed to parse when reading the request body.")
		return
	}

	var dbProjectOverrides database.ProjectOverrides
	err = m.Database.
		Where(&database.ProjectOverrides{
			ProjectID: projectID,
		}).
		FirstOrCreate(&dbProjectOverrides).Error
	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed reading project overrides for project with ID %d to database.",
			projectID))
		return
	}

	dbProjectOverrides.Description = reqOverridesUpdate.Description
	dbProjectOverrides.AvatarURL = reqOverridesUpdate.AvatarURL
	dbProjectOverrides.GitURL = reqOverridesUpdate.GitURL

	if err := m.Database.Save(&dbProjectOverrides).Error; err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed writing project overrides for project with ID %d to database.",
			projectID))
		return
	}

	resProject := modelconv.DBProjectOverridesToResponse(dbProjectOverrides)
	c.JSON(http.StatusOK, resProject)
}

// deleteProjectOverridesHandler godoc
// @id deleteProjectOverrides
// @summary Delete project's overrides with selected project ID
// @description This will revert all overrides to the specified project.
// @description Equivalent to running `PUT /project/{projectId}/overrides` with all fields set to `null`.
// @tags project
// @param projectId path uint true "project ID" minimum(0)
// @success 204 "Deleted"
// @failure 502 {object} problem.Response "Database is unreachable"
// @failure 400 {object} problem.Response "Bad request"
// @failure 404 {object} problem.Response "Project to delete overrides from is not found"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @router /project/{projectId}/override [delete]
func (m projectModule) deleteProjectOverridesHandler(c *gin.Context) {
	projectID, ok := ginutil.ParseParamUint(c, "projectId")
	if !ok {
		return
	}
	err := m.Database.
		Where(&database.ProjectOverrides{
			ProjectID: projectID,
		}).
		Delete(&database.ProjectOverrides{}).Error
	if err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf("Failed deleting project overrides for project with ID %d from database.", projectID))
		return
	}
	c.Status(http.StatusNoContent)
}

func fetchProjectByID(c *gin.Context, db *gorm.DB, projectID uint, whenMsg string) (database.Project, bool) {
	var dbProject database.Project
	ok := fetchDatabaseObjByID(c, databaseProjectPreloaded(db), &dbProject, projectID, "project", whenMsg)
	return dbProject, ok
}

func fetchProjectByIDSlim(c *gin.Context, db *gorm.DB, projectID uint, whenMsg string) (database.Project, bool) {
	var dbProject database.Project
	ok := fetchDatabaseObjByID(c, db, &dbProject, projectID, "project", whenMsg)
	return dbProject, ok
}

func databaseProjectPreloaded(db *gorm.DB) *gorm.DB {
	return db.Set("gorm:auto_preload", false).
		Preload(database.ProjectFields.Provider).
		Preload(database.ProjectFields.Branches, func(db *gorm.DB) *gorm.DB {
			return db.Order(database.BranchColumns.BranchID)
		}).
		Preload(database.ProjectFields.Token).
		Preload(database.ProjectFields.Overrides)
}

func (m projectModule) getBuildsCount(projectID uint) (int64, error) {
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

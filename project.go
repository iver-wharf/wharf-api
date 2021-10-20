package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/iver-wharf/wharf-core/pkg/ginutil"

	"encoding/json"
	"net/http"
	"net/url"

	"github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/messagebus-go"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/request"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
	"github.com/iver-wharf/wharf-api/pkg/modelconv"
	"github.com/iver-wharf/wharf-api/pkg/orderby"
	"github.com/iver-wharf/wharf-core/pkg/problem"
	"gopkg.in/guregu/null.v4"
	"gorm.io/gorm"
)

type projectModule struct {
	Database     *gorm.DB
	MessageQueue *messagebus.MQConnection
	Config       *Config
}

func (m projectModule) Register(g *gin.RouterGroup) {
	projects := g.Group("/projects")
	{
		projects.GET("", m.getProjectListHandler)
		projects.POST("/search", m.searchProjectListHandler)
		projects.GET("/:projectId/builds", m.getProjectBuildListHandler)
	}

	project := g.Group("/project")
	{
		project.GET("/:projectId", m.getProjectHandler)
		project.POST("", m.createProjectHandler)
		project.DELETE("/:projectId", m.deleteProjectHandler)
		project.PUT("/:projectId", m.updateProjectHandler)
		project.POST("/:projectId/:stage/run", m.startProjectBuildHandler)
	}
}

// getProjectListHandler godoc
// @id getProjectList
// @summary Returns all projects from database
// @description List all projects, or a window of projects using the `limit` and
// @description `offset` query parameters. Allows optional filtering parameters.
// @description Verbatim filters will match on the entire string, while the
// @description matching filters will search on substrings.
// @tags project
// @param limit query int false "Number of results to return. No limit if unset or non-positive. Required if offset is used."
// @param offset query int false "Skipped results, where 0 means from the start." minimum(0)
// @param name query string false "Filter by verbatim project name" extensions(x-nullable)
// @param groupName query string false "Filter by verbatim project group" extensions(x-nullable)
// @param description query string false "Filter by verbatim description" extensions(x-nullable)
// @param tokenId query uint false "Filter by token ID. Zero (0) will search for null values." minimum(0) extensions(x-nullable)
// @param providerId query uint false "Filter by provider ID. Zero (0) will search for null values." minimum(0) extensions(x-nullable)
// @success 200 {object} []response.Project
// @failure 502 {object} problem.Response "Database is unreachable"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @router /projects [get]
func (m projectModule) getProjectListHandler(c *gin.Context) {
	var params = struct {
		Limit       int     `form:"limit" binding:"required_with=Offset"`
		Offset      int     `form:"offset" binding:"min=0"`
		Name        *string `form:"name"`
		GroupName   *string `form:"groupName"`
		Description *string `form:"description"`
		TokenID     *uint   `form:"tokenId"`
		ProviderID  *uint   `form:"providerId"`
	}{}
	if err := c.ShouldBindQuery(&params); err != nil {
		ginutil.WriteInvalidBindError(c, err, "One or more parameters failed to parse when reading query parameters.")
		return
	}
	var dbProjects []database.Project
	var sc searchCollection
	err := databaseProjectPreloaded(m.Database).
		Clauses(optionalLimitOffsetClause(params.Limit, params.Offset)).
		Where(&database.Project{
			Description: sc.String(database.ProjectFields.Description, params.Description),
			TokenID:     sc.Uint(database.ProjectFields.TokenID, params.TokenID),
			ProviderID:  sc.Uint(database.ProjectFields.ProjectID, params.ProviderID),
		}, sc.fieldNames...).
		Find(&dbProjects).
		Error
	if err != nil {
		ginutil.WriteDBReadError(c, err, "Failed fetching list of projects from database.")
		return
	}
	resProjects := modelconv.DBProjectsToResponses(dbProjects)
	c.JSON(http.StatusOK, resProjects)
}

type searchCollection struct {
	fieldNames []interface{}
}

func (sc *searchCollection) addFieldName(field string) {
	sc.fieldNames = append(sc.fieldNames, field)
}

func (sc *searchCollection) Uint(field string, value *uint) uint {
	if value == nil {
		return 0
	}
	sc.addFieldName(field)
	return *value
}

func (sc *searchCollection) String(field string, value *string) string {
	if value == nil {
		return ""
	}
	sc.addFieldName(field)
	return *value
}

// searchProjectListHandler godoc
// @id searchProjectList
// @summary Search for projects from database
// @tags project
// @param project body request.ProjectSearch _ "Project search criteria"
// @success 200 {object} []response.Project
// @failure 502 {object} problem.Response "Database is unreachable"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @router /projects/search [post]
func (m projectModule) searchProjectListHandler(c *gin.Context) {
	var reqProjectSearch request.ProjectSearch
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
			TokenID:         reqProjectSearch.TokenID,
			ProviderID:      reqProjectSearch.ProviderID,
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

var buildJSONToColumns = map[string]string{
	response.BuildJSONFields.BuildID:     database.BuildColumns.BuildID,
	response.BuildJSONFields.Environment: database.BuildColumns.Environment,
	response.BuildJSONFields.CompletedOn: database.BuildColumns.CompletedOn,
	response.BuildJSONFields.ScheduledOn: database.BuildColumns.ScheduledOn,
	response.BuildJSONFields.StartedOn:   database.BuildColumns.StartedOn,
	response.BuildJSONFields.Stage:       database.BuildColumns.Stage,
	response.BuildJSONFields.StatusID:    database.BuildColumns.StatusID,
	response.BuildJSONFields.IsInvalid:   database.BuildColumns.IsInvalid,
}

// getProjectBuildListHandler godoc
// @id getProjectBuildList
// @summary Get slice of builds.
// @tags project
// @param projectId path uint true "project ID" minimum(0)
// @param limit query string true "number of fetched branches"
// @param offset query string true "PK of last branch taken"
// @param orderby query []string false "Sorting orders. Takes the property name followed by either 'asc' or 'desc'. Can be specified multiple times for more granular sorting. Defaults to '?orderby=buildId desc'"
// @success 200 {object} response.PaginatedBuilds
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /projects/{projectId}/builds [get]
func (m projectModule) getProjectBuildListHandler(c *gin.Context) {
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

	resPaginated := response.PaginatedBuilds{
		Builds:     modelconv.DBBuildsToResponses(dbBuilds),
		TotalCount: count,
	}
	c.JSON(http.StatusOK, resPaginated)
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

	dbProject := database.Project{
		Name:            reqProject.Name,
		GroupName:       reqProject.GroupName,
		Description:     reqProject.Description,
		AvatarURL:       reqProject.AvatarURL,
		TokenID:         reqProject.TokenID,
		ProviderID:      reqProject.ProviderID,
		BuildDefinition: reqProject.BuildDefinition,
		GitURL:          reqProject.GitURL,
	}
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
	dbProject.TokenID = reqProjectUpdate.TokenID
	dbProject.ProviderID = reqProjectUpdate.ProviderID
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

// startProjectBuildHandler godoc
// @id startProjectBuild
// @summary Responsible for run stage environment for selected project
// @tags project
// @accept json
// @param projectId path uint true "project ID" minimum(0)
// @param stage path string true "name of stage to run, or specify ALL to run everything"
// @param branch query string false "branch name, uses default branch if omitted"
// @param environment query string false "environment name"
// @param inputs body string _ "user inputs"
// @success 200 {object} response.BuildReferenceWrapper "Build scheduled"
// @failure 400 {object} problem.Response "Bad request, such as invalid body JSON"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Project was not found"
// @failure 502 {object} problem.Response "Database or code execution engine is unreachable"
// @router /project/{projectId}/{stage}/run [post]
func (m projectModule) startProjectBuildHandler(c *gin.Context) {
	projectID, ok := ginutil.ParseParamUint(c, "projectId")
	if !ok {
		return
	}
	dbProject, ok := fetchProjectByID(c, m.Database, projectID, "when starting a new build")
	if !ok {
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		ginutil.WriteBodyReadError(c, err, fmt.Sprintf(
			"Failed to read the input variables body when starting a new build for project with ID %d.",
			projectID))
		return
	}

	stageName := c.Param("stage")
	env, hasEnv := c.GetQuery("environment")
	branch, hasBranch := c.GetQuery("branch")

	if !hasBranch {
		b, ok := findDefaultBranch(dbProject.Branches)
		if !ok {
			ginutil.WriteDBNotFound(c, fmt.Sprintf(
				"No branch to build for project with ID %d was specified, and no default branch was found on the project.",
				projectID))
			return
		}
		branch = b.Name
	}

	now := time.Now().UTC()
	dbBuild := database.Build{
		ProjectID:   dbProject.ProjectID,
		ScheduledOn: null.TimeFrom(now),
		GitBranch:   branch,
		Environment: null.NewString(env, hasEnv),
		Stage:       stageName,
	}
	if err := m.Database.Create(&dbBuild).Error; err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed creating build on stage %q and branch %q for project with ID %d in database.",
			stageName, branch, projectID))
		return
	}

	dbBuildParams, err := parseDBBuildParams(dbBuild.BuildID, []byte(dbProject.BuildDefinition), body)
	if err != nil {
		dbBuild.IsInvalid = true
		if saveErr := m.Database.Save(&dbBuild).Error; saveErr != nil {
			c.Error(saveErr)
		}
		ginutil.WriteProblemError(c, err, problem.Response{
			Type:   "/prob/api/project/run/params-deserialize",
			Title:  "Parsing build parameters failed.",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf(
				"Failed to deserialize build parameters from request body for build on stage %q and branch %q for project with ID %d.",
				stageName, branch, projectID),
		})
		return
	}

	err = m.SaveBuildParams(dbBuildParams)
	if err != nil {
		dbBuild.IsInvalid = true
		if saveErr := m.Database.Save(&dbBuild).Error; saveErr != nil {
			c.Error(saveErr)
		}
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed saving build parameters for build on stage %q and branch %q for project with ID %d in database.",
			stageName, branch, projectID))
		return
	}

	dbJobParams, err := getDBJobParams(dbProject, dbBuild, dbBuildParams, m.Config.InstanceID)
	if err != nil {
		dbBuild.IsInvalid = true
		if saveErr := m.Database.Save(&dbBuild).Error; saveErr != nil {
			c.Error(saveErr)
		}
		ginutil.WriteProblemError(c, err, problem.Response{
			Type:   "/prob/api/project/run/params-serialize",
			Title:  "Serializing build parameters failed.",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf(
				"Failed to serialize build parameters before sending them onwards to Wharfs execution engine for build on stage %q and branch %q for project with ID %d.",
				stageName, branch, projectID),
		})
		return
	}

	if m.MessageQueue != nil {
		if err := m.MessageQueue.PublishMessage(struct {
			Project    response.Project
			Build      response.Build
			Parameters []response.BuildParam
		}{
			Project:    modelconv.DBProjectToResponse(dbProject),
			Build:      modelconv.DBBuildToResponse(dbBuild),
			Parameters: modelconv.DBBuildParamsToResponses(dbBuildParams),
		}); err != nil {
			log.Error().WithError(err).Message("Failed to publish message.")
			c.Error(err)
			dbBuild.IsInvalid = true
			if saveErr := m.Database.Save(&dbBuild).Error; saveErr != nil {
				ginutil.WriteDBWriteError(c, saveErr, fmt.Sprintf(
					"Failed to marking build with ID %d as invalid after failing to publish event message to message queue.",
					dbBuild.BuildID))
				log.Error().WithError(saveErr).Message("Failed to save build.")
				return
			}
		}
	}

	if m.Config.CI.MockTriggerResponse {
		log.Info().Message("Setting for mocking build triggers was true, mocking CI response.")
		c.JSON(http.StatusOK, modelconv.DBBuildToResponseBuildReferenceWrapper(dbBuild))
		return
	}

	_, err = triggerBuild(dbJobParams, m.Config.CI)
	if err != nil {
		dbBuild.IsInvalid = true
		if saveErr := m.Database.Save(&dbBuild).Error; saveErr != nil {
			c.Error(saveErr)
		}

		ginutil.WriteProblemError(c, err, problem.Response{
			Type:   "/prob/api/project/run/trigger",
			Title:  "Triggering build failed.",
			Status: http.StatusBadGateway,
			Detail: fmt.Sprintf(
				"Failed to trigger code execution engine to schedule the build with ID %d on stage %q on branch %q for project with ID %d.",
				dbBuild.BuildID, stageName, branch, projectID),
		})
		return
	}

	c.JSON(http.StatusOK, modelconv.DBBuildToResponseBuildReferenceWrapper(dbBuild))
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
		Preload(database.ProjectFields.Token)
}

func (m projectModule) SaveBuildParams(dbParams []database.BuildParam) error {
	for _, dbParam := range dbParams {
		if err := m.Database.Create(&dbParam).Error; err != nil {
			return err
		}
	}
	return nil
}

func parseDBBuildParams(buildID uint, buildDef []byte, vars []byte) ([]database.BuildParam, error) {
	type BuildDefinition struct {
		Inputs []struct {
			Name    string
			Type    string
			Default string
		}
	}

	var def BuildDefinition
	err := yaml.Unmarshal(buildDef, &def)
	if err != nil {
		log.Error().WithError(err).Message("Failed unmarshaling build-def.")
		return nil, err
	}

	log.Info().
		WithInt("inputs", len(def.Inputs)).
		Message("Unmarshaled build-def.")

	m := make(map[string]interface{})
	err = json.Unmarshal(vars, &m)
	if err != nil {
		log.Error().WithError(err).Message("Failed unmarshaling input variables JSON.")
		return nil, err
	}

	var params []database.BuildParam
	for _, input := range def.Inputs {
		param := database.BuildParam{
			Name:    input.Name,
			BuildID: buildID,
		}

		if m[input.Name] == nil {
			param.Value = input.Default
		} else {
			param.Value = fmt.Sprintf("%v", m[input.Name])
		}

		params = append(params, param)
	}

	return params, nil
}

func triggerBuild(dbJobParams []database.Param, conf CIConfig) (string, error) {
	q := ""
	for _, dbJobParam := range dbJobParams {
		if dbJobParam.Value != "" {
			q = fmt.Sprintf("%s&%s=%s", q, url.QueryEscape(dbJobParam.Name), url.QueryEscape(dbJobParam.Value))
		}
	}

	tokenStr := fmt.Sprintf("?token=%s", conf.TriggerToken)

	url := fmt.Sprintf("%s%s%s", conf.TriggerURL, tokenStr, q)
	fmt.Printf("POSTing to url: %v\n", url)
	log.Info().
		WithString("method", "POST").
		WithString("url", fmt.Sprintf("%s?token=%s%s", conf.TriggerURL, "*****", q)).
		Message("Triggering build.")

	var resp, err = http.Post(url, "", nil)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	var body, err2 = ioutil.ReadAll(resp.Body)
	if err2 != nil {
		return "", err2
	}

	var strBody = string(body)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return "", fmt.Errorf(strBody)
	}

	return strBody, err2
}

func getDBJobParams(
	dbProject database.Project,
	dbBuild database.Build,
	dbBuildParams []database.BuildParam,
	wharfInstanceID string,
) ([]database.Param, error) {
	var err error
	var v []byte
	if len(dbBuildParams) > 0 {
		m := make(map[string]interface{})

		for _, input := range dbBuildParams {
			m[input.Name] = input.Value
		}

		v, err = yaml.Marshal(m)
		if err != nil {
			log.Error().WithError(err).Message("Failed to marshal input variables YAML for build.")
			return nil, err
		}
	} else {
		log.Debug().Message("Skipping input variables, nothing in body.")
	}

	token := ""
	if dbProject.Token != nil {
		token = dbProject.Token.Token
	}

	dbJobParams := []database.Param{
		{Type: "string", Name: "REPO_NAME", Value: dbProject.Name},
		{Type: "string", Name: "REPO_GROUP", Value: strings.ToLower(dbProject.GroupName)},
		{Type: "string", Name: "REPO_BRANCH", Value: dbBuild.GitBranch},
		{Type: "string", Name: "GIT_BRANCH", Value: dbBuild.GitBranch},
		{Type: "string", Name: "RUN_STAGES", Value: dbBuild.Stage},
		{Type: "string", Name: "BUILD_REF", Value: strconv.FormatUint(uint64(dbBuild.BuildID), 10)},
		{Type: "string", Name: "VARS", Value: string(v)},
		{Type: "string", Name: "GIT_FULLURL", Value: dbProject.GitURL},
		{Type: "string", Name: "GIT_TOKEN", Value: token},
		{Type: "string", Name: "WHARF_PROJECT_ID", Value: strconv.FormatUint(uint64(dbProject.ProjectID), 10)},
		{Type: "string", Name: "WHARF_INSTANCE", Value: wharfInstanceID},
	}

	if dbBuild.Environment.Valid {
		dbJobParams = append(dbJobParams, database.Param{
			Type:  "string",
			Name:  "ENVIRONMENT",
			Value: dbBuild.Environment.String,
		})
	}

	return dbJobParams, nil
}

var defaultGetBuildsOrderBy = orderby.Column{Name: database.BuildColumns.BuildID, Direction: orderby.Desc}

func (m projectModule) getBuilds(projectID uint, limit int, offset int, orderBySlice orderby.Slice) ([]database.Build, error) {
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

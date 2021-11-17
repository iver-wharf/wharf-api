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
	"github.com/iver-wharf/wharf-api/internal/ptrconv"
	"github.com/iver-wharf/wharf-api/internal/wherefields"
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
	Database *gorm.DB
	Config   *Config
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
			projectByID.POST("/build", m.startProjectBuildHandler)
			projectByID.POST("/:stage/run", m.oldStartProjectBuildHandler)
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

// oldStartProjectBuildHandler godoc
// @id oldStartProjectBuild
// @deprecated
// @summary Responsible for run stage environment for selected project
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use `POST /project/{projectId}/build` instead.
// @tags project
// @accept json
// @param projectId path uint true "project ID" minimum(0)
// @param stage path string true "name of stage to run, or specify ALL to run everything"
// @param branch query string false "branch name, uses default branch if omitted"
// @param environment query string false "environment name"
// @param inputs body string _ "user inputs" example(foo:bar)
// @success 200 {object} response.BuildReferenceWrapper "Build scheduled"
// @failure 400 {object} problem.Response "Bad request, such as invalid body JSON"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Project was not found"
// @failure 502 {object} problem.Response "Database or code execution engine is unreachable"
// @router /project/{projectId}/{stage}/run [post]
func (m projectModule) oldStartProjectBuildHandler(c *gin.Context) {
	// not moved to `internal/deprecated` package as it's too much
	// code duplication
	projectID, ok := ginutil.ParseParamUint(c, "projectId")
	if !ok {
		return
	}
	stageName := c.Param("stage")
	m.startBuild(c, projectID, stageName)
}

// startProjectBuildHandler godoc
// @id startProjectBuild
// @summary Start a new build for the given project, with optional build stage, build environment, or repo branch filters.
// @tags project
// @accept json
// @param projectId path uint true "Project ID" minimum(0)
// @param stage query string false "Name of stage to run, or specify `ALL` to run all stages." default(ALL)
// @param branch query string false "Branch name. Uses project's default branch if omitted"
// @param environment query string false "Environment name filter. If left empty it will run all stages without any environment filters."
// @param inputs body request.BuildInputs _ "Input variable values. Map of variable names (as defined in the project's `.wharf-ci.yml` file) as keys paired with their string, boolean, or numeric value."
// @success 200 {object} response.BuildReferenceWrapper "Build scheduled"
// @failure 400 {object} problem.Response "Bad request, such as invalid body JSON"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Project was not found"
// @failure 502 {object} problem.Response "Database or code execution engine is unreachable"
// @router /project/{projectId}/build [post]
func (m projectModule) startProjectBuildHandler(c *gin.Context) {
	projectID, ok := ginutil.ParseParamUint(c, "projectId")
	if !ok {
		return
	}
	stageName, hasStageName := c.GetQuery("stage")
	if !hasStageName {
		stageName = "ALL"
	}
	m.startBuild(c, projectID, stageName)
}

func (m projectModule) startBuild(c *gin.Context, projectID uint, stageName string) {
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

	m := make(request.BuildInputs)
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

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"encoding/json"
	"net/http"
	"net/url"

	"github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/messagebus-go"
	"github.com/iver-wharf/wharf-api/pkg/httputils"
	"github.com/iver-wharf/wharf-api/pkg/orderby"
	"github.com/iver-wharf/wharf-api/pkg/problem"
	"gopkg.in/guregu/null.v4"
	"gorm.io/gorm"
)

type ProjectModule struct {
	Database     *gorm.DB
	MessageQueue *messagebus.MQConnection
}

type PaginatedBuilds struct {
	Builds     *[]Build `json:"builds"`
	TotalCount int64    `json:"totalCount"`
}

type BuildReferenceWrapper struct {
	BuildReference string `json:"buildRef"`
}

var wharfInstance = os.Getenv("WHARF_INSTANCE")
var triggerUrl = os.Getenv("CI_URL")
var token = os.Getenv("CI_TOKEN")

func (m ProjectModule) Register(g *gin.RouterGroup) {
	projects := g.Group("/projects")
	{
		projects.GET("", m.getProjectsHandler)
		projects.POST("/search", m.searchProjectsHandler)
		projects.GET("/:projectid/builds", m.getBuildsSliceHandler)
	}

	project := g.Group("/project")
	{
		project.GET("/:projectid", m.getProjectHandler)
		project.POST("", m.postProjectHandler)
		project.DELETE("/:projectid", m.deleteProjectHandler)
		project.PUT("", m.putProjectHandler)
		project.POST("/:projectid/:stage/run", m.runStageHandler)
	}
}

func (m ProjectModule) FindProjectByID(id uint) (Project, error) {
	var project Project
	err := m.Database.Set("gorm:auto_preload", false).
		Where(&Project{ProjectID: id}).
		Preload(projectAssocProvider).
		Preload(projectAssocBranches, func(db *gorm.DB) *gorm.DB {
			return db.Order(buildColumnName)
		}).
		Preload(projectAssocToken).
		First(&project).
		Error

	return project, err
}

// getProjectsHandler godoc
// @summary Returns all projects from database
// @tags project
// @success 200 {array} Project
// @failure 502 {object} problem.Response "Database is unreachable"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @router /projects [get]
func (m ProjectModule) getProjectsHandler(c *gin.Context) {
	var projects []Project
	err := m.Database.
		Preload(projectAssocProvider).
		Preload(projectAssocBranches, func(db *gorm.DB) *gorm.DB {
			return db.Order(buildColumnName)
		}).
		Find(&projects).
		Error
	if err != nil {
		problem.WriteDBReadError(c, err, "Failed fetching list of projects from database.")
		return
	}
	c.JSON(http.StatusOK, projects)
}

// searchProjectsHandler godoc
// @summary Search for projects from database
// @tags project
// @success 200 {array} Project
// @failure 502 {object} problem.Response "Database is unreachable"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @router /projects/search [post]
func (m ProjectModule) searchProjectsHandler(c *gin.Context) {
	var query Project
	if err := c.ShouldBindJSON(&query); err != nil {
		problem.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for the project object to search with.")
		return
	}
	var projects []Project
	err := m.Database.
		Where(&query).
		Preload(projectAssocProvider).
		Find(&projects).
		Error
	if err != nil {
		problem.WriteDBReadError(c, err, "Failed searching for projects in database.")
		return
	}
	c.JSON(http.StatusOK, projects)
}

// getBuildsSliceHandler godoc
// @summary Get slice of builds.
// @tags project
// @param projectid path int true "project ID"
// @param limit query string true "number of fetched branches"
// @param offset query string true "PK of last branch taken"
// @param orderby query []string false "Sorting orders. Takes the property name followed by either 'asc' or 'desc'. Can be specified multiple times for more granular sorting. Defaults to '?orderby=buildId desc'"
// @success 200 {object} PaginatedBuilds
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /projects/{projectid}/builds [get]
func (m ProjectModule) getBuildsSliceHandler(c *gin.Context) {
	projectID, ok := httputils.ParseParamUint(c, "projectid")
	if !ok {
		return
	}
	limit, ok := httputils.ParseQueryInt(c, "limit")
	if !ok {
		return
	}
	offset, ok := httputils.ParseQueryInt(c, "offset")
	if !ok {
		return
	}
	orderByQueryParams := c.QueryArray("orderby")
	orderBySlice, err := orderby.ParseSlice(orderByQueryParams, buildJsonToColumns)
	if err != nil {
		joinedOrders := strings.Join(orderByQueryParams, ", ")
		problem.WriteInvalidParamError(c, err, "orderby", fmt.Sprintf(
			"Failed parsing the %d sort ordering values: %s",
			len(orderByQueryParams),
			joinedOrders))
		return
	}

	builds, err := m.getBuilds(projectID, limit, offset, orderBySlice)
	if err != nil {
		problem.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching builds for project with ID %d from database.",
			projectID))
		return
	}

	count, err := m.getBuildsCount(projectID)
	if err != nil {
		problem.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching build count for project with ID %d from database.",
			projectID))
		return
	}

	c.JSON(http.StatusOK, PaginatedBuilds{Builds: &builds, TotalCount: count})
}

// getProjectHandler godoc
// @summary Returns project with selected project ID
// @tags project
// @param projectid path int true "project ID"
// @success 200 {object} Project
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Project not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /project/{projectid} [get]
func (m ProjectModule) getProjectHandler(c *gin.Context) {
	projectID, ok := httputils.ParseParamUint(c, "projectid")
	if !ok {
		return
	}
	project, err := m.FindProjectByID(projectID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		problem.WriteDBNotFound(c, fmt.Sprintf("Project with ID %d was not found.", projectID))
		return
	} else if err != nil {
		problem.WriteDBReadError(c, err, fmt.Sprintf("Failed fetching project with ID %d from database.", projectID))
		return
	}
	c.JSON(http.StatusOK, &project)
}

// postProjectHandler godoc
// @summary Updates project.
// @description It finds project by ID or if ID is set to 0 it takes group id, token id and name.
// @description First found project will have updated avatar, description and build definition
// @tags project
// @accept json
// @produce json
// @param project body Project true "project object"
// @success 200 {object} Project "Project has been updated"
// @success 201 {object} Project "Project has been created"
// @failure 400 {object} problem.Response "Bad request"
// @failure 404 {object} problem.Response "Project to update is not found"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /project [post]
func (m ProjectModule) postProjectHandler(c *gin.Context) {
	var project Project

	if err := c.ShouldBindJSON(&project); err != nil {
		problem.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for the project object to update.")
		return
	}

	var existingProject Project
	if project.ProjectID != 0 {
		err := m.Database.
			Where(&project, projectFieldProjectID).
			First(&existingProject).
			Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			problem.WriteDBNotFound(c, fmt.Sprintf(
				"Project with ID %d was not found. Please post a project model with 'projectId' left unset or set to 0 if you wish to create a new project.",
				project.ProjectID))
			return
		} else if err != nil {
			problem.WriteDBReadError(c, err, fmt.Sprintf(
				"Failed fetching project with ID %d from database.",
				project.ProjectID))
			return
		}
	} else {
		err := m.Database.
			Where(&project, projectFieldGroupName, projectFieldTokenID, projectFieldName).
			First(&existingProject).
			Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := m.Database.Create(&project).Error; err != nil {
				problem.WriteDBWriteError(c, err, fmt.Sprintf(
					"Failed creating new project with group %q, token ID %d, and name %q in database.",
					project.GroupName, project.TokenID, project.Name))
			} else {
				c.JSON(http.StatusCreated, project)
			}
			return
		} else if err != nil {
			problem.WriteDBReadError(c, err, fmt.Sprintf(
				"Failed to lookup project with group %q, token ID %d, and name %q from database.",
				project.GroupName, project.TokenID, project.Name))
			return
		}
	}

	existingProject.BuildDefinition = project.BuildDefinition
	existingProject.Description = project.Description
	existingProject.AvatarUrl = project.AvatarUrl

	m.Database.Save(existingProject)

	c.JSON(http.StatusOK, existingProject)
}

// deleteProjectHandler godoc
// @summary Delete project with selected project ID
// @tags project
// @param projectid path int true "project ID"
// @success 204 "Deleted"
// @failure 502 {object} problem.Response "Database is unreachable"
// @failure 400 {object} problem.Response "Bad request"
// @failure 404 {object} problem.Response "Project to delete is not found"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @router /project/{projectid} [delete]
func (m ProjectModule) deleteProjectHandler(c *gin.Context) {
	projectID, ok := httputils.ParseParamUint(c, "projectid")
	if !ok {
		return
	}

	project, err := m.FindProjectByID(projectID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		problem.WriteDBNotFound(c, fmt.Sprintf("Project with ID %d was not found in the database.", projectID))
	} else if err != nil {
		problem.WriteDBReadError(c, err, fmt.Sprintf("Failed fetching project with ID %d from database.", projectID))
		return
	}

	err = m.Database.Delete(&project).Error
	if err != nil {
		problem.WriteDBWriteError(c, err, fmt.Sprintf("Failed deleting project with ID %d from database.", projectID))
		return
	}

	c.Status(http.StatusNoContent)
}

// putProjectHandler godoc
// @summary Adds project when not exists.
// @description It finds project by ID or if ID is set to 0 it takes group name.
// @description First found project will be returned. If not found project will be added into database.
// @description It ignores branches array, build history and provider params.
// @tags project
// @accept json
// @produce json
// @param project body Project _ "project object"
// @success 200 {object} Project
// @failure 400 {object} problem.Response "Bad request, such as invalid body JSON or when altering group"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Project to update was not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /project [put]
func (m ProjectModule) putProjectHandler(c *gin.Context) {
	var project Project
	err := c.ShouldBindJSON(&project)
	if err != nil {
		problem.WriteInvalidBindError(c, err, "One or more parameters failed to parse when reading the request body.")
		return
	}

	var existingProject Project
	if project.ProjectID != 0 {
		existingProject, err = m.FindProjectByID(project.ProjectID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			problem.WriteDBNotFound(c, fmt.Sprintf(
				"Project with ID %d was not found in the database.",
				project.ProjectID))
			return
		} else if err != nil {
			problem.WriteDBReadError(c, err, fmt.Sprintf(
				"Failed fetching project by ID %d from database.",
				project.ProjectID))
			return
		}
	} else {
		err := m.Database.
			Where(&project, projectFieldName, projectFieldGroupName).
			First(&existingProject).
			Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			existingProject = project
		} else if err != nil {
			problem.WriteDBReadError(c, err, fmt.Sprintf(
				"Failed fetching project with name %q and group name %q from the database.",
				project.Name, project.GroupName))
			return
		}
	}

	if existingProject.GroupName != project.GroupName {
		problem.WriteProblem(c, problem.Response{
			Type:  "/prob/api/project/cannot-change-group",
			Title: "Project group cannot be changed.",
			Detail: fmt.Sprintf(
				"Changing the group of a project is prohibited. The client tried to change group on project %d from %q to %q.",
				project.ProjectID, existingProject.GroupName, project.GroupName),
			Status: http.StatusBadRequest,
		})
		return
	}

	project.ProjectID = existingProject.ProjectID

	if err := m.Database.Save(&project).Error; err != nil {
		problem.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed writing project with name %q and group name %q to database.",
			project.Name, project.GroupName))
		return
	}

	c.JSON(http.StatusOK, project)
}

// runStageHandler godoc
// @summary Responsible for run stage environment for selected project
// @tags project
// @accept json
// @param projectid path int true "project ID"
// @param stage path string true "name of stage to run, or specify ALL to run everything"
// @param branch query string false "branch name, uses default branch if omitted"
// @param environment query string false "environment name"
// @param inputs body string _ "user inputs"
// @success 200 {object} BuildReferenceWrapper "Build scheduled"
// @failure 400 {object} problem.Response "Bad request, such as invalid body JSON"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Project was not found"
// @failure 502 {object} problem.Response "Database or code execution engine is unreachable"
// @router /project/{projectid}/{stage}/run [post]
func (m ProjectModule) runStageHandler(c *gin.Context) {
	projectID, ok := httputils.ParseParamUint(c, "projectid")
	if !ok {
		return
	}

	project, err := m.FindProjectByID(projectID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		problem.WriteDBNotFound(c, fmt.Sprintf("Project with ID %d was not found in the database.", projectID))
		return
	} else if err != nil {
		problem.WriteDBReadError(c, err, fmt.Sprintf("Failed fetching project with ID %d from database.", projectID))
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		problem.WriteBodyReadError(c, err, fmt.Sprintf(
			"Failed to read the input variables body when starting a new build for project with ID %d.",
			projectID))
		return
	}

	stageName := c.Param("stage")
	env, hasEnv := c.GetQuery("environment")
	branch, hasBranch := c.GetQuery("branch")

	if !hasBranch {
		if b, ok := findDefaultBranch(project.Branches); !ok {
			problem.WriteDBNotFound(c, fmt.Sprintf(
				"No branch to build for project with ID %d was specified, and no default branch was found on the project.",
				projectID))
			return
		} else {
			branch = b.Name
		}
	}

	now := time.Now().UTC()
	build := Build{
		ProjectID:   project.ProjectID,
		ScheduledOn: &now,
		GitBranch:   branch,
		Environment: null.NewString(env, hasEnv),
		Stage:       stageName,
	}
	if err := m.Database.Create(&build).Error; err != nil {
		problem.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed creating build on stage %q and branch %q for project with ID %d in database.",
			stageName, branch, projectID))
		return
	}

	buildParams, err := ParseBuildParams(build.BuildID, []byte(project.BuildDefinition), body)
	if err != nil {
		build.IsInvalid = true
		if saveErr := m.Database.Save(&build).Error; saveErr != nil {
			c.Error(saveErr)
		}
		problem.WriteProblemError(c, err, problem.Response{
			Type:   "/prob/api/project/run/params-deserialize",
			Title:  "Parsing build parameters failed.",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf(
				"Failed to deserialize build parameters from request body for build on stage %q and branch %q for project with ID %d.",
				stageName, branch, projectID),
		})
		return
	}

	err = m.SaveBuildParams(buildParams)
	if err != nil {
		build.IsInvalid = true
		if saveErr := m.Database.Save(&build).Error; saveErr != nil {
			c.Error(saveErr)
		}
		problem.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed saving build parameters for build on stage %q and branch %q for project with ID %d in database.",
			stageName, branch, projectID))
		return
	}

	jobParams, err := getParams(project, build, buildParams)
	if err != nil {
		build.IsInvalid = true
		if saveErr := m.Database.Save(&build).Error; saveErr != nil {
			c.Error(saveErr)
		}
		problem.WriteProblemError(c, err, problem.Response{
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
			Project    Project
			Build      Build
			Parameters []BuildParam
		}{
			Project:    project,
			Build:      build,
			Parameters: buildParams,
		}); err != nil {
			fmt.Printf("Error on message publish: %+v", err)
			c.Error(err)
			build.IsInvalid = true
			if saveErr := m.Database.Save(&build).Error; saveErr != nil {
				problem.WriteDBWriteError(c, saveErr, fmt.Sprintf(
					"Failed to marking build with ID %d as invalid after failing to publish event message to message queue.",
					build.BuildID))
				fmt.Printf("Error on build save: %+v", saveErr)
				return
			}
		}
	}

	if os.Getenv("MOCK_LOCAL_CI_RESPONSE") != "" {
		fmt.Println("MOCK_LOCAL_CI_RESPONSE env var set to true, mocking CI response.")
		c.JSON(http.StatusOK, build.BuildReferenceWrapper())
		return
	}

	_, err = triggerBuild(jobParams)
	if err != nil {
		build.IsInvalid = true
		if saveErr := m.Database.Save(&build).Error; saveErr != nil {
			c.Error(saveErr)
		}

		problem.WriteProblemError(c, err, problem.Response{
			Type:   "/prob/api/project/run/trigger",
			Title:  "Triggering build failed.",
			Status: http.StatusBadGateway,
			Detail: fmt.Sprintf(
				"Failed to trigger code execution engine to schedule the build with ID %d on stage %q on branch %q for project with ID %d.",
				build.BuildID, stageName, branch, projectID),
		})
		return
	}

	c.JSON(http.StatusOK, build.BuildReferenceWrapper())
}

func (b Build) BuildReferenceWrapper() BuildReferenceWrapper {
	return BuildReferenceWrapper{BuildReference: strconv.FormatUint(uint64(b.BuildID), 10)}
}

func (m ProjectModule) FindBranches(projectID uint) ([]Branch, error) {
	var branches []Branch
	m.Database.Where(&Branch{ProjectID: projectID}).Find(&branches)
	return branches, nil
}

func (m ProjectModule) FindProvider(providerID uint) (Provider, error) {
	var provider Provider
	m.Database.Where(&Provider{ProviderID: providerID}).Find(&provider)
	return provider, nil
}

func (m ProjectModule) SaveBuildParams(params []BuildParam) error {
	for _, p := range params {
		if err := m.Database.Create(&p).Error; err != nil {
			return err
		}
	}
	return nil
}

func ParseBuildParams(buildID uint, buildDef []byte, vars []byte) ([]BuildParam, error) {
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
		fmt.Printf("Error: %v\n", err)
		return []BuildParam{}, err
	}

	fmt.Printf("Unmarshaled build-def, found %v inputs\n", len(def.Inputs))

	m := make(map[string]interface{})
	err = json.Unmarshal(vars, &m)
	if err != nil {
		fmt.Printf("Error unmarshaling json: %+v\n", err)
		return []BuildParam{}, err
	}

	params := []BuildParam{}
	for _, input := range def.Inputs {
		param := BuildParam{
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

func triggerBuild(params []Param) (string, error) {
	q := ""
	for _, param := range params {
		if param.Value != "" {
			q = fmt.Sprintf("%s&%s=%s", q, url.QueryEscape(param.Name), url.QueryEscape(param.Value))
		}
	}

	tokenStr := fmt.Sprintf("?token=%s", token)

	url := fmt.Sprintf("%s%s%s", triggerUrl, tokenStr, q)
	fmt.Printf("POSTing to url: %v\n", url)

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

func getParams(project Project, build Build, vars []BuildParam) ([]Param, error) {
	var err error
	var v []byte
	if len(vars) > 0 {
		m := make(map[string]interface{})

		for _, input := range vars {
			m[input.Name] = input.Value
		}

		v, err = yaml.Marshal(m)
		if err != nil {
			fmt.Printf("Error marshaling yaml: %+v\n", err)
			return []Param{}, err
		}
	} else {
		fmt.Println("Skipping input variables, nothing in body")
	}

	token := ""
	if project.Token != nil {
		token = project.Token.Token
	}

	params := []Param{
		{Type: "string", Name: "REPO_NAME", Value: project.Name},
		{Type: "string", Name: "REPO_GROUP", Value: strings.ToLower(project.GroupName)},
		{Type: "string", Name: "REPO_BRANCH", Value: build.GitBranch},
		{Type: "string", Name: "GIT_BRANCH", Value: build.GitBranch},
		{Type: "string", Name: "RUN_STAGES", Value: build.Stage},
		{Type: "string", Name: "BUILD_REF", Value: strconv.FormatUint(uint64(build.BuildID), 10)},
		{Type: "string", Name: "VARS", Value: string(v)},
		{Type: "string", Name: "GIT_FULLURL", Value: project.GitURL},
		{Type: "string", Name: "GIT_TOKEN", Value: token},
		{Type: "string", Name: "WHARF_PROJECT_ID", Value: strconv.FormatUint(uint64(project.ProjectID), 10)},
		{Type: "string", Name: "WHARF_INSTANCE", Value: wharfInstance},
	}

	if build.Environment.Valid {
		params = append(params, Param{Type: "string", Name: "ENVIRONMENT", Value: build.Environment.String})
	}

	return params, nil
}

var defaultGetBuildsOrderBy = orderby.OrderBy{Column: buildColumnBuildID, Direction: orderby.Desc}

func (m ProjectModule) getBuilds(projectID uint, limit int, offset int, orderBySlice []orderby.OrderBy) ([]Build, error) {
	var builds []Build
	var query = m.Database.
		Where(&Build{ProjectID: projectID}).
		Limit(limit).
		Offset(offset)
	query = orderby.ApplyAllToGormQuery(query, orderBySlice, defaultGetBuildsOrderBy)
	if err := query.Find(&builds).Error; err != nil {
		return []Build{}, err
	}
	return builds, nil
}

func (m ProjectModule) getBuildsCount(projectID uint) (int64, error) {
	var count int64
	if err := m.Database.
		Model(&Build{}).
		Where(&Build{ProjectID: projectID}).
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return count, nil
}

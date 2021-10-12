package modelconv

import (
	"github.com/ghodss/yaml"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
)

// DBProjectsToResponses converts a slice of database projects to a slice of
// response projects.
func DBProjectsToResponses(dbProjects []database.Project) []response.Project {
	resProjects := make([]response.Project, len(dbProjects))
	for i, dbProject := range dbProjects {
		resProjects[i] = DBProjectToResponse(dbProject)
	}
	return resProjects
}

// DBProjectToResponse converts a database project to a response project.
func DBProjectToResponse(dbProject database.Project) response.Project {
	var resProviderPtr *response.Provider
	if dbProject.Provider != nil {
		resProvider := DBProviderToResponse(*dbProject.Provider)
		resProviderPtr = &resProvider
	}
	parsedBuildDef, err := ParseBuildDefinition(dbProject.BuildDefinition)
	if err != nil {
		log.Warn().
			WithError(err).
			WithUint("project", dbProject.ProjectID).
			Message("Failed to parse build-definition.")
	}
	return response.Project{
		ProjectID:             dbProject.ProjectID,
		Name:                  dbProject.Name,
		GroupName:             dbProject.GroupName,
		Description:           dbProject.Description,
		AvatarURL:             dbProject.AvatarURL,
		TokenID:               dbProject.TokenID,
		ProviderID:            dbProject.ProviderID,
		Provider:              resProviderPtr,
		BuildDefinition:       dbProject.BuildDefinition,
		Branches:              DBBranchesToResponses(dbProject.Branches),
		GitURL:                dbProject.GitURL,
		ParsedBuildDefinition: parsedBuildDef,
	}
}

// ParseBuildDefinition parses a YAML-formatted build definition string to be
// used in a project response.
func ParseBuildDefinition(buildDef string) (interface{}, error) {
	if buildDef == "" {
		return nil, nil
	}
	var parsed interface{}
	err := yaml.Unmarshal([]byte(buildDef), &parsed)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}
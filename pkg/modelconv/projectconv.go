package modelconv

import (
	"github.com/ghodss/yaml"
	"github.com/iver-wharf/wharf-api/v5/internal/ptrconv"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/request"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/response"
	"gopkg.in/typ.v4"
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
		TimeMetadata:          DBTimeMetadataToResponse(dbProject.TimeMetadata),
		ProjectID:             dbProject.ProjectID,
		Name:                  dbProject.Name,
		GroupName:             dbProject.GroupName,
		Description:           typ.Coal(dbProject.Overrides.Description, dbProject.Description),
		AvatarURL:             typ.Coal(dbProject.Overrides.AvatarURL, dbProject.AvatarURL),
		TokenID:               ptrconv.UintPtr(dbProject.TokenID),
		ProviderID:            ptrconv.UintPtr(dbProject.ProviderID),
		Provider:              resProviderPtr,
		BuildDefinition:       dbProject.BuildDefinition,
		Branches:              DBBranchesToResponses(dbProject.Branches),
		GitURL:                typ.Coal(dbProject.Overrides.GitURL, dbProject.GitURL),
		RemoteProjectID:       dbProject.RemoteProjectID,
		ParsedBuildDefinition: parsedBuildDef,
	}
}

// ReqProjectToDatabase converts a request project to a database project.
func ReqProjectToDatabase(reqProject request.Project) database.Project {
	return database.Project{
		Name:            reqProject.Name,
		GroupName:       reqProject.GroupName,
		Description:     reqProject.Description,
		AvatarURL:       reqProject.AvatarURL,
		TokenID:         ptrconv.UintZeroNil(reqProject.TokenID),
		ProviderID:      ptrconv.UintZeroNil(reqProject.ProviderID),
		BuildDefinition: reqProject.BuildDefinition,
		GitURL:          reqProject.GitURL,
		RemoteProjectID: reqProject.RemoteProjectID,
	}
}

// ParseBuildDefinition parses a YAML-formatted build definition string to be
// used in a project response.
func ParseBuildDefinition(buildDef string) (any, error) {
	if buildDef == "" {
		return nil, nil
	}
	var parsed any
	err := yaml.Unmarshal([]byte(buildDef), &parsed)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

// DBProjectOverridesToResponse converts a database project's overrides to a
// response project's overrides.
func DBProjectOverridesToResponse(dbProjectOverrides database.ProjectOverrides) response.ProjectOverrides {
	return response.ProjectOverrides{
		ProjectID:   dbProjectOverrides.ProjectID,
		Description: dbProjectOverrides.Description,
		AvatarURL:   dbProjectOverrides.AvatarURL,
		GitURL:      dbProjectOverrides.GitURL,
	}
}

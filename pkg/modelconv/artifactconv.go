package modelconv

import (
	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/response"
)

// DBArtifactToResponse converts a database artifact to a response artifact.
func DBArtifactToResponse(dbArtifact database.Artifact) response.Artifact {
	return response.Artifact{
		TimeMetadata: DBTimeMetadataToResponse(dbArtifact.TimeMetadata),
		ArtifactID:   dbArtifact.ArtifactID,
		BuildID:      dbArtifact.BuildID,
		Name:         dbArtifact.Name,
		FileName:     dbArtifact.FileName,
	}
}

// DBArtifactsToResponses converts a slice of database artifacts to a slice of
// response artifacts.
func DBArtifactsToResponses(dbArtifacts []database.Artifact) []response.Artifact {
	resArtifacts := make([]response.Artifact, len(dbArtifacts))
	for i, dbArtifact := range dbArtifacts {
		resArtifacts[i] = DBArtifactToResponse(dbArtifact)
	}
	return resArtifacts
}

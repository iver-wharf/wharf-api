package modelconv

import (
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
)

// DBArtifactToResponse converts a database artifact to a response artifact.
func DBArtifactToResponse(dbArtifact database.Artifact) response.Artifact {
	return response.Artifact{
		ArtifactID: dbArtifact.ArtifactID,
		BuildID:    dbArtifact.BuildID,
		Name:       dbArtifact.Name,
		FileName:   dbArtifact.FileName,
	}
}

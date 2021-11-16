package modelconv

import (
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
)

// DBTimeMetadataToResponse converts a database timestamp metadata to a response
// timestamp metadata.
func DBTimeMetadataToResponse(timeMetadata database.TimeMetadata) response.TimeMetadata {
	return response.TimeMetadata{
		CreatedAt: timeMetadata.CreatedAt,
		UpdatedAt: timeMetadata.UpdatedAt,
	}
}

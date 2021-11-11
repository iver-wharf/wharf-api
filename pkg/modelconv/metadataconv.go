package modelconv

import (
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
)

func DBTimeMetadataToResponse(timeMetadata database.TimeMetadata) response.TimeMetadata {
	return response.TimeMetadata{
		CreatedAt: timeMetadata.CreatedAt,
		UpdatedAt: timeMetadata.UpdatedAt,
	}
}

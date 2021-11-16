package modelconv

import (
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
)

// DBProvidersToResponses converts a slice of database providers to a slice of
// response providers.
func DBProvidersToResponses(dbProviders []database.Provider) []response.Provider {
	resProviders := make([]response.Provider, len(dbProviders))
	for i, dbProvider := range dbProviders {
		resProviders[i] = DBProviderToResponse(dbProvider)
	}
	return resProviders
}

// DBProviderToResponse converts a database provider to a response provider.
func DBProviderToResponse(dbProvider database.Provider) response.Provider {
	return response.Provider{
		TimeMetadata: DBTimeMetadataToResponse(dbProvider.TimeMetadata),
		ProviderID:   dbProvider.ProviderID,
		Name:         response.ProviderName(dbProvider.Name),
		URL:          dbProvider.URL,
		TokenID:      dbProvider.TokenID,
	}
}

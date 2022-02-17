package modelconv

import (
	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/response"
)

// DBTokensToResponses converts a slice of database tokens to a slice of
// response tokens.
func DBTokensToResponses(dbTokens []database.Token) []response.Token {
	resTokens := make([]response.Token, len(dbTokens))
	for i, dbToken := range dbTokens {
		resTokens[i] = DBTokenToResponse(dbToken)
	}
	return resTokens
}

// DBTokenToResponse converts a database token to a response token.
func DBTokenToResponse(dbToken database.Token) response.Token {
	return response.Token{
		TimeMetadata: DBTimeMetadataToResponse(dbToken.TimeMetadata),
		TokenID:      dbToken.TokenID,
		Token:        dbToken.Value,
		UserName:     dbToken.UserName,
	}
}

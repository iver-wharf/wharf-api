package modelconv

import (
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
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
		TokenID:  dbToken.TokenID,
		Token:    dbToken.Token,
		UserName: dbToken.UserName,
	}
}

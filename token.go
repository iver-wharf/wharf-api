package main

import (
	"fmt"

	"github.com/iver-wharf/wharf-api/internal/wherefields"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/request"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
	"github.com/iver-wharf/wharf-api/pkg/modelconv"
	"github.com/iver-wharf/wharf-api/pkg/orderby"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"

	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type tokenModule struct {
	Database *gorm.DB
}

func (m tokenModule) Register(g *gin.RouterGroup) {
	token := g.Group("/token")
	{
		token.GET("", m.getTokenListHandler)
		token.POST("", m.createTokenHandler)

		tokenByID := token.Group("/:tokenId")
		{
			tokenByID.GET("", m.getTokenHandler)
			tokenByID.PUT("", m.updateTokenHandler)
		}
	}
}

var tokenJSONToColumns = map[string]database.SafeSQLName{
	response.TokenJSONFields.TokenID:  database.TokenColumns.TokenID,
	response.TokenJSONFields.Token:    database.TokenColumns.Token,
	response.TokenJSONFields.UserName: database.TokenColumns.UserName,
}

var defaultGetTokensOrderBy = orderby.Column{Name: database.TokenColumns.TokenID, Direction: orderby.Desc}

// getBuildListHandler godoc
// @id getTokenList
// @summary Get slice of tokens.
// @description List all tokens, or a window of tokens using the `limit` and `offset` query parameters. Allows optional filtering parameters.
// @description Verbatim filters will match on the entire string used to find exact matches,
// @description while the matching filters are meant for searches by humans where it tries to find soft matches and is therefore inaccurate by nature.
// @tags token
// @param limit query int false "Number of results to return. No limiting is applied if empty (`?limit=`) or non-positive (`?limit=0`). Required if `offset` is used." default(100)
// @param offset query int false "Skipped results, where 0 means from the start." minimum(0) default(0)
// @param orderby query []string false "Sorting orders. Takes the property name followed by either 'asc' or 'desc'. Can be specified multiple times for more granular sorting. Defaults to `?orderby=tokenId desc`"
// @param userName query string false "Filter by verbatim token user name."
// @param userNameMatch query string false "Filter by matching token user name. Cannot be used with `userName`."
// @success 200 {object} response.PaginatedTokens
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /token [get]
func (m tokenModule) getTokenListHandler(c *gin.Context) {
	var params = struct {
		commonGetQueryParams

		UserName      *string `form:"userName"`
		UserNameMatch *string `form:"userNameMatch" binding:"excluded_with=UserNameMatch"`
	}{
		commonGetQueryParams: defaultCommonGetQueryParams,
	}
	if !bindCommonGetQueryParams(c, &params) {
		return
	}
	orderBySlice, ok := parseCommonOrderBySlice(c, params.OrderBy, tokenJSONToColumns)
	if !ok {
		return
	}

	var where wherefields.Collection
	query := m.Database.
		Clauses(orderBySlice.ClauseIfNone(defaultGetTokensOrderBy)).
		Where(&database.Token{
			UserName: where.String(database.TokenFields.UserName, params.UserName),
		}, where.NonNilFieldNames()...).
		Scopes(
			whereLikeScope(map[database.SafeSQLName]*string{
				database.TokenColumns.UserName: params.UserNameMatch,
			}),
		)

	var dbTokens []database.Token
	var totalCount int64
	err := findDBPaginatedSliceAndTotalCount(query, params.Limit, params.Offset, &dbTokens, &totalCount)
	if err != nil {
		ginutil.WriteDBReadError(c, err, "Failed fetching list of tokens from database.")
		return
	}

	c.JSON(http.StatusOK, response.PaginatedTokens{
		Tokens:     modelconv.DBTokensToResponses(dbTokens),
		TotalCount: totalCount,
	})
}

// getTokenHandler godoc
// @id getToken
// @summary Returns token with selected token ID
// @tags token
// @param tokenId path uint true "Token ID" minimum(0)
// @success 200 {object} response.Token
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /token/{tokenId} [get]
func (m tokenModule) getTokenHandler(c *gin.Context) {
	tokenID, ok := ginutil.ParseParamUint(c, "tokenId")
	if !ok {
		return
	}

	dbToken, ok := fetchTokenByID(c, m.Database, tokenID, "")
	if !ok {
		return
	}

	resToken := modelconv.DBTokenToResponse(dbToken)
	c.JSON(http.StatusOK, resToken)
}

// createTokenHandler godoc
// @id createToken
// @summary Add token to database.
// @description Add token to database. Provider in post object has to exists or should be empty.
// @tags token
// @accept json
// @produce json
// @param token body request.Token _ "Token to create"
// @success 201 {object} response.Token
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Referenced provider not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /token [post]
func (m tokenModule) createTokenHandler(c *gin.Context) {
	var reqToken request.Token
	if err := c.ShouldBindJSON(&reqToken); err != nil {
		ginutil.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for the token object to create.")
		return
	}

	dbToken := database.Token{
		Token:    reqToken.Token,
		UserName: reqToken.UserName,
	}

	if reqToken.ProviderID != 0 {
		dbProvider, ok := fetchProviderByID(c, m.Database, reqToken.ProviderID, "when creating a new token")
		if !ok {
			return
		}
		dbProvider.Token = &dbToken
		if err := m.Database.Save(&dbProvider).Error; err != nil {
			ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
				"Failed updating provider with ID %d with a new token.",
				reqToken.ProviderID))
			return
		}
	} else {
		if err := m.Database.Create(&dbToken).Error; err != nil {
			ginutil.WriteDBWriteError(c, err, "Failed creating a new token.")
			return
		}
	}

	resToken := modelconv.DBTokenToResponse(dbToken)
	c.JSON(http.StatusCreated, resToken)
}

// updateTokenHandler godoc
// @id updateToken
// @summary Update token in database.
// @description Updates a token by replacing all of its fields.
// @tags token
// @accept json
// @produce json
// @param tokenId path uint true "ID of token to update" minimum(0)
// @param token body request.TokenUpdate _ "New token values"
// @success 200 {object} response.Token
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Token not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /token/{tokenId} [put]
func (m tokenModule) updateTokenHandler(c *gin.Context) {
	tokenID, ok := ginutil.ParseParamUint(c, "tokenId")
	if !ok {
		return
	}
	var reqToken request.TokenUpdate
	if err := c.ShouldBindJSON(&reqToken); err != nil {
		ginutil.WriteInvalidBindError(c, err, "One or more parameters failed to parse when reading the request body.")
		return
	}
	dbToken, ok := fetchTokenByID(c, m.Database, tokenID, "when updating token")
	if !ok {
		return
	}

	dbToken.Token = reqToken.Token
	dbToken.UserName = reqToken.UserName

	if err := m.Database.Save(&dbToken).Error; err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed to update token by ID %d.",
			tokenID))
	}

	resToken := modelconv.DBTokenToResponse(dbToken)
	c.JSON(http.StatusOK, resToken)
}

func fetchTokenByID(c *gin.Context, db *gorm.DB, tokenID uint, whenMsg string) (database.Token, bool) {
	var dbToken database.Token
	ok := fetchDatabaseObjByID(c, db, &dbToken, tokenID, "token", whenMsg)
	return dbToken, ok
}

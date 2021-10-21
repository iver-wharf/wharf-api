package main

import (
	"fmt"

	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/request"
	"github.com/iver-wharf/wharf-api/pkg/modelconv"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"

	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type tokenModule struct {
	Database *gorm.DB
}

func (m tokenModule) Register(g *gin.RouterGroup) {
	tokens := g.Group("/tokens")
	{
		tokens.GET("", m.getTokenListHandler)
		tokens.POST("/search", m.searchTokenListHandler)
	}

	token := g.Group("/token")
	{
		token.GET("/:tokenId", m.getTokenHandler)
		token.POST("", m.createTokenHandler)
		token.PUT("/:tokenId", m.updateTokenHandler)
	}
}

// getTokenListHandler godoc
// @id getTokenList
// @summary Returns first 100 tokens
// @tags token
// @success 200 {object} []response.Token
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /tokens [get]
func (m tokenModule) getTokenListHandler(c *gin.Context) {
	var dbTokens []database.Token
	err := m.Database.Limit(100).Find(&dbTokens).Error
	if err != nil {
		ginutil.WriteDBReadError(c, err, "Failed fetching list of tokens from database.")
		return
	}

	resTokens := modelconv.DBTokensToResponses(dbTokens)
	c.JSON(http.StatusOK, resTokens)
}

// getTokenHandler godoc
// @id getToken
// @summary Returns token with selected token ID
// @tags token
// @param tokenId path uint true "Token ID"
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

// searchTokenListHandler godoc
// @id searchTokenList
// @summary Returns arrays of tokens that match to search criteria.
// @description Returns arrays of tokens that match to search criteria.
// @description It takes into consideration only token string and user name.
// @tags token
// @accept json
// @produce json
// @param token body request.TokenSearch _ "Token search criteria"
// @success 200 {object} []response.Token
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /tokens/search [post]
func (m tokenModule) searchTokenListHandler(c *gin.Context) {
	var reqToken request.Token
	if err := c.ShouldBindJSON(&reqToken); err != nil {
		ginutil.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for the token object to search with.")
		return
	}

	var dbTokens []database.Token
	err := m.Database.
		Where(&database.Token{
			Token:    reqToken.Token,
			UserName: reqToken.UserName,
		}, database.TokenFields.Token, database.TokenFields.UserName).
		Find(&dbTokens).
		Error
	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed searching for token by value and with username %q in database.",
			reqToken.UserName))
		return
	}

	resTokens := modelconv.DBTokensToResponses(dbTokens)
	c.JSON(http.StatusOK, resTokens)
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
// @param tokenId path uint true "ID of token to update"
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

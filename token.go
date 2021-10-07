package main

import (
	"errors"
	"fmt"

	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/request"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
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
		token.PUT("", m.updateTokenHandler)
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

	resTokens := dbTokensToResponseTokens(dbTokens)
	c.JSON(http.StatusOK, resTokens)
}

// getTokenHandler godoc
// @id getToken
// @summary Returns token with selected token ID
// @tags token
// @param tokenId path int true "Token ID"
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

	var dbToken database.Token
	err := m.Database.First(&dbToken, tokenID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ginutil.WriteDBNotFound(c, fmt.Sprintf(
			"Token with ID %d was not found.", tokenID))
		return
	} else if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching token by ID %d from database.", tokenID))
		return
	}

	resToken := dbTokenToResponseToken(dbToken)
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

	resTokens := dbTokensToResponseTokens(dbTokens)
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

	if reqToken.ProviderID.Valid && reqToken.ProviderID.Int64 != 0 {
		var provider database.Provider
		err := m.Database.Find(&provider, reqToken.ProviderID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ginutil.WriteDBNotFound(c, fmt.Sprintf(
				"Provider with ID %d was not found when creating token.",
				reqToken.ProviderID.Int64))
			return
		} else if err != nil {
			ginutil.WriteDBReadError(c, err, fmt.Sprintf(
				"Failed fetching project with ID %d from database when creating token.",
				reqToken.ProviderID.Int64))
			return
		}
		provider.Token = &dbToken
		if err := m.Database.Save(&provider).Error; err != nil {
			ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
				"Failed updating provider with ID %d with a new token.",
				reqToken.ProviderID.Int64))
			return
		}
	} else {
		if err := m.Database.Create(&dbToken).Error; err != nil {
			ginutil.WriteDBWriteError(c, err, "Failed creating a new token.")
			return
		}
	}

	resToken := dbTokenToResponseToken(dbToken)
	c.JSON(http.StatusCreated, resToken)
}

// updateTokenHandler godoc
// @id updateToken
// @summary Update token in database.
// @description Updates a token by replacing all of its fields.
// @tags token
// @accept json
// @produce json
// @param tokenId path int true "ID of token to update"
// @param token body request.Token _ "New token values"
// @success 200 {object} response.Token
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /token/{tokenId} [put]
func (m tokenModule) updateTokenHandler(c *gin.Context) {
	tokenId, ok := ginutil.ParseParamInt(c, "tokenId")
	if !ok {
		return
	}
	var reqToken request.TokenUpdate
	if err := c.ShouldBindJSON(&reqToken); err != nil {
		ginutil.WriteInvalidBindError(c, err, "One or more parameters failed to parse when reading the request body.")
		return
	}
	var dbToken database.Token
	if err := m.Database.First(&dbToken, tokenId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ginutil.WriteDBNotFound(c, fmt.Sprintf("No token was found by ID %d.", tokenId))
		} else {
			ginutil.WriteDBReadError(c, err, fmt.Sprintf("Failed to find token by ID %d.", tokenId))
		}
		return
	}

	dbToken.Token = reqToken.Token
	dbToken.UserName = reqToken.UserName

	if err := m.Database.Save(&dbToken).Error; err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed to update token by ID %d.",
			tokenId))
	}

	resToken := dbTokenToResponseToken(dbToken)
	c.JSON(http.StatusOK, resToken)
}

func dbTokensToResponseTokens(dbTokens []database.Token) []response.Token {
	resTokens := make([]response.Token, len(dbTokens))
	for i, dbToken := range dbTokens {
		resTokens[i] = dbTokenToResponseToken(dbToken)
	}
	return resTokens
}

func dbTokenToResponseToken(dbToken database.Token) response.Token {
	return response.Token{
		TokenID:  dbToken.TokenID,
		Token:    dbToken.Token,
		UserName: dbToken.UserName,
	}
}

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
		tokens.GET("", m.getTokensHandler)
		tokens.POST("/search", m.postSearchTokenHandler)
	}

	token := g.Group("/token")
	{
		token.GET("/:tokenid", m.getTokenHandler)
		token.POST("", m.postTokenHandler)
		token.PUT("", m.putTokenHandler)
	}
}

// getTokensHandler godoc
// @summary Returns first 100 tokens
// @tags token
// @success 200 {object} []response.Token
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /tokens [get]
func (m tokenModule) getTokensHandler(c *gin.Context) {
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
// @summary Returns token with selected token ID
// @tags token
// @param tokenid path int true "Token ID"
// @success 200 {object} response.Token
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /token/{tokenid} [get]
func (m tokenModule) getTokenHandler(c *gin.Context) {
	tokenID, ok := ginutil.ParseParamUint(c, "tokenid")
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

// postSearchTokenHandler godoc
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
func (m tokenModule) postSearchTokenHandler(c *gin.Context) {
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

// postTokenHandler godoc
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
func (m tokenModule) postTokenHandler(c *gin.Context) {
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
		var provider database.Provider
		err := m.Database.Find(&provider, reqToken.ProviderID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ginutil.WriteDBNotFound(c, fmt.Sprintf(
				"Provider with ID %d was not found when creating token.",
				reqToken.ProviderID))
			return
		} else if err != nil {
			ginutil.WriteDBReadError(c, err, fmt.Sprintf(
				"Failed fetching project with ID %d from database when creating token.",
				reqToken.ProviderID))
			return
		}
		provider.Token = &dbToken
		if err := m.Database.Save(&provider).Error; err != nil {
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

	resToken := dbTokenToResponseToken(dbToken)
	c.JSON(http.StatusCreated, resToken)
}

// postTokenHandler godoc
// @summary Put token in database.
// @description Creates a new token if a match is not found.
// @tags token
// @accept json
// @produce json
// @param token body request.Token _ "Token to add or update"
// @success 200 {object} response.Token
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /token [put]
func (m tokenModule) putTokenHandler(c *gin.Context) {
	var reqToken request.Token
	if err := c.ShouldBindJSON(&reqToken); err != nil {
		ginutil.WriteInvalidBindError(c, err, "One or more parameters failed to parse when reading the request body.")
		return
	}
	dbToken := database.Token{
		Token:    reqToken.Token,
		UserName: reqToken.UserName,
	}
	var dbProvider database.Provider
	if reqToken.ProviderID != 0 {
		if err := m.Database.Find(&dbProvider, reqToken.ProviderID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				ginutil.WriteDBNotFound(c, fmt.Sprintf("No provider was found by ID %d when fetching or creating token by username %q.",
					reqToken.ProviderID,
					reqToken.UserName))
			} else {
				ginutil.WriteDBReadError(c, err, fmt.Sprintf(
					"Failed to find associate provider by ID %d when fetching or creating token by username %q.",
					reqToken.ProviderID,
					reqToken.UserName))
			}
			return
		}
	}
	err := m.Database.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(dbToken).FirstOrCreate(&dbToken).Error; err != nil {
			return err
		}
		if reqToken.ProviderID != 0 {
			dbProvider.Token = &dbToken
			dbProvider.TokenID = dbToken.TokenID
			return tx.Save(&dbProvider).Error
		}
		return nil
	})
	if err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed fetch or create on token by username %q and token value.",
			reqToken.UserName))
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

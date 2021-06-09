package main

import (
	"errors"
	"fmt"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/pkg/httputils"
	"github.com/iver-wharf/wharf-api/pkg/problem"
	"gorm.io/gorm"
)

type TokenModule struct {
	Database *gorm.DB
}

func (m TokenModule) Register(g *gin.RouterGroup) {
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
// @success 200 {object} []Token
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /tokens [get]
func (m TokenModule) getTokensHandler(c *gin.Context) {

	var tokens []Token
	err := m.Database.Limit(100).Find(&tokens).Error
	if err != nil {
		problem.WriteDBReadError(c, err, "Failed fetching list of tokens from database.")
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// getTokenHandler godoc
// @summary Returns token with selected token ID
// @tags token
// @param tokenid path int true "Token ID"
// @success 200 {object} Token
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /token/{tokenid} [get]
func (m TokenModule) getTokenHandler(c *gin.Context) {
	tokenID, ok := httputils.ParseParamUint(c, "tokenid")
	if !ok {
		return
	}

	var token Token
	err := m.Database.First(&token, tokenID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		problem.WriteDBNotFound(c, fmt.Sprintf(
			"Token with ID %d was not found.", tokenID))
		return
	} else if err != nil {
		problem.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching token by ID %d from database.", tokenID))
		return
	}

	c.JSON(http.StatusOK, token)
}

// postSearchTokenHandler godoc
// @summary Returns arrays of tokens that match to search criteria.
// @description Returns arrays of tokens that match to search criteria.
// @description It takes into consideration only token string and user name.
// @tags token
// @accept json
// @produce json
// @param token body Token _ "token object"
// @success 200 {object} []Token
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /tokens/search [post]
func (m TokenModule) postSearchTokenHandler(c *gin.Context) {
	var token Token
	if err := c.ShouldBindJSON(&token); err != nil {
		problem.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for the token object to search with.")
		return
	}

	var tokens []Token
	err := m.Database.
		Where(&token, tokenFieldToken, tokenFieldUserName).
		Find(&tokens).
		Error
	if err != nil {
		problem.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed searching for token by value and with username %q in database.",
			token.UserName))
		return
	}

	c.JSON(http.StatusOK, tokens)
}

type TokenWithProviderID struct {
	Token
	ProviderID uint `json:"providerId"`
}

// postTokenHandler godoc
// @summary Add token to database.
// @description Add token to database. Provider in post object has to exists or should be empty.
// @tags token
// @accept json
// @produce json
// @param token body TokenWithProviderID _ "token object"
// @success 201 {object} Token
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Referenced provider not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /token [post]
func (m TokenModule) postTokenHandler(c *gin.Context) {
	var (
		tokenWithProviderID TokenWithProviderID
		token               *Token
		providerID          uint
	)

	if err := c.ShouldBindJSON(&tokenWithProviderID); err != nil {
		problem.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for the token object to create.")
		return
	}
	token = &tokenWithProviderID.Token
	providerID = tokenWithProviderID.ProviderID

	if providerID != 0 {
		var provider Provider
		err := m.Database.Find(&provider, providerID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			problem.WriteDBNotFound(c, fmt.Sprintf(
				"Provider with ID %d was not found when creating token.",
				providerID))
			return
		} else if err != nil {
			problem.WriteDBReadError(c, err, fmt.Sprintf(
				"Failed fetching project with ID %d from database when creating token.",
				providerID))
			return
		}
		provider.Token = token
		if err := m.Database.Save(&provider).Error; err != nil {
			problem.WriteDBWriteError(c, err, fmt.Sprintf(
				"Failed updating provider with ID %d with a new token.",
				providerID))
			return
		}
	} else {
		if err := m.Database.Create(token).Error; err != nil {
			problem.WriteDBWriteError(c, err, "Failed creating a new token.")
			return
		}
	}

	c.JSON(http.StatusCreated, token)
}

// postTokenHandler godoc
// @summary Put token in database.
// @description Creates a new token if a match is not found.
// @tags token
// @accept json
// @produce json
// @param token body TokenWithProviderID _ "token object"
// @success 200 {object} Token
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /token [put]
func (m TokenModule) putTokenHandler(c *gin.Context) {
	var token Token
	if err := c.ShouldBindJSON(&token); err != nil {
		problem.WriteInvalidBindError(c, err, "One or more parameters failed to parse when reading the request body.")
		return
	}
	var placedToken Token
	if err := m.Database.Where(token).FirstOrCreate(&placedToken).Error; err != nil {
		problem.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed fetch or create on token by username %q and token value.",
			token.UserName))
		return
	}
	c.JSON(http.StatusOK, placedToken)
}

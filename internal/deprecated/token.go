package deprecated

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/request"
	"github.com/iver-wharf/wharf-api/pkg/modelconv"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"gorm.io/gorm"
)

// Token specifies fields when creating a new token using the deprecated
// endpoint
// 	PUT /token
type Token struct {
	Token      string `json:"token" format:"password" validate:"required"`
	UserName   string `json:"userName" validate:"required"`
	ProviderID uint   `json:"providerId"`
}

// TokenModule holds deprecated endpoint handlers for /token
type TokenModule struct {
	Database *gorm.DB
}

// Register adds all deprecated endpoints to a given Gin router group.
func (m TokenModule) Register(g *gin.RouterGroup) {
	tokens := g.Group("/tokens")
	{
		tokens.GET("", m.getTokenListHandler)
		tokens.POST("/search", m.searchTokenListHandler)
	}

	token := g.Group("/token")
	{
		token.PUT("", m.updateTokenHandler)
	}
}

// getTokenListHandler godoc
// @id oldGetTokenList
// @deprecated
// @summary Returns first 100 tokens
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use `GET /token` instead.
// @description Added in v0.2.0.
// @tags token
// @success 200 {object} []response.Token
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /tokens [get]
func (m TokenModule) getTokenListHandler(c *gin.Context) {
	var dbTokens []database.Token
	err := m.Database.Limit(100).Find(&dbTokens).Error
	if err != nil {
		ginutil.WriteDBReadError(c, err, "Failed fetching list of tokens from database.")
		return
	}

	resTokens := modelconv.DBTokensToResponses(dbTokens)
	c.JSON(http.StatusOK, resTokens)
}

// searchTokenListHandler godoc
// @id oldSearchTokenList
// @deprecated
// @summary Returns arrays of tokens that match to search criteria.
// @description Returns arrays of tokens that match to search criteria.
// @description It takes into consideration only token string and user name.
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use `GET /token` instead.
// @description Added in v0.2.10.
// @tags token
// @accept json
// @produce json
// @param token body request.TokenSearch _ "Token search criteria"
// @success 200 {object} []response.Token
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /tokens/search [post]
func (m TokenModule) searchTokenListHandler(c *gin.Context) {
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

// updateTokenHandler godoc
// @id oldUpdateToken
// @deprecated
// @summary Put token in database.
// @description Creates a new token if a match is not found.
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use POST /token to create, or PUT /token/{tokenId} to update instead.
// @description Added in v4.1.0.
// @tags token
// @accept json
// @produce json
// @param token body deprecated.Token _ "Token to add or update"
// @success 200 {object} response.Token
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /token [put]
func (m TokenModule) updateTokenHandler(c *gin.Context) {
	var reqToken Token
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

	resToken := modelconv.DBTokenToResponse(dbToken)
	c.JSON(http.StatusOK, resToken)
}

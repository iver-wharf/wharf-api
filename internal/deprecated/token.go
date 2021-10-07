package deprecated

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/request"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"gorm.io/gorm"
)

type tokenModule struct {
	Database *gorm.DB
}

func (m tokenModule) Register(g *gin.RouterGroup) {
	token := g.Group("/token")
	{
		token.PUT("", m.updateTokenHandler)
	}
}

// updateTokenHandler godoc
// @id updateToken
// @deprecated
// @summary Put token in database.
// @description Creates a new token if a match is not found.
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use POST /token to create, or PUT /token/{tokenId} to update instead.
// @tags token
// @accept json
// @produce json
// @param token body request.Token _ "Token to add or update"
// @success 200 {object} response.Token
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /token [put]
func (m tokenModule) updateTokenHandler(c *gin.Context) {
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

func dbTokenToResponseToken(dbToken database.Token) response.Token {
	return response.Token{
		TokenID:  dbToken.TokenID,
		Token:    dbToken.Token,
		UserName: dbToken.UserName,
	}
}

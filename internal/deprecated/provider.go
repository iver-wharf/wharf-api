package deprecated

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/request"
	"github.com/iver-wharf/wharf-api/pkg/modelconv"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"github.com/iver-wharf/wharf-core/pkg/problem"
	"gorm.io/gorm"
)

// ProviderUpdate specifies fields when updating a provider in the deprecated
// endpoint
// 	PUT /provider
type ProviderUpdate struct {
	ProviderID uint                 `json:"providerId"`
	Name       request.ProviderName `json:"name" enums:"azuredevops,gitlab,github" validate:"required" binding:"required"`
	URL        string               `json:"url" validate:"required" binding:"required"`
	TokenID    uint                 `json:"tokenId"`
}

// ProviderModule holds deprecated endpoint handlers for /provider
type ProviderModule struct {
	Database *gorm.DB
}

// Register adds all deprecated endpoints to a given Gin router group.
func (m ProviderModule) Register(g *gin.RouterGroup) {
	providers := g.Group("/providers")
	{
		providers.GET("", m.getProviderListHandler)
		providers.POST("/search", m.searchProviderListHandler)
	}

	provider := g.Group("/provider")
	{
		provider.PUT("", m.updateProviderHandler)
	}
}

// getProviderListHandler godoc
// @id oldGetProviderList
// @deprecated
// @summary Returns first 100 providers
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use `GET /provider` instead.
// @description Added in v0.3.9.
// @tags provider
// @success 200 {object} []response.Provider
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /providers [get]
func (m ProviderModule) getProviderListHandler(c *gin.Context) {
	var dbProviders []database.Provider
	err := m.Database.Limit(100).Find(&dbProviders).Error
	if err != nil {
		ginutil.WriteDBReadError(c, err, "Failed fetching list of projects from database.")
		return
	}
	resProviders := modelconv.DBProvidersToResponses(dbProviders)
	c.JSON(http.StatusOK, resProviders)
}

// searchProviderListHandler godoc
// @id oldSearchProviderList
// @deprecated
// @summary Returns arrays of providers that match to search criteria.
// @description Returns arrays of providers that match to search criteria.
// @description It takes into consideration name, URL and token ID.
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use `GET /provider` instead.
// @description Added in v0.3.9.
// @tags provider
// @accept json
// @produce json
// @param provider body request.ProviderSearch _ "provider object"
// @success 200 {object} []response.Provider
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /providers/search [post]
func (m ProviderModule) searchProviderListHandler(c *gin.Context) {
	var reqProvider request.ProviderSearch
	if err := c.ShouldBindJSON(&reqProvider); err != nil {
		ginutil.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for the provider object to search with.")
		return
	}

	validName, isValid := reqProvider.Name.ValidString()
	if !isValid {
		writeInvalidProviderNameProblem(c, reqProvider.Name)
		return
	}

	var dbProviders []database.Provider
	if reqProvider.TokenID != 0 {
		err := m.Database.
			Where(&database.Provider{
				Name:    validName,
				URL:     reqProvider.URL,
				TokenID: reqProvider.TokenID,
			},
				database.ProviderFields.Name,
				database.ProviderFields.URL,
				database.ProviderFields.TokenID).
			Find(&dbProviders).
			Error
		if err != nil {
			ginutil.WriteDBReadError(c, err, fmt.Sprintf(
				"Failed fetching list of providers with name %q, URL %q, and with token ID %d from database.",
				reqProvider.Name, reqProvider.URL, reqProvider.TokenID))
			return
		}
	} else {
		err := m.Database.
			Where(&database.Provider{
				Name: validName,
				URL:  reqProvider.URL,
			},
				database.ProviderFields.Name,
				database.ProviderFields.URL).
			Find(&dbProviders).
			Error
		if err != nil {
			ginutil.WriteDBReadError(c, err, fmt.Sprintf(
				"Failed fetching list of providers with name %q and URL %q from database.",
				reqProvider.Name, reqProvider.URL))
			return
		}
	}

	resProviders := modelconv.DBProvidersToResponses(dbProviders)
	c.JSON(http.StatusOK, resProviders)
}

// updateProviderHandler godoc
// @id oldUpdateProvider
// @deprecated
// @summary Put provider in database.
// @description Creates a new provider if a match is not found.
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use POST /provider to create, or PUT /provider/{providerId} to update instead.
// @description Added in v4.1.0.
// @tags provider
// @accept json
// @produce json
// @param provider body deprecated.ProviderUpdate _ "provider object"
// @success 200 {object} response.Provider
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /provider [put]
func (m ProviderModule) updateProviderHandler(c *gin.Context) {
	var reqProviderUpdate ProviderUpdate
	if err := c.ShouldBindJSON(&reqProviderUpdate); err != nil {
		ginutil.WriteInvalidBindError(c, err, "One or more parameters failed to parse when reading the request body.")
		return
	}
	validName, isValid := reqProviderUpdate.Name.ValidString()
	if !isValid {
		writeInvalidProviderNameProblem(c, reqProviderUpdate.Name)
		return
	}
	dbProvider := database.Provider{
		ProviderID: reqProviderUpdate.ProviderID,
		Name:       validName,
		URL:        reqProviderUpdate.URL,
		TokenID:    reqProviderUpdate.TokenID,
	}
	if err := m.Database.Where(dbProvider).FirstOrCreate(&dbProvider).Error; err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed fetch or create on inputProvider with name %q.",
			reqProviderUpdate.Name))
		return
	}
	resProvider := modelconv.DBProviderToResponse(dbProvider)
	c.JSON(http.StatusOK, resProvider)
}

func writeInvalidProviderNameProblem(c *gin.Context, actual request.ProviderName) {
	ginutil.WriteProblem(c, problem.Response{
		Type:   "/prob/api/provider/invalid-name",
		Title:  "Invalid provider name.",
		Status: http.StatusBadRequest,
		Detail: fmt.Sprintf(
			"Provider name was %q but can only be one of the following values: %s.",
			actual, request.ProviderNameValues),
		Instance: c.Request.RequestURI + "#name",
	})
}

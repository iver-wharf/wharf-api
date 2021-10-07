package main

import (
	"errors"
	"fmt"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/request"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"github.com/iver-wharf/wharf-core/pkg/problem"
	"gorm.io/gorm"
)

type providerModule struct {
	Database *gorm.DB
}

func (m providerModule) Register(g *gin.RouterGroup) {
	providers := g.Group("/providers")
	{
		providers.GET("", m.getProviderListHandler)
		providers.POST("/search", m.searchProviderListHandler)
	}

	provider := g.Group("/provider")
	{
		provider.GET("/:providerId", m.getProviderHandler)
		provider.POST("", m.createProviderHandler)
		provider.PUT("", m.updateProviderHandler)
	}
}

// getProviderListHandler godoc
// @id getProviderList
// @summary Returns first 100 providers
// @tags provider
// @success 200 {object} []response.Provider
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /providers [get]
func (m providerModule) getProviderListHandler(c *gin.Context) {
	var dbProviders []database.Provider
	err := m.Database.Limit(100).Find(&dbProviders).Error
	if err != nil {
		ginutil.WriteDBReadError(c, err, "Failed fetching list of projects from database.")
		return
	}
	resProviders := dbProvidersToResponses(dbProviders)
	c.JSON(http.StatusOK, resProviders)
}

// getProviderHandler godoc
// @id getProvider
// @summary Returns provider with selected provider ID
// @tags provider
// @param providerId path int true "Provider ID"
// @success 200 {object} response.Provider
// @failure 400 {object} problem.Response "Bad request"
// @failure 404 {object} problem.Response "Provider not found"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /provider/{providerId} [get]
func (m providerModule) getProviderHandler(c *gin.Context) {
	providerID, ok := ginutil.ParseParamUint(c, "providerId")
	if !ok {
		return
	}

	var dbProvider database.Provider
	err := m.Database.First(&dbProvider, providerID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ginutil.WriteDBNotFound(c, fmt.Sprintf(
			"Provider with ID %d was not found.",
			providerID))
		return
	} else if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching provider with ID %d from database.",
			providerID))
		return
	}

	resProvider := dbProviderToResponse(dbProvider)
	c.JSON(http.StatusOK, resProvider)
}

// searchProviderListHandler godoc
// @id searchProviderList
// @summary Returns arrays of providers that match to search criteria.
// @description Returns arrays of providers that match to search criteria.
// @description It takes into consideration name, URL and token ID.
// @tags provider
// @accept json
// @produce json
// @param provider body request.ProviderSearch _ "provider object"
// @success 200 {object} []response.Provider
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /providers/search [post]
func (m providerModule) searchProviderListHandler(c *gin.Context) {
	var reqProvider request.ProviderSearch
	if err := c.ShouldBindJSON(&reqProvider); err != nil {
		ginutil.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for the provider object to search with.")
		return
	}

	validName, isValid := validateRequestProviderName(reqProvider.Name)
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

	resProviders := dbProvidersToResponses(dbProviders)
	c.JSON(http.StatusOK, resProviders)
}

// createProviderHandler godoc
// @id createProvider
// @summary Add provider to database.
// @description Add provider to database. Token in post object has to exists or should be empty.
// @description Token will has to be updated Provider ID during this operation.
// @tags provider
// @accept json
// @produce json
// @param provider body request.Provider _ "provider object"
// @success 201 {object} response.Provider
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /provider [post]
func (m providerModule) createProviderHandler(c *gin.Context) {
	var reqProvider request.Provider
	if err := c.ShouldBindJSON(&reqProvider); err != nil {
		ginutil.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for the provider object to search with.")
		return
	}

	validName, isValid := validateRequestProviderName(reqProvider.Name)
	if !isValid {
		writeInvalidProviderNameProblem(c, reqProvider.Name)
		return
	}

	dbProvider := database.Provider{
		Name:    validName,
		URL:     reqProvider.URL,
		TokenID: reqProvider.TokenID,
	}
	// Sets provider.TokenID through association
	if err := m.Database.Create(&dbProvider).Error; err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed to create provider with name %q and URL %q to database.",
			reqProvider.Name, reqProvider.URL))
		return
	}

	resProvider := dbProviderToResponse(dbProvider)
	c.JSON(http.StatusCreated, resProvider)
}

// updateProviderHandler godoc
// @id updateProvider
// @summary Put provider in database.
// @description Creates a new provider if a match is not found.
// @tags provider
// @accept json
// @produce json
// @param provider body request.ProviderUpdate _ "provider object"
// @success 200 {object} response.Provider
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /provider [put]
func (m providerModule) updateProviderHandler(c *gin.Context) {
	var reqProviderUpdate request.ProviderUpdate
	if err := c.ShouldBindJSON(&reqProviderUpdate); err != nil {
		ginutil.WriteInvalidBindError(c, err, "One or more parameters failed to parse when reading the request body.")
		return
	}
	validName, isValid := validateRequestProviderName(reqProviderUpdate.Name)
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
	resProvider := dbProviderToResponse(dbProvider)
	c.JSON(http.StatusOK, resProvider)
}

func fetchProviderByID(c *gin.Context, db *gorm.DB, providerID uint) (database.Provider, bool) {
	var dbProvider database.Provider
	if err := db.Find(&dbProvider, providerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ginutil.WriteDBNotFound(c, fmt.Sprintf(
				"Provider with ID %d was not found when creating token.",
				providerID))
		} else {
			ginutil.WriteDBReadError(c, err, fmt.Sprintf(
				"Failed fetching project with ID %d from database when creating token.",
				providerID))
		}
		return database.Provider{}, false
	}
	return dbProvider, true
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

func validateRequestProviderName(name request.ProviderName) (string, bool) {
	if !name.IsValid() {
		return "", false
	}
	return string(name), true
}

func dbProvidersToResponses(dbProviders []database.Provider) []response.Provider {
	resProviders := make([]response.Provider, len(dbProviders))
	for i, dbProvider := range dbProviders {
		resProviders[i] = dbProviderToResponse(dbProvider)
	}
	return resProviders
}

func dbProviderToResponse(dbProvider database.Provider) response.Provider {
	return response.Provider{
		ProviderID: dbProvider.ProviderID,
		Name:       response.ProviderName(dbProvider.Name),
		URL:        dbProvider.URL,
		TokenID:    dbProvider.TokenID,
	}
}

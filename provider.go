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

type ProviderName int

const (
	Azuredevops ProviderName = 0 + iota
	Gitlab
	Github
)

func (provider ProviderName) toString() string {
	return [...]string{"azuredevops", "gitlab", "github"}[provider]
}

type ProviderModule struct {
	Database *gorm.DB
}

func (m ProviderModule) Register(g *gin.RouterGroup) {
	providers := g.Group("/providers")
	{
		providers.GET("", m.getProvidersHandler)
		providers.POST("/search", m.postSearchProviderHandler)
	}

	provider := g.Group("/provider")
	{
		provider.GET("/:providerid", m.getProviderHandler)
		provider.POST("", m.postProviderHandler)
	}
}

// getProvidersHandler godoc
// @summary Returns first 100 providers
// @tags provider
// @success 200 {object} []Provider
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /providers [get]
func (m ProviderModule) getProvidersHandler(c *gin.Context) {
	var providers []Provider
	err := m.Database.Limit(100).Find(&providers).Error
	if err != nil {
		problem.WriteDBReadError(c, err, "Failed fetching list of projects from database.")
		return
	}
	c.JSON(http.StatusOK, providers)
}

// getProviderHandler godoc
// @summary Returns provider with selected provider ID
// @tags provider
// @param providerid path int true "Provider ID"
// @success 200 {object} Provider
// @failure 400 {object} problem.Response "Bad request"
// @failure 404 {object} problem.Response "Provider not found"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /provider/{providerid} [get]
func (m ProviderModule) getProviderHandler(c *gin.Context) {
	providerID, ok := httputils.ParseParamUint(c, "providerid")
	if !ok {
		return
	}

	var provider Provider
	err := m.Database.First(&provider, providerID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		problem.WriteDBNotFound(c, fmt.Sprintf(
			"Provider with ID %d was not found.",
			providerID))
		return
	} else if err != nil {
		problem.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching provider with ID %d from database.",
			providerID))
		return
	}

	c.JSON(http.StatusOK, provider)
}

// postSearchProviderHandler godoc
// @summary Returns arrays of providers that match to search criteria.
// @description Returns arrays of providers that match to search criteria.
// @description It takes into consideration name, URL, UploadURL and token ID.
// @tags provider
// @accept json
// @produce json
// @param provider body Provider _ "provider object"
// @success 200 {object} []Provider
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /providers/search [post]
func (m ProviderModule) postSearchProviderHandler(c *gin.Context) {
	var provider Provider
	if err := c.ShouldBindJSON(&provider); err != nil {
		problem.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for the provider object to search with.")
		return
	}

	var providers []Provider
	if provider.TokenID != 0 {
		err := m.Database.
			Where(&provider, providerFieldName, providerFieldURL, providerFieldUploadURL, providerFieldTokenID).
			Find(&providers).
			Error
		if err != nil {
			problem.WriteDBReadError(c, err, fmt.Sprintf(
				"Failed fetching list of providers with name %q, URL %q, upload URL %q, and with token ID %d from database.",
				provider.Name, provider.URL, provider.UploadURL, provider.TokenID))
			return
		}
	} else {
		err := m.Database.
			Where(&provider, providerFieldName, providerFieldURL, providerFieldUploadURL).
			Find(&providers).
			Error
		if err != nil {
			problem.WriteDBReadError(c, err, fmt.Sprintf(
				"Failed fetching list of providers with name %q, URL %q, and with upload URL %q from database.",
				provider.Name, provider.URL, provider.UploadURL))
			return
		}
	}

	c.JSON(http.StatusOK, providers)
}

// postProviderHandler godoc
// @summary Add provider to database.
// @description Add provider to database. Token in post object has to exists or should be empty.
// @description Token will has to be updated Provider ID during this operation.
// @tags provider
// @accept json
// @produce json
// @param provider body Provider _ "provider object"
// @success 201 {object} Provider
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /provider [post]
func (m ProviderModule) postProviderHandler(c *gin.Context) {
	var provider Provider
	if err := c.ShouldBindJSON(&provider); err != nil {
		problem.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for the provider object to search with.")
		return
	}

	if provider.Name != Azuredevops.toString() && provider.Name != Gitlab.toString() && provider.Name != Github.toString() {
		problem.WriteProblem(c, problem.Response{
			Type:   "/prob/api/provider/invalid-name",
			Title:  "Invalid provider name.",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf(
				"Provider name was %q but can only be one of the following values: %q, %q, %q.",
				provider.Name, Azuredevops.toString(), Gitlab.toString(), Github.toString()),
			Instance: c.Request.RequestURI + "#name",
		})
		return
	}

	// Sets provider.TokenID through association
	if err := m.Database.Create(&provider).Error; err != nil {
		problem.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed to create provider with name %q and URL %q to database.",
			provider.Name, provider.URL))
		return
	}

	c.JSON(http.StatusCreated, provider)
}

// putProviderHandler godoc
// @summary Put provider in database.
// @description Creates a new provider if a match is not found.
// @tags provider
// @accept json
// @produce json
// @param provider body Provider _ "provider object"
// @success 200 {object} Provider
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /provider [put]
func (m ProviderModule) putProviderHandler(c *gin.Context) {
	var inputProvider Provider
	if err := c.ShouldBindJSON(&inputProvider); err != nil {
		problem.WriteInvalidBindError(c, err, "One or more parameters failed to parse when reading the request body.")
		return
	}
	var provider Provider
	if err := m.Database.Where(inputProvider).FirstOrCreate(&provider).Error; err != nil {
		problem.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed fetch or create on inputProvider with name %q.",
			inputProvider.Name))
		return
	}
	c.JSON(http.StatusOK, provider)
}

package main

import (
	"fmt"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/v5/internal/wherefields"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/request"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/response"
	"github.com/iver-wharf/wharf-api/v5/pkg/modelconv"
	"github.com/iver-wharf/wharf-api/v5/pkg/orderby"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"github.com/iver-wharf/wharf-core/pkg/problem"
	"gorm.io/gorm"
)

type providerModule struct {
	Database *gorm.DB
}

func (m providerModule) Register(g *gin.RouterGroup) {
	provider := g.Group("/provider")
	{
		provider.GET("", m.getProviderListHandler)
		provider.POST("", m.createProviderHandler)

		providerByID := provider.Group("/:providerId")
		{
			providerByID.GET("", m.getProviderHandler)
			providerByID.PUT("", m.updateProviderHandler)
		}
	}
}

var providerJSONToColumns = map[string]database.SafeSQLName{
	response.ProviderJSONFields.ProviderID: database.ProviderColumns.ProviderID,
	response.ProviderJSONFields.Name:       database.ProviderColumns.Name,
	response.ProviderJSONFields.URL:        database.ProviderColumns.URL,
}

var defaultGetProvidersOrderBy = orderby.Column{Name: database.ProviderColumns.ProviderID, Direction: orderby.Desc}

// getBuildListHandler godoc
// @id getProviderList
// @summary Get slice of providers.
// @description List all providers, or a window of providers using the `limit` and `offset` query parameters. Allows optional filtering parameters.
// @description Verbatim filters will match on the entire string used to find exact matches,
// @description while the matching filters are meant for searches by humans where it tries to find soft matches and is therefore inaccurate by nature.
// @description Added in v5.0.0.
// @tags provider
// @produce json
// @param limit query int false "Number of results to return. No limiting is applied if empty (`?limit=`) or non-positive (`?limit=0`). Required if `offset` is used." default(100)
// @param offset query int false "Skipped results, where 0 means from the start." minimum(0) default(0)
// @param orderby query []string false "Sorting orders. Takes the property name followed by either 'asc' or 'desc'. Can be specified multiple times for more granular sorting. Defaults to `?orderby=providerId desc`"
// @param name query string false "Filter by verbatim provider name."
// @param url query string false "Filter by verbatim provider URL."
// @param nameMatch query string false "Filter by matching provider name. Cannot be used with `name`."
// @param urlMatch query string false "Filter by matching provider URL. Cannot be used with `url`."
// @param match query string false "Filter by matching on any supported fields."
// @param pretty query bool false "Pretty indented JSON output"
// @success 200 {object} response.PaginatedProviders
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt provider"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /provider [get]
func (m providerModule) getProviderListHandler(c *gin.Context) {
	var params = struct {
		commonGetQueryParams

		Name *string `form:"name"`
		URL  *string `form:"url"`

		NameMatch *string `form:"nameMatch" binding:"excluded_with=Name"`
		URLMatch  *string `form:"urlMatch" binding:"excluded_with=URL"`

		Match *string `form:"match"`
	}{
		commonGetQueryParams: defaultCommonGetQueryParams,
	}
	if !bindCommonGetQueryParams(c, &params) {
		return
	}
	orderBySlice, ok := parseCommonOrderBySlice(c, params.OrderBy, providerJSONToColumns)
	if !ok {
		return
	}

	var where wherefields.Collection
	query := m.Database.
		Clauses(orderBySlice.ClauseIfNone(defaultGetProvidersOrderBy)).
		Where(&database.Provider{
			Name: where.String(database.ProviderFields.Name, params.Name),
			URL:  where.String(database.ProviderFields.URL, params.URL),
		}, where.NonNilFieldNames()...).
		Scopes(
			whereLikeScope(map[database.SafeSQLName]*string{
				database.ProviderColumns.Name: params.NameMatch,
				database.ProviderColumns.URL:  params.URLMatch,
			}),
			whereAnyLikeScope(
				params.Match,
				database.ProviderColumns.Name,
				database.ProviderColumns.URL,
			),
		)

	var dbProviders []database.Provider
	var totalCount int64
	err := findDBPaginatedSliceAndTotalCount(query, params.Limit, params.Offset, &dbProviders, &totalCount)
	if err != nil {
		ginutil.WriteDBReadError(c, err, "Failed fetching list of providers from database.")
		return
	}

	renderJSON(c, http.StatusOK, response.PaginatedProviders{
		List:       modelconv.DBProvidersToResponses(dbProviders),
		TotalCount: totalCount,
	})
}

// getProviderHandler godoc
// @id getProvider
// @summary Returns provider with selected provider ID
// @description Added in v0.3.9.
// @tags provider
// @produce json
// @param providerId path uint true "Provider ID" minimum(0)
// @param pretty query bool false "Pretty indented JSON output"
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

	dbProvider, ok := fetchProviderByID(c, m.Database, providerID, "")
	if !ok {
		return
	}

	resProvider := modelconv.DBProviderToResponse(dbProvider)
	renderJSON(c, http.StatusOK, resProvider)
}

// createProviderHandler godoc
// @id createProvider
// @summary Add provider to database.
// @description Add provider to database. Token in post object has to exists or should be empty.
// @description Token will has to be updated Provider ID during this operation.
// @description Added in v0.3.9.
// @tags provider
// @accept json
// @produce json
// @param provider body request.Provider _ "Provider to create"
// @param pretty query bool false "Pretty indented JSON output"
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

	validName, isValid := reqProvider.Name.ValidString()
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

	resProvider := modelconv.DBProviderToResponse(dbProvider)
	renderJSON(c, http.StatusCreated, resProvider)
}

// updateProviderHandler godoc
// @id updateProvider
// @summary Update provider in database.
// @description Updates a provider by replacing all of its fields.
// @description Added in v5.0.0.
// @tags provider
// @accept json
// @produce json
// @param providerId path uint _ "ID of provider to update" minimum(0)
// @param provider body request.ProviderUpdate _ "New provider values"
// @param pretty query bool false "Pretty indented JSON output"
// @success 200 {object} response.Provider
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Provider or token not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /provider/{providerId} [put]
func (m providerModule) updateProviderHandler(c *gin.Context) {
	providerID, ok := ginutil.ParseParamUint(c, "providerId")
	if !ok {
		return
	}
	var reqProviderUpdate request.ProviderUpdate
	if err := c.ShouldBindJSON(&reqProviderUpdate); err != nil {
		ginutil.WriteInvalidBindError(c, err, "One or more parameters failed to parse when reading the request body.")
		return
	}
	validName, isValid := reqProviderUpdate.Name.ValidString()
	if !isValid {
		writeInvalidProviderNameProblem(c, reqProviderUpdate.Name)
		return
	}
	dbProvider, ok := fetchProviderByID(c, m.Database, providerID, "when updating provider")
	if !ok {
		return
	}
	if reqProviderUpdate.TokenID != 0 {
		// Only called to validate the TokenID field
		_, ok := fetchTokenByID(c, m.Database, reqProviderUpdate.TokenID, "when updating provider")
		if !ok {
			return
		}
	}

	dbProvider.Name = validName
	dbProvider.URL = reqProviderUpdate.URL
	dbProvider.TokenID = reqProviderUpdate.TokenID

	if err := m.Database.Save(&dbProvider).Error; err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed to update provider by ID %d.",
			providerID))
		return
	}

	resProvider := modelconv.DBProviderToResponse(dbProvider)
	renderJSON(c, http.StatusOK, resProvider)
}

func fetchProviderByID(c *gin.Context, db *gorm.DB, providerID uint, whenMsg string) (database.Provider, bool) {
	var dbProvider database.Provider
	ok := fetchDatabaseObjByID(c, db, &dbProvider, providerID, "provider", whenMsg)
	return dbProvider, ok
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

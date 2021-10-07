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

func (m ProviderModule) Register(g *gin.RouterGroup) {
	provider := g.Group("/provider")
	{
		provider.PUT("", m.updateProviderHandler)
	}
}

// updateProviderHandler godoc
// @id updateProvider
// @summary Put provider in database.
// @description Creates a new provider if a match is not found.
// @description Deprecated since v5.0.0. Planned for removal in v6.0.0.
// @description Use POST /provider to create, or PUT /provider/{providerId} to update instead.
// @tags provider
// @accept json
// @produce json
// @param provider body request.ProviderUpdate _ "provider object"
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

func validateRequestProviderName(name request.ProviderName) (string, bool) {
	if !name.IsValid() {
		return "", false
	}
	return string(name), true
}

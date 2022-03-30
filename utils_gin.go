package main

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/iver-wharf/wharf-api/v5/pkg/orderby"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	ua "github.com/mileusna/useragent"
)

type commonGetQueryParams struct {
	Limit  int `form:"limit" binding:"required_with=Offset"`
	Offset int `form:"offset" binding:"min=0"`

	OrderBy []string `form:"orderby"`
}

var defaultCommonGetQueryParams = commonGetQueryParams{
	Limit:  100,
	Offset: 0,
}

func bindCommonGetQueryParams(c *gin.Context, params any) bool {
	if err := c.ShouldBindQuery(params); err != nil {
		ginutil.WriteInvalidBindError(c, err, "One or more parameters failed to parse when reading query parameters.")
		return false
	}
	return true
}

func parseCommonOrderBySlice(c *gin.Context, orders []string, fieldToColumnNames map[string]database.SafeSQLName) (orderby.Slice, bool) {
	orderBySlice, err := orderby.ParseSlice(orders, fieldToColumnNames)
	if err != nil {
		joinedOrders := strings.Join(orders, ", ")
		ginutil.WriteInvalidParamError(c, err, "orderby", fmt.Sprintf(
			"Failed parsing the %d sort ordering values: %s",
			len(orders),
			joinedOrders))
		return nil, false
	}
	return orderBySlice, true
}

func renderJSON(c *gin.Context, code int, response any) {
	if shouldIndentJSONResponse(c) {
		c.IndentedJSON(code, response)
	} else {
		c.JSON(code, response)
	}
}

func shouldIndentJSONResponse(c *gin.Context) bool {
	prettyQuery, ok := c.GetQuery("pretty")
	if ok {
		return prettyQuery == "" || strings.EqualFold(prettyQuery, "true")
	}
	agent := ua.Parse(c.Request.UserAgent())
	return agent.Name == "curl"
}

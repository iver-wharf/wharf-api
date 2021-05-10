package httputils

import (
	"fmt"
	"math/bits"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/pkg/problem"
)

// RequireParamString tries to read the named path parameter from the request
// and checks that it's not empty.
//
// If it fails, it will write out a problem response using
// problem.WriteProblem with the status code 400 (Bad Request).
func RequireParamString(c *gin.Context, paramName string) (string, bool) {
	return requireString(c, paramName, c.Param(paramName))
}

// RequireQueryString tries to read the named query parameter from the request
// and checks that it's not empty.
//
// If it fails, it will write out a problem response using
// problem.WriteProblem with the status code 400 (Bad Request).
func RequireQueryString(c *gin.Context, queryName string) (string, bool) {
	return requireString(c, queryName, c.Query(queryName))
}

func requireString(c *gin.Context, paramName string, paramValue string) (string, bool) {
	if paramValue == "" {
		problem.WriteProblem(c, problem.Response{
			Type:     "/prob/api/missing-param-string",
			Title:    "Missing string value.",
			Status:   http.StatusBadRequest,
			Detail:   fmt.Sprintf("A string value (text) was expected on parameter %q, but it was either omitted or empty.", paramName),
			Instance: fmt.Sprintf("%s#%s", c.Request.RequestURI, paramName),
		})
		return "", false
	}
	return paramValue, true
}

// ParseParamUint tries to read the named path parameter from the request and
// parse it to an uint.
//
// If it fails, it will write out a problem response using
// problem.WriteProblemError with the status code 400 (Bad Request).
func ParseParamUint(c *gin.Context, paramName string) (uint, bool) {
	return parseUint(c, paramName, c.Param(paramName))
}

// ParseParamInt tries to read the named path parameter from the request and
// parse it to an int.
//
// If it fails, it will write out a problem response using
// problem.WriteProblemError with the status code 400 (Bad Request).
func ParseParamInt(c *gin.Context, paramName string) (int, bool) {
	return parseInt(c, paramName, c.Param(paramName))
}

// ParseQueryUint tries to read the named query parameter from the request and
// parse it to an uint.
//
// If it fails, it will write out a problem response using
// problem.WriteProblemError with the status code 400 (Bad Request).
func ParseQueryUint(c *gin.Context, queryName string) (uint, bool) {
	return parseUint(c, queryName, c.Query(queryName))
}

// ParseQueryInt tries to read the named query parameter from the request and
// parse it to an int.
//
// If it fails, it will write out a problem response using
// problem.WriteProblemError with the status code 400 (Bad Request).
func ParseQueryInt(c *gin.Context, queryName string) (int, bool) {
	return parseInt(c, queryName, c.Query(queryName))
}

func parseUint(c *gin.Context, paramName, paramValue string) (uint, bool) {
	value, err := strconv.ParseUint(paramValue, 10, bits.UintSize)
	if err != nil {
		problem.WriteProblemError(c, err, problem.Response{
			Type:     "/prob/api/invalid-param-uint",
			Title:    "Invalid positive integer value.",
			Status:   http.StatusBadRequest,
			Detail:   fmt.Sprintf("Failed to interpret parameter %q with value %q as an unsigned (positive) integer.", paramName, paramValue),
			Instance: fmt.Sprintf("%s#%s", c.Request.RequestURI, paramName),
		})
		return 0, false
	}
	return uint(value), true
}

func parseInt(c *gin.Context, paramName, paramValue string) (int, bool) {
	value, err := strconv.ParseInt(paramValue, 10, bits.UintSize)
	if err != nil {
		problem.WriteProblemError(c, err, problem.Response{
			Type:     "/prob/api/invalid-param-int",
			Title:    "Invalid integer value.",
			Status:   http.StatusBadRequest,
			Detail:   fmt.Sprintf("Failed to interpret parameter %q with value %q as a signed (positive or negative) integer.", paramName, paramValue),
			Instance: fmt.Sprintf("%s#%s", c.Request.RequestURI, paramValue),
		})
		return 0, false
	}
	return int(value), true
}

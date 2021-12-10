package main

import (
	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/response"
)

type healthModule struct{}

func (m healthModule) Register(g *gin.RouterGroup) {
	g.GET("/ping", m.pingHandler)
	g.GET("/health", m.healthHandler)
}

// DeprecatedRegister adds API health-related endpoints to a Gin-Gonic engine.
//
// Deprecated: Not part of the /api group for endpoints. Tentatively planned
// for complete removal in v6.
func (m healthModule) DeprecatedRegister(e *gin.Engine) {
	e.GET("/", m.pingHandler)         // added in v0.1.8
	e.GET("/health", m.healthHandler) // added in v0.7.1
}

// pingHandler godoc
// @id pingHandler
// @summary Ping
// @description You guessed it. Pong.
// @description Added in v4.2.0.
// @tags health
// @produce json
// @success 200 {object} response.Ping
// @router /ping [get]
func (m healthModule) pingHandler(c *gin.Context) {
	c.JSON(200, response.Ping{Message: "pong"})
}

// healthHandler godoc
// @id getHealth
// @summary Healthcheck for the API
// @description To be used by Kubernetes or alike.
// @description Added in v0.7.1.
// @tags health
// @produce json
// @success 200 {object} response.HealthStatus
// @router /health [get]
func (m healthModule) healthHandler(c *gin.Context) {
	c.JSON(200, response.HealthStatus{Message: "API is healthy.", IsHealthy: true})
}

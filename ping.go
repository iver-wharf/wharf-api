package main

import (
	"github.com/gin-gonic/gin"
)

type healthModule struct{}

func (m healthModule) Register(g *gin.RouterGroup) {
	g.GET("/ping", m.ping)
	g.GET("/health", m.health)
}

// DeprecatedRegister adds API health-related endpoints to a Gin-Gonic engine.
//
// Deprecated: Not part of the /api group for endpoints. Tentatively planned
// for complete removal in v6.
func (m healthModule) DeprecatedRegister(e *gin.Engine) {
	e.GET("/", m.ping)
	e.GET("/health", m.health)
}

// Ping pongs.
type Ping struct {
	Message string `json:"message"`
}

// ping godoc
// @summary Ping
// @description You guessed it. Pong.
// @tags health
// @produce json
// @success 200 {object} Ping
// @router /ping [get]
func (m healthModule) ping(c *gin.Context) {
	c.JSON(200, Ping{Message: "pong"})
}

// HealthStatus holds a human-readable string stating the health of the API and
// its integrations, as well as a boolean for easy machine-readability.
type HealthStatus struct {
	Message   string `example:"API is healthy." json:"message"`
	IsHealthy bool   `example:"true" json:"isHealthy"`
}

// health godoc
// @summary Healthcheck for the API
// @description To be used by Kubernetes or alike.
// @tags health
// @produce json
// @success 200 {object} HealthStatus
// @router /health [get]
func (m healthModule) health(c *gin.Context) {
	c.JSON(200, HealthStatus{Message: "API is healthy.", IsHealthy: true})
}

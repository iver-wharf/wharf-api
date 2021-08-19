package main

import (
	"github.com/gin-gonic/gin"
)

type HealthModule struct{}

func (m HealthModule) Register(g *gin.RouterGroup) {
	g.GET("/ping", m.ping)
	g.GET("/health", m.health)
}

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
func (m HealthModule) ping(c *gin.Context) {
	c.JSON(200, Ping{Message: "pong"})
}

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
func (m HealthModule) health(c *gin.Context) {
	c.JSON(200, HealthStatus{Message: "API is healthy.", IsHealthy: true})
}

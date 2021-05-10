package main

import (
	"github.com/gin-gonic/gin"
)

type HTTPModule interface {
	Register(*gin.RouterGroup)
}

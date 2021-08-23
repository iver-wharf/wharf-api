package main

import (
	"github.com/gin-gonic/gin"
)

type httpModule interface {
	Register(*gin.RouterGroup)
}

package problem

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RecoverHandle(c *gin.Context, err interface{}) {
	WriteProblem(c, Response{
		Type:   "/prob/api/internal-server-error",
		Title:  "Internal server error.",
		Status: http.StatusInternalServerError,
		Detail: fmt.Sprintf("Unhandled error: %s", err),
	})
}

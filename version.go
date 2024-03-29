package main

import (
	"net/http"

	_ "embed"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-core/pkg/app"
)

// AppVersion holds metadata about this application's version. This value is
// exposed from the following endpoint:
//	GET /api/version
var AppVersion app.Version

//go:embed version.yaml
var versionFile []byte

func loadEmbeddedVersionFile() error {
	return app.UnmarshalVersionYAML(versionFile, &AppVersion)
}

// getVersionHandler godoc
// @id getVersion
// @summary Returns the version of this API
// @description Added in v4.0.0.
// @tags meta
// @produce json
// @param pretty query bool false "Pretty indented JSON output"
// @success 200 {object} app.Version
// @router /version [get]
func getVersionHandler(c *gin.Context) {
	renderJSON(c, http.StatusOK, AppVersion)
}

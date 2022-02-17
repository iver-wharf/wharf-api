package main

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/response"
)

type engineModule struct {
	CIConfig *CIConfig
}

func (m engineModule) Register(r *gin.RouterGroup) {
	r.GET("/engine", m.getEngineList)
}

// getEngineList godoc
// @id getEngineList
// @summary Get list of engines.
// @description Added in v5.1.0.
// @tags engine
// @success 200 {object} response.EngineList "Engines"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @router /engine [get]
func (m engineModule) getEngineList(c *gin.Context) {
	if m.CIConfig == nil {
		c.JSON(200, response.EngineList{})
		return
	}
	conf := *m.CIConfig
	var res response.EngineList
	if defaultEng, hasDefault := getDefaultEngineFromConfig(conf); hasDefault {
		resDefaultEng := convCIEngineToResponse(defaultEng)
		res.DefaultEngine = &resDefaultEng
	}
	engines := getEnginesFromConfig(conf)
	res.List = convCIEnginesToResponses(engines)
	c.JSON(200, res)
}

func getEnginesFromConfig(ciConf CIConfig) []CIEngineConfig {
	var engines []CIEngineConfig
	if ciConf.Engine.URL != "" {
		engines = append(engines, ciConf.Engine)
	}
	if ciConf.Engine2.URL != "" {
		engines = append(engines, ciConf.Engine2)
	}
	return engines
}

func getDefaultEngineFromConfig(ciConf CIConfig) (CIEngineConfig, bool) {
	switch {
	case ciConf.Engine.URL != "":
		return ciConf.Engine, true
	case ciConf.Engine2.URL != "":
		return ciConf.Engine2, true
	default:
		return CIEngineConfig{}, false
	}
}

func lookupEngineOrDefaultFromConfig(ciConf CIConfig, id string) (CIEngineConfig, bool) {
	switch {
	case ciConf.MockTriggerResponse:
		return CIEngineConfig{
			ID:   "mock",
			Name: "Mock engine",
			URL:  "http://mock-url.wharf-api.localhost",
		}, true
	case id == "":
		return getDefaultEngineFromConfig(ciConf)
	default:
		return lookupEngineFromConfig(ciConf, id)
	}
}

func lookupEngineFromConfig(ciConf CIConfig, id string) (CIEngineConfig, bool) {
	switch {
	case id == "":
		return getDefaultEngineFromConfig(ciConf)
	case strings.EqualFold(ciConf.Engine.ID, id):
		return ciConf.Engine, true
	case strings.EqualFold(ciConf.Engine2.ID, id):
		return ciConf.Engine2, true
	default:
		return CIEngineConfig{}, false
	}
}

func convCIEngineToResponse(engine CIEngineConfig) response.Engine {
	return response.Engine{
		ID:   engine.ID,
		Name: engine.Name,
		URL:  engine.URL,
	}
}

func convCIEnginesToResponses(engines []CIEngineConfig) []response.Engine {
	resEngines := make([]response.Engine, len(engines))
	for i, e := range engines {
		resEngines[i] = convCIEngineToResponse(e)
	}
	return resEngines
}

func lookupResponseEngineFromConfig(ciConf CIConfig, id string) *response.Engine {
	engine, ok := lookupEngineFromConfig(ciConf, id)
	if !ok {
		return nil
	}
	resEngine := convCIEngineToResponse(engine)
	return &resEngine
}

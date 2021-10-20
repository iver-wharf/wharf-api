package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func findDefaultBranch(branches []database.Branch) (database.Branch, bool) {
	for _, branch := range branches {
		if branch.Default {
			return branch, true
		}
	}

	return database.Branch{}, false
}

func fetchDatabaseObjByID(c *gin.Context, db *gorm.DB, modelPtr interface{}, id uint, name, whenMsg string) bool {
	var spacedWhenMsg string
	if whenMsg != "" {
		spacedWhenMsg = " " + whenMsg
	}
	if err := db.First(modelPtr, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ginutil.WriteDBNotFound(c, fmt.Sprintf(
				"%s with ID %d was not found%s.",
				strings.Title(name), id, spacedWhenMsg))
		} else {
			ginutil.WriteDBReadError(c, err, fmt.Sprintf(
				"Failed fetching %s with ID %d from database%s.",
				strings.ToLower(name), id, spacedWhenMsg))
		}
		return false
	}
	return true
}

func optionalLimitOffsetClause(limit, offset int) clause.Expression {
	if limit <= 0 {
		return clause.Limit{}
	}
	if offset <= 0 {
		offset = 0
	}
	return clause.Limit{
		Limit:  limit,
		Offset: offset,
	}
}

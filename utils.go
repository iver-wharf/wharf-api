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

func optionalLimitOffsetScope(limit, offset int) func(*gorm.DB) *gorm.DB {
	if limit <= 0 {
		return gormIdentityScope
	}
	if offset <= 0 {
		offset = 0
	}
	return func(db *gorm.DB) *gorm.DB {
		return db.Clauses(clause.Limit{
			Limit:  limit,
			Offset: offset,
		})
	}
}

func gormIdentityScope(db *gorm.DB) *gorm.DB {
	return db
}

func whereLikeScope(pairs map[string]*string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		b := newGormClauseBuilder(db.Dialector)
		expressions := b.likeExprsFromMap(pairs)
		if len(expressions) == 0 {
			return db
		}
		return db.Clauses(clause.And(expressions...))
	}
}

func whereAnyLikeScope(value *string, keys ...string) func(*gorm.DB) *gorm.DB {
	if value == nil || *value == "" {
		return gormIdentityScope
	}
	return func(db *gorm.DB) *gorm.DB {
		b := newGormClauseBuilder(db.Dialector)
		expressions := b.likeExprsFromSliceSameValue(value, keys...)
		if expressions == nil {
			return db
		}
		return db.Clauses(clause.Or(expressions...))
	}
}

type gormClauseBuilder struct {
	dialect DBDriver
}

func newGormClauseBuilder(dialector gorm.Dialector) gormClauseBuilder {
	return gormClauseBuilder{dialect: DBDriver(dialector.Name())}
}

func (b gormClauseBuilder) likeExprsFromMap(pairs map[string]*string) []clause.Expression {
	if len(pairs) == 0 {
		return nil
	}
	expressions := make([]clause.Expression, 0, len(pairs))
	for key, value := range pairs {
		if expr := b.likeExpr(key, value); expr != nil {
			expressions = append(expressions, expr)
		}
	}
	return expressions
}

func (b gormClauseBuilder) likeExprsFromSliceSameValue(value *string, keys ...string) []clause.Expression {
	if len(keys) == 0 {
		return nil
	}
	if value == nil || *value == "" {
		return nil
	}
	expressions := make([]clause.Expression, 0, len(keys))
	for _, key := range keys {
		if expr := b.likeExpr(key, value); expr != nil {
			expressions = append(expressions, expr)
		}
	}
	return expressions
}

var sqlLikeEscaper = strings.NewReplacer(
	`\`, `\\`,
	`?`, `\?`,
	`_`, `\_`,
	`%`, `\%`,
)

func (b gormClauseBuilder) likeExpr(key string, value *string) clause.Expression {
	if value == nil || *value == "" {
		return nil
	}
	var sqlBuilder strings.Builder
	sqlBuilder.WriteString(key)
	sqlBuilder.WriteByte(' ')
	switch b.dialect {
	case DBDriverPostgres:
		// Case insensitive LIKE
		// https://www.postgresql.org/docs/9.6/functions-matching.html#FUNCTIONS-LIKE
		sqlBuilder.WriteString("ILIKE")
	default:
		// Sqlite is always case-insensitive
		// https://www.sqlite.org/lang_expr.html#like
		sqlBuilder.WriteString("LIKE")
	}
	sqlBuilder.WriteString(` ? ESCAPE '\'`)

	var varBuilder strings.Builder
	varBuilder.WriteByte('%')
	sqlLikeEscaper.WriteString(&varBuilder, *value)
	varBuilder.WriteByte('%')
	return clause.Expr{
		SQL:  sqlBuilder.String(),
		Vars: []interface{}{varBuilder.String()},
	}
}

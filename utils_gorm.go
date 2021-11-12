package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func findDBPaginatedSliceAndTotalCount(dbQuery *gorm.DB, limit, offset int, slicePtr interface{}, totalCount *int64) error {
	err := dbQuery.Scopes(optionalLimitOffsetScope(limit, offset)).Find(slicePtr).Error
	if err != nil {
		return err
	}

	return dbQuery.Count(totalCount).Error
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
	if offset < 0 {
		offset = 0
	}
	return func(db *gorm.DB) *gorm.DB {
		return db.Clauses(clause.Limit{
			Limit:  limit,
			Offset: offset,
		})
	}
}

func optionalTimeRangeScope(column database.SafeSQLName, min, max *time.Time) func(*gorm.DB) *gorm.DB {
	if min == nil && max == nil {
		return gormIdentityScope
	}
	return func(db *gorm.DB) *gorm.DB {
		switch {
		case min == nil:
			return db.Where(string(column)+" < ?", *max)
		case max == nil:
			return db.Where(string(column)+" > ?", *min)
		default:
			return db.Where(string(column)+" BETWEEN ? AND ?", *min, *max)
		}
	}
}

func gormIdentityScope(db *gorm.DB) *gorm.DB {
	return db
}

func whereLikeScope(pairs map[database.SafeSQLName]*string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		b := newGormClauseBuilder(db.Dialector)
		expressions := b.likeExprsFromMap(pairs)
		if len(expressions) == 0 {
			return db
		}
		return db.Clauses(clause.And(expressions...))
	}
}

func whereAnyLikeScope(value *string, keys ...database.SafeSQLName) func(*gorm.DB) *gorm.DB {
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

func (b gormClauseBuilder) likeExprsFromMap(pairs map[database.SafeSQLName]*string) []clause.Expression {
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

func (b gormClauseBuilder) likeExprsFromSliceSameValue(value *string, keys ...database.SafeSQLName) []clause.Expression {
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

func (b gormClauseBuilder) likeExpr(key database.SafeSQLName, value *string) clause.Expression {
	if value == nil || *value == "" {
		return nil
	}
	var sqlString string
	if b.dialect == DBDriverPostgres {
		// ILIKE is the case insensitive LIKE in PostgreSQL
		// https://www.postgresql.org/docs/9.6/functions-matching.html#FUNCTIONS-LIKE
		sqlString = string(key) + ` ILIKE ? ESCAPE '\'`
	} else {
		// Sqlite is always case-insensitive
		// https://www.sqlite.org/lang_expr.html#like
		sqlString = string(key) + ` LIKE ? ESCAPE '\'`
	}
	return clause.Expr{
		SQL:  sqlString,
		Vars: []interface{}{newLikeContainsValue(*value)},
	}
}

// newLikeContainsValue generates an SQL value for a LIKE query, and escapes all
// special LIKE characters such as %, ?, _, and \ itself. Examples:
// 	"foo" // => "%foo%"
// 	"ab%cd" // => "%ab\%cd%"
func newLikeContainsValue(value string) string {
	if value == "" {
		return "%"
	}
	var varBuilder strings.Builder
	varBuilder.WriteByte('%')
	sqlLikeEscaper.WriteString(&varBuilder, value)
	varBuilder.WriteByte('%')
	return varBuilder.String()
}

var sqlLikeEscaper = strings.NewReplacer(
	`\`, `\\`,
	`?`, `\?`,
	`_`, `\_`,
	`%`, `\%`,
)

package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

func TestInsertJoin_postgres(t *testing.T) {
	db, err := gorm.Open(postgres.Open("host=localhost"), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
		NamingStrategy:       schema.NamingStrategy{SingularTable: true},
	})
	require.NoError(t, err)
	// https://stackoverflow.com/a/36039580
	//
	// INSERT INTO log (build_id, message, timestamp)
	// SELECT val.build_id,val.message,val.timestamp
	// FROM (
	//   VALUES
	//     (9, 'hello', CURRENT_TIMESTAMP),
	//     (9, 'there', CURRENT_TIMESTAMP)
	// ) val(build_id, message, timestamp)
	// JOIN build USING (build_id);
	dbLogs := []database.Log{
		{BuildID: 1, Message: "first", Timestamp: time.Now()},
		{BuildID: 1, Message: "second", Timestamp: time.Now()},
		{BuildID: 1, Message: "third", Timestamp: time.Now()},
	}
	var sb strings.Builder
	var params []interface{}
	for _, dbLog := range dbLogs {
		if sb.Len() > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString("\n  (?, ?, ?)")
		params = append(params, dbLog.BuildID, dbLog.Message, dbLog.Timestamp)
	}

	rawSql := db.Raw(fmt.Sprintf(`
INSERT INTO log (build_id, message, timestamp)
SELECT val.build_id, val.message, val.timestamp
FROM (VALUES%s
) val (build_id, message, timestamp)
JOIN build USING (build_id)
`, sb.String()), params...).Statement.SQL.String()
	t.Error(rawSql)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Raw(fmt.Sprintf(`
INSERT INTO log (build_id, message, timestamp)
SELECT val.build_id, val.message, val.timestamp
FROM (VALUES%s
) val (build_id, message, timestamp)
JOIN build USING (build_id)
`, sb.String()), params...)
	})
	t.Error(sql)

}

func TestInsertJoin_sqlite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("somefile.db"), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
		NamingStrategy:       schema.NamingStrategy{SingularTable: true},
	})
	require.NoError(t, err)
	dbLogs := []database.Log{
		{BuildID: 1, Message: "first", Timestamp: time.Now()},
		{BuildID: 1, Message: "second", Timestamp: time.Now()},
		{BuildID: 1, Message: "third", Timestamp: time.Now()},
	}

	rawSql := db.Clauses(clause.Insert{Modifier: "OR IGNORE"}).
		Create(dbLogs).Statement.SQL.String()
	t.Error(rawSql)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Clauses(clause.Insert{Modifier: "OR IGNORE"}).
			Create(dbLogs)
	})
	t.Error(sql)
}

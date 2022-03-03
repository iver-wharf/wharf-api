package main

import (
	"testing"
	"time"

	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var (
	testingTimestamp    = time.Date(2022, 2, 21, 8, 24, 0, 0, time.UTC)
	testingTimestampStr = testingTimestamp.Format(time.RFC3339)
	testingDBLogs       = []database.Log{
		{BuildID: 1, Message: "first", Timestamp: testingTimestamp},
		{BuildID: 1, Message: "second", Timestamp: testingTimestamp},
		{BuildID: 1, Message: "third", Timestamp: testingTimestamp},
	}
)

func TestCreateLogsBatchPostgres(t *testing.T) {
	db, err := gorm.Open(postgres.Open("host=localhost"), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
		NamingStrategy:       schema.NamingStrategy{SingularTable: true},
	})
	require.NoError(t, err)
	query := createLogBatchPostgresQuery(db, testingDBLogs)
	got := query.Statement.SQL.String()
	want := `
INSERT INTO log (build_id, message, timestamp)
SELECT val.build_id, val.message, val.timestamp
FROM (
  VALUES ($1::bigint,$2::text,$3::timestamp with time zone), ($4,$5,$6), ($7,$8,$9)
) AS val (build_id, message, timestamp)
JOIN build USING (build_id)
RETURNING log_id, build_id, message, timestamp
`
	assert.Equal(t, want, got)
}

func TestCreateLogsBatchSqlite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
		NamingStrategy:       schema.NamingStrategy{SingularTable: true},
	})
	require.NoError(t, err)
	query := createLogBatchSqliteQuery(db, testingDBLogs)
	got := query.Statement.SQL.String()
	want := "INSERT OR IGNORE INTO `log` (`build_id`,`message`,`timestamp`) " +
		"VALUES (?,?,?),(?,?,?),(?,?,?) RETURNING `log_id`"
	assert.Equal(t, want, got)
}

func TestCreateLogsBatchSqlite_setsLogIDs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
		Logger:         logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, runDatabaseMigrations(db, DBDriverSqlite))

	dbLogs := make([]database.Log, len(testingDBLogs))
	copy(dbLogs, testingDBLogs)

	require.NoError(t, createLogBatchSqliteQuery(db, dbLogs).Error)

	got := make([]int, len(dbLogs))
	for i, dbLog := range dbLogs {
		got[i] = int(dbLog.LogID)
	}
	want := []int{1, 2, 3}
	assert.Equal(t, want, got)
}

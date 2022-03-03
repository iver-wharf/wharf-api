//go:build postgres
// +build postgres

package main

import (
	"testing"
	"time"

	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Requires a local instance of postgres running. Can be done by running:
//
//   docker run --rm -it --network host -e POSTGRES_PASSWORD=password postgres
//
// Then run these tests with the `postgres` tag:
//
//   go test -tags postgres -run TestCreateLogsBatchPostgres_setsLogIDs .

func TestCreateLogsBatchPostgres_setsLogIDs(t *testing.T) {
	db, err := openDatabasePostgres(DBConfig{
		Name:     "wharf",
		Host:     "localhost",
		Port:     5432,
		Username: "postgres",
		Password: "password",
	})
	require.NoError(t, err, "open database")
	require.NoError(t, runDatabaseMigrations(db, DBDriverSqlite), "run migrations")

	dbBuildID := testingSeedBuild(t, db)
	t.Logf("build ID: %d", dbBuildID)

	dbLogs := []database.Log{
		{BuildID: dbBuildID, Message: "first", Timestamp: testingTimestamp},
		{BuildID: dbBuildID, Message: "second", Timestamp: testingTimestamp},
		{BuildID: dbBuildID, Message: "third", Timestamp: testingTimestamp},
	}

	createdLogs, err := createLogBatchPostgres(db, dbLogs)
	require.NoError(t, err, "create logs")
	assert.Equal(t, 3, len(createdLogs), "created logs len")

	type logWithoutID struct {
		BuildID   uint
		Message   string
		Timestamp string
	}

	gotIDs := make([]int, len(createdLogs))
	gotLogsWithoutIDs := make([]logWithoutID, len(createdLogs))
	for i, dbLog := range createdLogs {
		gotIDs[i] = int(dbLog.LogID)
		gotLogsWithoutIDs[i] = logWithoutID{
			BuildID:   dbLog.BuildID,
			Message:   dbLog.Message,
			Timestamp: dbLog.Timestamp.UTC().Format(time.RFC3339),
		}
	}

	t.Logf("log IDs: %v", gotIDs)
	notWantIDs := []int{0, 0, 0}
	assert.NotEqual(t, notWantIDs, gotIDs)

	wantLogs := []logWithoutID{
		{BuildID: dbBuildID, Message: "first", Timestamp: testingTimestampStr},
		{BuildID: dbBuildID, Message: "second", Timestamp: testingTimestampStr},
		{BuildID: dbBuildID, Message: "third", Timestamp: testingTimestampStr},
	}
	assert.Equal(t, wantLogs, gotLogsWithoutIDs)
}

func testingSeedBuild(t *testing.T, db *gorm.DB) uint {
	var (
		dbProject = database.Project{
			GroupName: "test-group",
			Name:      "test-project",
		}
		dbBuild database.Build
	)
	require.NoError(t, db.Create(&dbProject).Error, "create project")
	require.NotZero(t, dbProject.ProjectID, "project ID")
	dbBuild.ProjectID = dbProject.ProjectID
	require.NoError(t, db.Create(&dbBuild).Error, "create build")
	require.NotZero(t, dbBuild.BuildID, "build ID")
	return dbBuild.BuildID
}

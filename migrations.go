package main

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"gorm.io/gorm"
)

func runDatabaseMigrations(db *gorm.DB, driver DBDriver) error {
	m := newMigrator(db)
	return m.Migrate()
}

var migrationOptions = gormigrate.Options{
	TableName:                 "migration",
	IDColumnName:              "migration_id",
	IDColumnSize:              255,
	UseTransaction:            true,
	ValidateUnknownMigrations: true,
}

var migrations = []*gormigrate.Migration{
	// None yet.
}

// migrateInitSchema is called when no previous migrations were found, while
// also skipping all other migration steps and declaring them as "applied".
//
// This speeds up the initial migration to not require applying all migrations
// one by one on the first run.
func migrateInitSchema(db *gorm.DB) error {
	if err := migrateBeforeGormigrate(db); err != nil {
		return err
	}
	tables := []any{
		&database.Token{}, &database.Provider{},
		&database.Project{}, &database.ProjectOverrides{},
		&database.Branch{}, &database.Build{}, &database.Log{},
		&database.Artifact{}, &database.BuildParam{}, &database.Param{},
		&database.TestResultDetail{}, &database.TestResultSummary{},
	}
	db.DisableForeignKeyConstraintWhenMigrating = true
	if err := db.AutoMigrate(tables...); err != nil {
		return err
	}
	db.DisableForeignKeyConstraintWhenMigrating = false
	return db.AutoMigrate(tables...)
}

func newMigrator(db *gorm.DB) *gormigrate.Gormigrate {
	m := gormigrate.New(db, &migrationOptions, migrations)
	m.InitSchema(migrateInitSchema)
	return m
}

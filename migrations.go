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
	// create persons table
	{
		ID: "201608301400",
		Migrate: func(tx *gorm.DB) error {
			// it's a good pratice to copy the struct inside the function,
			// so side effects are prevented if the original struct changes during the time
			type Person struct {
				gorm.Model
				Name string
			}
			return tx.AutoMigrate(&Person{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable("people")
		},
	},
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
	return db.AutoMigrate(
		&database.Token{}, &database.Provider{},
		&database.Project{}, &database.ProjectOverrides{},
		&database.Branch{}, &database.Build{}, &database.Log{},
		&database.Artifact{}, &database.BuildParam{}, &database.Param{},
		&database.TestResultDetail{}, &database.TestResultSummary{},
	)
}

func newMigrator(db *gorm.DB) *gormigrate.Gormigrate {
	m := gormigrate.New(db, &migrationOptions, migrations)
	m.InitSchema(migrateInitSchema)
	return m
}

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
	tables := []interface{}{
		&database.Token{}, &database.Provider{},
		&database.Project{}, &database.ProjectOverrides{},
		&database.Branch{}, &database.Build{}, &database.Log{},
		&database.Artifact{}, &database.BuildParam{}, &database.Param{},
		&database.TestResultDetail{}, &database.TestResultSummary{},
	}

	migrateConstraints(db, tables)

	// since v5.0.0, all columns that were not nil'able in the GORM models
	// has been migrated to not be nullable in the database either.
	if err := migrateWharfColumnsToNotNull(db); err != nil {
		return err
	}

	oldColumns := []columnToDrop{
		// since v3.1.0, the token.provider_id column was removed as it induced a
		// circular dependency between the token and provider tables
		{&database.Token{}, "provider_id"},
		// Since v5.0.0, the Provider.upload_url column was removed as it was
		// unused.
		{&database.Provider{}, "upload_url"},
	}
	if err := dropOldColumns(db, oldColumns); err != nil {
		return err
	}

	// In v4.2.0 the index param_idx_build_id for artifact was
	// changed to artifact_idx_build_id to match the name of the
	// table.
	oldIndices := []indexToDrop{
		{"artifact", "param_idx_build_id"},
	}
	if err := dropOldIndices(db, oldIndices); err != nil {
		return err
	}

	return db.AutoMigrate(tables...)
}

func newMigrator(db *gorm.DB) *gormigrate.Gormigrate {
	m := gormigrate.New(db, &migrationOptions, migrations)
	m.InitSchema(migrateInitSchema)
	return m
}

package main

import (
	"fmt"

	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"gorm.io/gorm"
)

func runDatabaseMigrations(db *gorm.DB) error {
	tables := []interface{}{
		&database.Token{}, &database.Provider{}, &database.Project{},
		&database.Branch{}, &database.Build{}, &database.Log{},
		&database.Artifact{}, &database.BuildParam{}, &database.Param{},
		&database.TestResultDetail{}, &database.TestResultSummary{}}

	db.DisableForeignKeyConstraintWhenMigrating = true
	if err := db.AutoMigrate(tables...); err != nil {
		return fmt.Errorf("migrating without constraints: %w", err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		// since v4.2.1, drop these constraints to refresh the constraint behavior.
		// Previously it was RESTRICT, now it's CASCADE.
		if err := dropOldConstraints(tx, []constraintToDrop{
			{"artifact", "fk_artifact_build"},
			{"log", "fk_log_build"},
			{"build", "fk_build_project"},
			{"log", "fk_log_build"},
			{"branch", "fk_project_branches"},
			{"build_param", "fk_build_params"},
		}); err != nil {
			return err
		}

		tx.DisableForeignKeyConstraintWhenMigrating = false
		if err := tx.AutoMigrate(tables...); err != nil {
			return fmt.Errorf("migrating with constraints: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	// since v3.1.0, new constraints with other names are added by GORM
	// new constraints are named something like "fk_artifact_build" instead
	oldConstraints := []constraintToDrop{
		{"artifact", "artifact_build_id_build_build_id_foreign"},
		{"branch", "branch_project_id_project_project_id_foreign"},
		{"branch", "branch_token_id_token_token_id_foreign"},
		{"build", "build_project_id_project_project_id_foreign"},
		{"build_param", "build_param_build_id_build_build_id_foreign"},
		{"log", "log_build_id_build_build_id_foreign"},
		{"project", "project_provider_id_provider_provider_id_foreign"},
		{"project", "project_token_id_token_token_id_foreign"},
		{"provider", "provider_token_id_token_token_id_foreign"},
		{"token", "token_provider_id_provider_provider_id_foreign"},
	}
	if err := dropOldConstraints(db, oldConstraints); err != nil {
		return err
	}

	oldColumns := []columnToDrop{
		// since v3.1.0, the token.provider_id column was removed as it induced a
		// circular dependency between the token and provider tables
		{&Token{}, "provider_id"},
		// Since v5.0.0, the Provider.upload_url column was removed as it was
		// unused.
		{&Provider{}, "upload_url"},
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

	return dropOldIndices(db, oldIndices)
}

type constraintToDrop struct {
	table string
	name  string
}

func dropOldConstraints(db *gorm.DB, constraints []constraintToDrop) error {
	log.Debug().WithInt("constraints", len(constraints)).Message("Dropping old constraints.")
	for _, constraint := range constraints {
		if err := dropOldConstraint(db, constraint.table, constraint.name); err != nil {
			return err
		}
	}
	return nil
}

func dropOldConstraint(db *gorm.DB, table string, constraintName string) error {
	if db.Migrator().HasConstraint(table, constraintName) {
		log.Info().
			WithString("table", table).
			WithString("constraint", constraintName).
			Message("Dropping old constraint.")
		if err := db.Migrator().DropConstraint(table, constraintName); err != nil {
			return fmt.Errorf("drop old constraint: %w", err)
		}
	} else {
		log.Debug().
			WithString("table", table).
			WithString("constraint", constraintName).
			Message("No old constraint to remove.")
	}
	return nil
}

type columnToDrop struct {
	model      interface{}
	columnName string
}

func dropOldColumns(db *gorm.DB, columns []columnToDrop) error {
	log.Debug().WithInt("columns", len(columns)).Message("Dropping old columns.")
	for _, dbColumn := range columns {
		if err := dropOldColumn(db, dbColumn.model, dbColumn.columnName); err != nil {
			return err
		}
	}
	return nil
}

func dropOldColumn(db *gorm.DB, model interface{}, columnName string) error {
	if db.Migrator().HasColumn(model, columnName) {
		log.Info().
			WithString("column", columnName).
			WithString("model", fmt.Sprintf("%T", model)).
			Message("Dropping old column.")
		if err := db.Migrator().DropColumn(model, columnName); err != nil {
			return fmt.Errorf("drop old column: %w", err)
		}
	} else {
		log.Debug().
			WithString("column", columnName).
			WithString("model", fmt.Sprintf("%T", model)).
			Message("No old column to remove.")
	}
	return nil
}

type indexToDrop struct {
	table string
	name  string
}

func dropOldIndices(db *gorm.DB, indices []indexToDrop) error {
	log.Debug().WithInt("indices", len(indices)).Message("Dropping old indices.")
	for _, dbIndex := range indices {
		if err := dropOldIndex(db, dbIndex.table, dbIndex.name); err != nil {
			return err
		}
	}
	return nil
}

func dropOldIndex(db *gorm.DB, table string, indexName string) error {
	if db.Migrator().HasIndex(table, indexName) {
		log.Info().
			WithString("table", table).
			WithString("index", indexName).
			Message("Dropping old index.")
		if err := db.Migrator().DropIndex(table, indexName); err != nil {
			return fmt.Errorf("drop old index: %w", err)
		}
	} else {
		log.Debug().
			WithString("table", table).
			WithString("index", indexName).
			Message("No old index to remove.")
	}
	return nil
}

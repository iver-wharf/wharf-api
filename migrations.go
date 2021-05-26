package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func runDatabaseMigrations(db *gorm.DB) error {
	tables := []interface{}{
		&Token{}, &Provider{}, &Project{},
		&Branch{}, &Build{}, &Log{},
		&Artifact{}, &BuildParam{}, &Param{}}

	db.DisableForeignKeyConstraintWhenMigrating = true
	if err := db.AutoMigrate(tables...); err != nil {
		return fmt.Errorf("migrating without constraints: %w", err)
	}

	db.DisableForeignKeyConstraintWhenMigrating = false
	if err := db.AutoMigrate(tables...); err != nil {
		return fmt.Errorf("migrating with constraints: %w", err)
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

	// since v3.1.0, the token.provider_id column was removed as it induced a
	// circular dependency between the token and provider tables
	return dropOldColumn(db, &Token{}, "provider_id")
}

type constraintToDrop struct {
	table string
	name  string
}

func dropOldConstraints(db *gorm.DB, constraints []constraintToDrop) error {
	log.Debugf("Dropping %d old constraints", len(constraints))
	for _, constraint := range constraints {
		if err := dropOldConstraint(db, constraint.table, constraint.name); err != nil {
			return err
		}
	}
	return nil
}

func dropOldConstraint(db *gorm.DB, table string, constraintName string) error {
	if db.Migrator().HasConstraint(table, constraintName) {
		log.Infof("Dropping old constraint for table %q: %q\n", table, constraintName)
		if err := db.Migrator().DropConstraint(table, constraintName); err != nil {
			return fmt.Errorf("drop old constraint: %w", err)
		}
	} else {
		log.Debugf("No old constraint to remove for table %q with name: %q\n", table, constraintName)
	}
	return nil
}

func dropOldColumn(db *gorm.DB, model interface{}, columnName string) error {
	if db.Migrator().HasColumn(model, columnName) {
		log.Infof("Dropping old column for type %T: %q\n", model, columnName)
		if err := db.Migrator().DropColumn(model, columnName); err != nil {
			return fmt.Errorf("drop old column: %w", err)
		}
	} else {
		log.Debugf("No old column to remove for type %T with name: %q\n", model, columnName)
	}
	return nil
}

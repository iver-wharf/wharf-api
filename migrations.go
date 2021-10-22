package main

import (
	"fmt"

	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func runDatabaseMigrations(db *gorm.DB, driver DBDriver) error {
	tables := []interface{}{
		&database.Token{}, &database.Provider{}, &database.Project{},
		&database.Branch{}, &database.Build{}, &database.Log{},
		&database.Artifact{}, &database.BuildParam{}, &database.Param{},
		&database.TestResultDetail{}, &database.TestResultSummary{}}

	db.DisableForeignKeyConstraintWhenMigrating = true
	if err := db.AutoMigrate(tables...); err != nil {
		return fmt.Errorf("migrating without constraints: %w", err)
	}

	if dbDriverSupportsForeignKeyConstraints(driver) {
		migrateConstraints(db, tables)
	} else {
		log.Warn().
			WithString("driver", string(driver)).
			Message("Skipping foreign key constraints, as chosen DB does not support it." +
				" We advice against using this driver for production!")
	}

	if err := migrateWharfColumnsToNotNull(driver, db); err != nil {
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

	return dropOldIndices(db, oldIndices)
}

func migrateConstraints(db *gorm.DB, tables []interface{}) error {
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
	return dropOldConstraints(db, oldConstraints)
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

func migrateWharfColumnsToNotNull(driver DBDriver, db *gorm.DB) error {
	if err := migrateColumnsToNotNull(driver, db, &database.Token{},
		database.TokenFields.UserName,
	); err != nil {
		return fmt.Errorf("migrating columns to not null for project: %w", err)
	}

	if err := migrateColumnsToNotNull(driver, db, &database.Project{},
		database.ProjectFields.GroupName,
		database.ProjectFields.Description,
		database.ProjectFields.AvatarURL,
		database.ProjectFields.GitURL,
		database.ProjectFields.BuildDefinition,
	); err != nil {
		return fmt.Errorf("migrating columns to not null for project: %w", err)
	}

	if err := migrateColumnsToNotNull(driver, db, &database.BuildParam{},
		database.BuildParamFields.Value,
	); err != nil {
		return fmt.Errorf("migrating columns to not null for build_param: %w", err)
	}

	if err := migrateColumnsToNotNull(driver, db, &database.Param{},
		database.ParamFields.Value,
		database.ParamFields.DefaultValue,
	); err != nil {
		return fmt.Errorf("migrating columns to not null for param: %w", err)
	}

	if err := migrateColumnsToNotNull(driver, db, &database.Artifact{},
		database.ArtifactFields.FileName,
	); err != nil {
		return fmt.Errorf("migrating columns to not null for param: %w", err)
	}

	if err := migrateColumnsToNotNull(driver, db, &database.TestResultSummary{},
		database.TestResultSummaryFields.FileName,
	); err != nil {
		return fmt.Errorf("migrating columns to not null for test_result_summary: %w", err)
	}

	return nil
}

func migrateColumnsToNotNull(driver DBDriver, db *gorm.DB, model interface{}, fieldNames ...string) error {
	if !db.Migrator().HasTable(model) {
		log.Debug().WithStringf("model", "%T", model).Message("Skipping changing column to not null as the table does not exist.")
		return nil
	}
	actualTypes, err := db.Migrator().ColumnTypes(model)
	if err != nil {
		return err
	}
	actualTypesPerDBName := map[string]gorm.ColumnType{}
	for _, columnType := range actualTypes {
		actualTypesPerDBName[columnType.Name()] = columnType
	}

	stmt := db.Model(model).Statement
	if err := stmt.Parse(model); err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, fieldName := range fieldNames {
			wantedField := stmt.Schema.LookUpField(fieldName)
			if wantedField == nil {
				return fmt.Errorf("unknown field when changing to not null: %q", fieldName)
			}
			actualType, ok := actualTypesPerDBName[wantedField.DBName]
			if !ok {
				log.Warn().
					WithStringf("model", "%T", model).
					WithString("field", fieldName).
					Message("Cannot change column to not null as the column does not exist in the table.")
				continue
			}
			if err := migrateColumnToNotNull(driver, tx, model, wantedField, actualType); err != nil {
				return err
			}
		}
		return nil
	})
}

func migrateColumnToNotNull(driver DBDriver, db *gorm.DB, model interface{}, want *schema.Field, actual gorm.ColumnType) error {
	if want.PrimaryKey {
		return fmt.Errorf("cannot operate changing the column to not null for a primary key: %q", want.Name)
	}
	if !want.NotNull {
		return fmt.Errorf("struct field is not declared to be not null: %q", want.Name)
	}
	if nullable, ok := actual.Nullable(); !ok || !nullable {
		// Already not null
		return nil
	}

	defaultValue, err := sqlDefaultValue(want.DataType)
	if err != nil {
		return fmt.Errorf("get default value: %w", err)
	}

	defaultValueSQLString, err := sqlDefaultValueSQLString(want.DataType)
	if err != nil {
		return fmt.Errorf("get default value as SQL string: %w", err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(model).Where(want.DBName, nil).Update(want.DBName, defaultValue).Error; err != nil {
			return fmt.Errorf("set all to zero where %q is null: %w", want.DBName, err)
		}

		switch driver {
		case DBDriverPostgres:
			tableName := want.Schema.Table
			if err := tx.
				Exec(
					fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" SET DEFAULT %s`,
						tableName, want.DBName, defaultValueSQLString),
				).Error; err != nil {
				return fmt.Errorf("set default to %q: %w", defaultValueSQLString, err)
			}
			if err := tx.
				Exec(
					fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" SET NOT NULL`,
						tableName, want.DBName),
				).Error; err != nil {
				return fmt.Errorf("set not null: %w", err)
			}
			return nil
		case DBDriverSqlite:
			// Sqlite migrator recreates the table with the field updated
			return db.Migrator().AlterColumn(model, want.Name)
		default:
			return fmt.Errorf("migrating column to not null has not been implemented for this DB driver: %q", driver)
		}
	}); err != nil {
		return err
	}
	log.Info().
		WithStringf("model", "%T", model).
		WithString("field", want.Name).
		WithString("column", want.DBName).
		Message("Changed column to not null.")
	return nil
}

func sqlDefaultValue(dataType schema.DataType) (interface{}, error) {
	switch dataType {
	case schema.Bool:
		return false, nil
	case schema.Int:
		return int(0), nil
	case schema.Uint:
		return uint(0), nil
	case schema.Float:
		return float32(0), nil
	case schema.String:
		return "", nil
	default:
		return nil, fmt.Errorf("unsupported data type: %q", dataType)
	}
}

func sqlDefaultValueSQLString(dataType schema.DataType) (string, error) {
	switch dataType {
	case schema.Bool:
		return "false", nil
	case schema.Int, schema.Uint, schema.Float:
		return "0", nil
	case schema.String:
		return "''", nil
	default:
		return "", fmt.Errorf("unsupported data type: %q", dataType)
	}
}

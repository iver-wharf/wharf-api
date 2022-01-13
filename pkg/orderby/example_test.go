package orderby_test

import (
	"fmt"

	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/iver-wharf/wharf-api/v5/pkg/orderby"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Project struct {
	ProjectID uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:500;not null"`
	GroupName string `gorm:"size:500"`
}

func ExampleColumn_String() {
	fmt.Printf("1: %q\n", orderby.Column{"build_id", orderby.Asc})
	fmt.Printf("2: %q\n", orderby.Column{"build_id", orderby.Desc})
	// Output:
	// 1: "build_id asc"
	// 2: "build_id desc"
}

func ExampleColumn_Clause() {
	db := dryRunDB()
	var projects []Project

	orderBy := orderby.Column{"group_name", orderby.Asc}
	printStmt(db.Model(&Project{}).Clauses(orderBy.Clause()).Find(&projects))

	// Output:
	// SELECT * FROM "projects" ORDER BY "group_name"
}

func ExampleParseDirection() {
	for _, str := range []string{"asc", "desc", "foo"} {
		if d, err := orderby.ParseDirection(str); err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Println("Direction:", d)
		}
	}
	// Output:
	// Direction: asc
	// Direction: desc
	// Error: "foo": invalid direction, only 'asc' or 'desc' supported
}

func ExampleParse() {
	fieldToColumnNames := map[string]database.SafeSQLName{
		"buildId": "build_id",
	}
	fields := []string{
		"buildId asc",
		"  buildId   desc  ",
		"foobar asc",
		"buildId foo",
	}
	for _, field := range fields {
		if order, err := orderby.Parse(field, fieldToColumnNames); err != nil {
			fmt.Printf("Invalid sort order: %v\n", err)
		} else {
			fmt.Printf("Sort by %q\n", order)
		}
	}
	// Output:
	// Sort by "build_id asc"
	// Sort by "build_id desc"
	// Invalid sort order: failed mapping field name to column name: "foobar": invalid or unsupported ordering field
	// Invalid sort order: failed parsing ordering direction: "foo": invalid direction, only 'asc' or 'desc' supported
}

func ExampleSlice_Clause() {
	db := dryRunDB()
	var projects []Project

	orderBySlice := orderby.Slice{{"group_name", orderby.Asc}, {"name", orderby.Desc}}
	multiOrderByQuery := db.Model(&Project{}).Clauses(orderBySlice.Clause())
	printStmt(multiOrderByQuery.Find(&projects))

	// Output:
	// SELECT * FROM "projects" ORDER BY "group_name","name" DESC
}

func ExampleSlice_ClauseIfNone() {
	db := dryRunDB()
	var projects []Project

	fallbackOrderBy := orderby.Column{"project_id", orderby.Asc}
	orderBySlice := orderby.Slice{} // intentionally empty
	fallbackQuery := db.Model(&Project{}).Clauses(orderBySlice.ClauseIfNone(fallbackOrderBy))
	printStmt(fallbackQuery.Find(&projects))

	// Output:
	// SELECT * FROM "projects" ORDER BY "project_id"
}

func printStmt(tx *gorm.DB) {
	fmt.Println(tx.Statement.SQL.String())
}

func dryRunDB() *gorm.DB {
	db, err := gorm.Open(postgres.New(postgres.Config{}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		panic(fmt.Sprintf("error opening DB: %v", err))
	}
	return db
}

package orderby

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ExampleOrderBy_String() {
	fmt.Printf("1: %q\n", OrderBy{"build_id", Asc})
	fmt.Printf("2: %q\n", OrderBy{"build_id", Desc})
	// Output:
	// 1: "build_id asc"
	// 2: "build_id desc"
}

func ExampleParseDirection() {
	for _, str := range []string{"asc", "desc", "foo"} {
		if d, err := ParseDirection(str); err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Println("Direction:", d)
		}
	}
	// Output:
	// Direction: asc
	// Direction: desc
	// Error: invalid direction, only 'asc' or 'desc' supported, but got: "foo"
}

func ExampleParse() {
	fieldToColumnNames := map[string]string{
		"buildId": "build_id",
	}
	fields := []string{
		"buildId asc",
		"  buildId   desc  ",
		"foobar asc",
		"buildId foo",
	}
	for _, field := range fields {
		if order, err := Parse(field, fieldToColumnNames); err != nil {
			fmt.Printf("Invalid sort order: %v\n", err)
		} else {
			fmt.Printf("Sort by %q\n", order)
		}
	}
	// Output:
	// Sort by "build_id asc"
	// Sort by "build_id desc"
	// Invalid sort order: failed mapping field name to column name: invalid or unsupported ordering field: "foobar"
	// Invalid sort order: failed parsing ordering direction: invalid direction, only 'asc' or 'desc' supported, but got: "foo"
}

func ExampleParse_withoutNameMapping() {
	fields := []string{
		"buildId asc",
		"  buildId   desc  ",
		"foobar asc",
		"buildId foo",
	}
	for _, field := range fields {
		// leave second argument as nil
		if order, err := Parse(field, nil); err != nil {
			fmt.Printf("Invalid sort order: %v\n", err)
		} else {
			fmt.Printf("Sort by %q\n", order)
		}
	}
	// Output:
	// Sort by "buildId asc"
	// Sort by "buildId desc"
	// Sort by "foobar asc"
	// Invalid sort order: failed parsing ordering direction: invalid direction, only 'asc' or 'desc' supported, but got: "foo"
}

func ExampleApplyAllToGormQuery() {
	db := dryRunDB()
	type Project struct {
		ProjectID uint   `gorm:"primaryKey"`
		Name      string `gorm:"size:500;not null"`
		GroupName string `gorm:"size:500"`
	}
	var projects []Project

	fmt.Println(db.Model(&Project{}).Find(&projects).Statement.SQL.String())
	// Result: SELECT * FROM "projects"

	orderBySlice := []OrderBy{{"group_name", Asc}, {"name", Desc}}
	multiOrderByQuery := ApplyAllToGormQuery(db.Model(&Project{}), orderBySlice, OrderBy{})
	fmt.Println(multiOrderByQuery.Find(&projects).Statement.SQL.String())
	// Result: SELECT * FROM "projects" ORDER BY group_name asc,name desc

	fallbackOrderBy := OrderBy{"project_id", Asc}
	fallbackQuery := ApplyAllToGormQuery(db.Model(&Project{}), []OrderBy{}, fallbackOrderBy)
	fmt.Println(fallbackQuery.Find(&projects).Statement.SQL.String())
	// Result: SELECT * FROM "projects" ORDER BY project_id asc

	// Output:
	// SELECT * FROM "projects"
	// SELECT * FROM "projects" ORDER BY group_name asc,name desc
	// SELECT * FROM "projects" ORDER BY project_id asc
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

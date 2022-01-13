package wherefields_test

import (
	"fmt"

	"github.com/iver-wharf/wharf-api/v5/internal/wherefields"
	"gorm.io/gorm"
)

func ExampleCollection() {
	db := &gorm.DB{} // placeholder GORM DB reference

	type Project struct {
		ProjectID uint
		Name      string
		GroupName string
	}

	params := struct {
		ProjectID *uint
		Name      *string
		GroupName *string
	}{}

	var where wherefields.Collection
	var dbProjects []Project
	err := db.
		Where(&Project{
			ProjectID: where.Uint("ProjectID", params.ProjectID),
			Name:      where.String("Name", params.Name),
			GroupName: where.String("GroupName", params.GroupName),
		}, where.NonNilFieldNames()...).
		Find(&dbProjects).
		Error

	if err != nil {
		fmt.Println("Err:", err)
		return
	}

	fmt.Println("Found projects:", dbProjects)
}

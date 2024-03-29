package orderby

import (
	"errors"
	"fmt"
	"io"

	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"gorm.io/gorm/clause"
)

var (
	// ErrEmptyString is returned when parsing an orderby column and the string
	// was empty.
	ErrEmptyString = errors.New("empty ordering string")
	// ErrMissingDirection is returned when parsing an orderby column but the
	// direction was not found in the string.
	// 	"myfield desc" // OK
	// 	"myfield"      // err!
	ErrMissingDirection = errors.New("missing ordering direction, 'asc' or 'desc'")
	// ErrInvalidField is returned when parsing an orderby column against a map
	// of accepted values, and none matched.
	ErrInvalidField = errors.New("invalid or unsupported ordering field")
	// ErrNilParseMap is returned when parsing but the map that was passed was
	// nil.
	ErrNilParseMap = errors.New("field->column names map is nil")
)

// Column specifies a column or field to be sorted and its sorting direction.
type Column struct {
	Name      database.SafeSQLName
	Direction Direction
}

// String converts an ordering to a string representation. The result is meant
// to be parsable by the orderby.Parse function.
func (o Column) String() string {
	return fmt.Sprintf("%s %s", o.Name, o.Direction)
}

func (o Column) clauseOrderByColumn() clause.OrderByColumn {
	return clause.OrderByColumn{
		Column: clause.Column{
			Name: string(o.Name),
		},
		Desc: o.Direction == Desc,
	}
}

// Clause returns a GORM clause expression to apply the column ordering to the
// query. Meant to be used on the gorm.DB.Clauses function.
func (o Column) Clause() clause.Expression {
	return clause.OrderBy{
		Columns: []clause.OrderByColumn{
			o.clauseOrderByColumn(),
		},
	}
}

// Parse interprets an ordering/sorting definition and optionally translates the
// inputted field name using a map.
func Parse(query string, fieldToColumnNames map[string]database.SafeSQLName) (Column, error) {
	field, direction, err := scanQueryForOrderBy(query)
	if err != nil {
		return Column{}, fmt.Errorf("failed scanning orderby string: %w", err)
	}
	return parseFromFieldAndDirection(field, direction, fieldToColumnNames)
}

func scanQueryForOrderBy(query string) (field, direction string, err error) {
	var valuesRead int
	valuesRead, err = fmt.Sscanf(query, "%s %s", &field, &direction)
	if err == io.EOF {
		if valuesRead == 0 {
			err = ErrEmptyString
		} else if valuesRead == 1 {
			err = ErrMissingDirection
		}
	}
	return
}

func parseFromFieldAndDirection(field, directionStr string, fieldToColumnNames map[string]database.SafeSQLName) (Column, error) {
	column, err := mapOrderByField(field, fieldToColumnNames)
	if err != nil {
		return Column{}, fmt.Errorf("failed mapping field name to column name: %w", err)
	}

	direction, err := ParseDirection(directionStr)
	if err != nil {
		return Column{}, fmt.Errorf("failed parsing ordering direction: %w", err)
	}

	return Column{Name: column, Direction: direction}, nil
}

func mapOrderByField(field string, fieldToColumnNames map[string]database.SafeSQLName) (database.SafeSQLName, error) {
	if fieldToColumnNames == nil {
		return "", ErrNilParseMap
	}
	column, ok := fieldToColumnNames[field]
	if !ok {
		return "", fmt.Errorf("%q: %w", field, ErrInvalidField)
	}
	return column, nil
}

package orderby

import (
	"errors"
	"fmt"
	"io"

	"gorm.io/gorm"
)

type OrderBy struct {
	Column    string
	Direction Direction
}

// String converts an OrderBy to a string with the column name, followed by 'asc'
// if OrderBy.Ascending is true or 'desc' otherwise, separated by a space.
func (o OrderBy) String() string {
	return fmt.Sprintf("%s %s", o.Column, o.Direction)
}

// Parse interprets a ordering/sorting definition and optionally translates the
// inputted field name using a map.
func Parse(query string, fieldToColumnNames map[string]string) (OrderBy, error) {
	field, direction, err := scanQueryForOrderBy(query)
	if err != nil {
		return OrderBy{}, fmt.Errorf("failed scanning orderby string: %w", err)
	}
	return parseFromFieldAndDirection(field, direction, fieldToColumnNames)
}

func scanQueryForOrderBy(query string) (field, direction string, err error) {
	var valuesRead int
	valuesRead, err = fmt.Sscanf(query, "%s %s", &field, &direction)
	if err == io.EOF {
		if valuesRead == 0 {
			err = errors.New("empty ordering string")
		} else if valuesRead == 1 {
			err = errors.New("missing ordering direction, 'asc' or 'desc'")
		}
	}
	return
}

func parseFromFieldAndDirection(field, directionStr string, fieldToColumnNames map[string]string) (OrderBy, error) {
	column, err := mapOrderByField(field, fieldToColumnNames)
	if err != nil {
		return OrderBy{}, fmt.Errorf("failed mapping field name to column name: %w", err)
	}

	direction, err := ParseDirection(directionStr)
	if err != nil {
		return OrderBy{}, fmt.Errorf("failed parsing ordering direction: %w", err)
	}

	return OrderBy{Column: column, Direction: direction}, nil
}

func mapOrderByField(field string, fieldToColumnNames map[string]string) (string, error) {
	if fieldToColumnNames == nil {
		return field, nil
	}
	column, ok := fieldToColumnNames[field]
	if !ok {
		return "", fmt.Errorf("invalid or unsupported ordering field: %q", field)
	}
	return column, nil
}

// ParseSlice returns a new slice where each element has been interpreted by the
// Parse function, or the error of the first failed parsing.
func ParseSlice(queries []string, fieldToColumnNames map[string]string) ([]OrderBy, error) {
	sqlOrderings := make([]OrderBy, len(queries))
	for i, qo := range queries {
		if orderBy, err := Parse(qo, fieldToColumnNames); err != nil {
			return nil, err
		} else {
			sqlOrderings[i] = orderBy
		}
	}
	return sqlOrderings, nil
}

// ApplyAllToGormQuery adds each OrderBy to the gorm.DB query in order, or if
// the orderings slice is empty then the ifNone ordering is applied.
func ApplyAllToGormQuery(query *gorm.DB, orderBySlice []OrderBy, ifNone OrderBy) *gorm.DB {
	if len(orderBySlice) == 0 {
		query = query.Order(ifNone.String())
	} else {
		for _, o := range orderBySlice {
			query = query.Order(o.String())
		}
	}
	return query
}

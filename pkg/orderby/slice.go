package orderby

import (
	"strings"

	"gorm.io/gorm/clause"
)

// Slice is a Go slice of column orderings, meant to represent multiple
// orderings to apply in order.
type Slice []Column

// String converts an OrderBy to a string with the column name, followed by 'asc'
// if OrderBy.Ascending is true or 'desc' otherwise, separated by a space.
func (slice Slice) String() string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, o := range slice {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(o.String())
	}
	sb.WriteByte(']')
	return sb.String()
}

// Clause returns a GORM clause expression to apply the list of column orderings
// to the query. Meant to be used on the gorm.DB.Clauses function.
func (slice Slice) Clause() clause.Expression {
	clauseOrderBy := clause.OrderBy{
		Columns: make([]clause.OrderByColumn, len(slice)),
	}
	for i, o := range slice {
		clauseOrderBy.Columns[i] = o.clauseOrderByColumn()
	}
	return clauseOrderBy
}

// ClauseIfNone returns a GORM clause expression to apply the list of column
// orderings to the query, or a fallback ordering if the list is empty. Meant to
// be used on the gorm.DB.Clauses function.
func (slice Slice) ClauseIfNone(ifNone Column) clause.Expression {
	if len(slice) == 0 {
		return ifNone.Clause()
	}
	return slice.Clause()
}

// ParseSlice returns a new slice where each element has been interpreted by the
// Parse function, or the error of the first failed parsing.
func ParseSlice(queries []string, fieldToColumnNames map[string]string) (Slice, error) {
	sqlOrderings := make([]Column, len(queries))
	for i, qo := range queries {
		orderBy, err := Parse(qo, fieldToColumnNames)
		if err != nil {
			return nil, err
		}
		sqlOrderings[i] = orderBy
	}
	return sqlOrderings, nil
}

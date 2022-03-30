package wherefields

import "gopkg.in/guregu/null.v4"

// Collection stores field names of non-nil values. Meant to be used with the
// GORM .Where() clause.
type Collection struct {
	fieldNames []any
}

// NonNilFieldNames returns the slice of field names. Meant to be used as the
// trailing arguments to the GORM .Where() clause.
func (sc *Collection) NonNilFieldNames() []any {
	return sc.fieldNames
}

// AddFieldName adds a string that will be returned later by NonNilFieldNames.
func (sc *Collection) AddFieldName(field string) {
	sc.fieldNames = append(sc.fieldNames, field)
}

// Int stores the field name if the value was non-nil and returns the value of
// the field, or zero (0) if it was nil.
func (sc *Collection) Int(field string, value *int) int {
	if value == nil {
		return 0
	}
	sc.AddFieldName(field)
	return *value
}

// Uint stores the field name if the value was non-nil and returns the value of
// the field, or zero (0) if it was nil.
func (sc *Collection) Uint(field string, value *uint) uint {
	if value == nil {
		return 0
	}
	sc.AddFieldName(field)
	return *value
}

// UintPtrZeroNil stores the field name if the value was non-nil and returns the
// value of the field and translates zero (0) to nil.
func (sc *Collection) UintPtrZeroNil(field string, value *uint) *uint {
	if value == nil {
		return nil
	}
	sc.AddFieldName(field)
	if *value == 0 {
		return nil
	}
	return value
}

// String stores the field name if the value was non-nil and returns the value
// of the field, or empty string ("") if it was nil.
func (sc *Collection) String(field string, value *string) string {
	if value == nil {
		return ""
	}
	sc.AddFieldName(field)
	return *value
}

// NullStringEmptyNull stores the field name if the value was non-nil and
// returns the value of the field and translates empty ("") to null.
func (sc *Collection) NullStringEmptyNull(field string, value *string) null.String {
	if value == nil {
		return null.String{}
	}
	sc.AddFieldName(field)
	if *value == "" {
		return null.String{}
	}
	return null.StringFrom(*value)
}

// Bool stores the field name if the value was non-nil and returns the value
// of the field, or false if it was nil.
func (sc *Collection) Bool(field string, value *bool) bool {
	if value == nil {
		return false
	}
	sc.AddFieldName(field)
	return *value
}

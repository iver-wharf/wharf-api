package wherefields

// Collection stores field names of non-nil values. Meant to be used with the
// GORM .Where() clause.
type Collection struct {
	fieldNames []interface{}
}

// NonNilFieldNames returns the slice of field names. Meant to be used as the
// trailing arguments to the GORM .Where() clause.
func (sc *Collection) NonNilFieldNames() []interface{} {
	return sc.fieldNames
}

func (sc *Collection) addFieldName(field string) {
	sc.fieldNames = append(sc.fieldNames, field)
}

// Uint stores the field name if the value was non-nil and returns the value of
// the field, or zero (0) if it was nil.
func (sc *Collection) Uint(field string, value *uint) uint {
	if value == nil {
		return 0
	}
	sc.addFieldName(field)
	return *value
}

// String stores the field name if the value was non-nil and returns the value
// of the field, or empty string ("") if it was nil.
func (sc *Collection) String(field string, value *string) string {
	if value == nil {
		return ""
	}
	sc.addFieldName(field)
	return *value
}

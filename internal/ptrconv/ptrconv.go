package ptrconv

// UintPtr converts an uint pointer to a uint value, where nil is
// translated to zero (0).
func UintPtr(value *uint) uint {
	if value == nil {
		return 0
	}
	return *value
}

// UintZeroNil converts an uint value to a uint pointer, where zero (0) is
// translated to nil.
func UintZeroNil(value uint) *uint {
	if value == 0 {
		return nil
	}
	return &value
}

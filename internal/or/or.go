// Package or helps with a simple use case: defining fallback values, similar
// to the null coalescing operator.
package or

// String returns the first non-empty string from the set of variadic parameters.
func String(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

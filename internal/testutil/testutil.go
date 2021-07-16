package testutil

import (
	"os"
	"testing"
)

// Setenv sets the environment variable, and then unsets it when the test has
// finished.
func Setenv(t *testing.T, key, value string) {
	os.Setenv(key, value)
	t.Cleanup(func() { os.Unsetenv(key) })
}

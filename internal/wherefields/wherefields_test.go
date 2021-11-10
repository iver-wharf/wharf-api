package wherefields

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollection(t *testing.T) {
	var (
		where Collection

		myString    = "foobar"
		emptyString string
		myUint      uint = 123
		zeroUint    uint

		want = []string{
			"MyString",
			"EmptyString",
			"MyUint",
			"ZeroUint",
		}
	)

	where.String("MyString", &myString)
	where.String("EmptyString", &emptyString)
	where.Uint("MyUint", &myUint)
	where.Uint("ZeroUint", &zeroUint)

	where.String("NilString", nil)
	where.Uint("NilUint", nil)

	got := where.NonNilFieldNames()
	assert.ElementsMatch(t, want, got)
}

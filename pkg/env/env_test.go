package env

import (
	"testing"
	"time"

	"github.com/iver-wharf/wharf-api/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBind(t *testing.T) {
	var (
		myString   string
		myInt      int
		myUInt64   uint64
		myBool     bool
		myDuration time.Duration
	)

	testutil.Setenv(t, "MY_STR", "bar")
	testutil.Setenv(t, "MY_BOOL", "true")
	testutil.Setenv(t, "MY_INT", "-125")
	testutil.Setenv(t, "MY_UINT64", "255")
	testutil.Setenv(t, "MY_DURATION", "15s")

	Bind(&myString, "MY_STR")
	require.NoError(t, BindBool(&myBool, "MY_BOOL"))
	require.NoError(t, BindInt(&myInt, "MY_INT"))
	require.NoError(t, BindUInt64(&myUInt64, "MY_UINT64"))
	require.NoError(t, BindDuration(&myDuration, "MY_DURATION"))

	assert.Equal(t, "bar", myString)
	assert.Equal(t, true, myBool)
	assert.Equal(t, int(-125), myInt)
	assert.Equal(t, uint64(255), myUInt64)
	assert.Equal(t, 15*time.Second, myDuration)
}

func TestBindMultiple(t *testing.T) {
	var myStruct struct {
		a string
		b string
		c string
	}

	testutil.Setenv(t, "MYSTRUCT_A", "one")
	testutil.Setenv(t, "MYSTRUCT_B", "two")
	testutil.Setenv(t, "MYSTRUCT_C", "tre")

	BindMultiple(map[*string]string{
		&myStruct.a: "MYSTRUCT_A",
		&myStruct.b: "MYSTRUCT_B",
		&myStruct.c: "MYSTRUCT_C",
	})

	assert.Equal(t, "one", myStruct.a)
	assert.Equal(t, "two", myStruct.b)
	assert.Equal(t, "tre", myStruct.c)
}

func TestBindMultiple_noErrorOnNilMap(t *testing.T) {
	assert.NoError(t, BindMultiple(nil))
	assert.NoError(t, BindMultipleInt(nil))
}

package env

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Bind updates the Go variable via the pointer with the value of the
// environment variable, if set and not empty.
//
// An error is returned if it failed to get the environment variable.
func Bind(ptr *string, key string) error {
	if value, ok := os.LookupEnv(key); ok {
		*ptr = value
	}
	return nil // can't fail, but just for consitency with the other funcs
}

// BindMultiple updates the Go variables via the pointers with the values of the
// environment variables, if set and not empty, for each respective pair in
// the map.
//
// An error is returned if it failed to get any of the environment variables for
// the mappings.
func BindMultiple(mappings map[*string]string) error {
	for ptr, key := range mappings {
		if err := Bind(ptr, key); err != nil {
			return err
		}
	}
	return nil
}

// LookupNoEmpty retrieves the value of the environment variable but returns
// false if the environment variable was not set or empty.
func LookupNoEmpty(key string) (string, bool) {
	var str, ok = os.LookupEnv(key)
	if str == "" {
		return "", false
	}
	return str, ok
}

// BindNoEmpty updates the Go variable via the pointer with the value of the
// environment variable, if set and not empty.
func BindNoEmpty(ptr *string, key string) error {
	if str, ok := LookupNoEmpty(key); ok {
		*ptr = str
	}
	return nil // can't fail, but just for consitency with the other funcs
}

// LookupOptionalBool retrieves the environment variable value, parsed as a
// bool, or returns the fallback value if the environment variable is unset or
// empty.
//
// An error is returned if it failed to parse.
func LookupOptionalBool(name string, fallback bool) (bool, error) {
	if envStr, ok := LookupNoEmpty(name); ok {
		envBool, err := strconv.ParseBool(envStr)
		if err != nil {
			return false, fmt.Errorf("env: %q: unable to parse bool: %q", name, envStr)
		}
		return envBool, nil
	}
	return fallback, nil
}

// BindBool updates the Go variable via the pointer with the value of the
// environment variable, parsed as a bool, if set and not empty.
//
// An error is returned if it failed to parse.
func BindBool(ptr *bool, name string) (err error) {
	*ptr, err = LookupOptionalBool(name, *ptr)
	return
}

// LookupOptionalUInt64 retrieves the environment variable value, parsed as an
// uin64, or returns the fallback value if the environment variable is unset or
// empty.
//
// An error is returned if it failed to parse.
func LookupOptionalUInt64(name string, fallback uint64) (uint64, error) {
	if envStr, ok := LookupNoEmpty(name); ok {
		envInt, err := strconv.ParseUint(envStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("env: %q: unable to parse uint64: %q", name, envStr)
		}
		return envInt, nil
	}
	return fallback, nil
}

// BindUInt64 updates the Go variable via the pointer with the value of the
// environment variable, parsed as an uint64, if set and not empty.
//
// An error is returned if it failed to parse.
func BindUInt64(ptr *uint64, name string) (err error) {
	*ptr, err = LookupOptionalUInt64(name, *ptr)
	return
}

// LookupOptionalInt retrieves the environment variable value, parsed as an
// int, or returns the fallback value if the environment variable is unset or
// empty.
//
// An error is returned if it failed to parse.
func LookupOptionalInt(name string, fallback int) (int, error) {
	if envStr, ok := LookupNoEmpty(name); ok {
		envInt, err := strconv.ParseInt(envStr, 10, strconv.IntSize)
		if err != nil {
			return 0, fmt.Errorf("env: %q: unable to parse int: %q", name, envStr)
		}
		return int(envInt), nil
	}
	return fallback, nil
}

// BindInt updates the Go variable via the pointer with the value of the
// environment variable, parsed as an int, if set and not empty.
//
// An error is returned if it failed to parse.
func BindInt(ptr *int, name string) (err error) {
	*ptr, err = LookupOptionalInt(name, *ptr)
	return
}

// BindMultipleInt updates the Go variables via the pointers with the values of
// the environment variables, parsed as ints, if set and not empty, for each
// respective pair in the map.
//
// An error is returned if it failed to parse any of the mappings.
func BindMultipleInt(mappings map[*int]string) error {
	for ptr, key := range mappings {
		if err := BindInt(ptr, key); err != nil {
			return err
		}
	}
	return nil
}

// LookupOptionalDuration retrieves the environment variable value, parsed as an
// time.Duration, or returns the fallback value if the environment variable is
// unset or empty.
//
// An error is returned if it failed to parse.
func LookupOptionalDuration(name string, fallback time.Duration) (time.Duration, error) {
	if envStr, ok := LookupNoEmpty(name); ok {
		envDuration, err := time.ParseDuration(envStr)
		if err != nil {
			return 0, fmt.Errorf("env: %q: unable to parse duration: %q", name, envStr)
		}
		return envDuration, nil
	}
	return fallback, nil
}

// BindDuration updates the Go variable via the pointer with the value of the
// environment variable, parsed as time.Duration, if set and not empty.
//
// An error is returned if it failed to parse.
func BindDuration(ptr *time.Duration, name string) (err error) {
	*ptr, err = LookupOptionalDuration(name, *ptr)
	return
}

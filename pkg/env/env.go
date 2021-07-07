package env

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Bind updates the Go variable via the pointer with the value of the
// environment variable, if set.
func Bind(ptr *string, key string) {
	if value, ok := os.LookupEnv(key); ok {
		*ptr = value
	}
}

// BindMultiple updates the Go variables via the pointers with the values of the
// environment variables, if set, for each respective pair in the map.
func BindMultiple(mappings map[*string]string) {
	for ptr, key := range mappings {
		Bind(ptr, key)
	}
}

// LookupNoEmpty retrieves the value of the environment variable but returns
// false if the environment variable was empty or not set.
func LookupNoEmpty(key string) (string, bool) {
	var str, ok = os.LookupEnv(key)
	if str == "" {
		return "", false
	}
	return str, ok
}

// BindNoEmpty updates the Go variable via the pointer with the value of the
// environment variable, if set and not empty.
func BindNoEmpty(ptr *string, key string) bool {
	if str, ok := LookupNoEmpty(key); ok {
		*ptr = str
		return true
	}
	return false
}

// LookupOptionalBool retrieves the environment variable value, parsed as a
// bool, or returns the fallback value if the environment variable is unset.
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

func BindBool(ptr *bool, name string) (err error) {
	*ptr, err = LookupOptionalBool(name, *ptr)
	return
}

// LookupOptionalUInt64 retrieves the environment variable value, parsed as an
// uin64, or returns the fallback value if the environment variable is unset.
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

func BindUInt64(ptr *uint64, name string) (err error) {
	*ptr, err = LookupOptionalUInt64(name, *ptr)
	return
}

// LookupOptionalInt retrieves the environment variable value, parsed as an
// int, or returns the fallback value if the environment variable is unset.
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

func BindInt(ptr *int, name string) (err error) {
	*ptr, err = LookupOptionalInt(name, *ptr)
	return
}

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
// unset.
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

func BindDuration(ptr *time.Duration, name string) (err error) {
	*ptr, err = LookupOptionalDuration(name, *ptr)
	return
}

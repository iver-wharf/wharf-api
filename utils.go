package main

import (
	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
)

func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func findDefaultBranch(branches []database.Branch) (database.Branch, bool) {
	for _, branch := range branches {
		if branch.Default {
			return branch, true
		}
	}

	return database.Branch{}, false
}

func stringSliceToInterfaces(values []string) []any {
	newSlice := make([]any, len(values))
	for i, v := range values {
		newSlice[i] = v
	}
	return newSlice
}

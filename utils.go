package main

import (
	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
)

func findDefaultBranch(branches []database.Branch) (database.Branch, bool) {
	for _, branch := range branches {
		if branch.Default {
			return branch, true
		}
	}
	return database.Branch{}, false
}

func asAnySlice[S ~[]E, E any](values S) []any {
	newSlice := make([]any, len(values))
	for i, v := range values {
		newSlice[i] = v
	}
	return newSlice
}

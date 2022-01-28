package modelconv

import "github.com/iver-wharf/wharf-api/v5/pkg/model/response"

// EngineLookup is a callback for finding the engine response based on its ID.
// It is expected to return nil if no engine was found by that ID.
//
// The callback should not return any fallback or default values. The match is
// expected to be an exact match based on ID.
type EngineLookup func(id string) *response.Engine

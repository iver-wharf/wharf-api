// Package request contains plain old Go types used in the Gin endpoint handlers
// and Swaggo documentation for the HTTP request models, with Gin- and
// Swaggo-specific Go tags.
package request

import "time"

// Reference doc about the Go tags:
//  TAG                  SOURCE                   DESCRIPTION
//  json:"foo"           encoding/json            Serializes field with the name "foo"
//  format:"date-time"   swaggo/swag              Swagger format
//  validate:"required"  swaggo/swag              Mark Swagger field as required/non-nullable
//  binding:"required"   go-playground/validator  Gin's Bind will error if nil or zero
//
// go-playground/validator uses the tag "validate" by default, but Gin overrides
// changes that to "binding".

// TokenSearch holds values used in verbatim searches for tokens.
type TokenSearch struct {
	Token    string `json:"token" format:"password"`
	UserName string `json:"userName"`
}

// Token specifies fields when creating a new token.
type Token struct {
	Token      string `json:"token" format:"password" validate:"required"`
	UserName   string `json:"userName" validate:"required"`
	ProviderID uint   `json:"providerId"`
}

// Token specifies fields when adding a new branch to a project.
type Branch struct {
	ProjectID uint   `json:"projectId" validate:"required"`
	Name      string `json:"name" validate:"required"`
	Default   bool   `json:"default"`
	TokenID   uint   `json:"tokenId"`
}

// LogOrStatusUpdate is a single log line, together with its timestamp of when
// it was logged; or a build status update.
//
// The build status field takes precedence, and if set it will update the
// build's status, while the message and the timestamp is ignored.
type LogOrStatusUpdate struct {
	Message   string      `json:"message"`
	Timestamp time.Time   `json:"timestamp" format:"date-time"`
	Status    BuildStatus `json:"status" enum:",Scheduling,Running,Completed,Failed"`
}

// BuildStatus is an enum of different states for a build.
type BuildStatus string

const (
	// BuildScheduling means the build has been registered, but no code
	// execution has begun yet. This is usually quite an ephemeral state.
	BuildScheduling BuildStatus = "Scheduling"
	// BuildRunning means the build is executing right now. The execution
	// engine has load in the target code paths and repositories.
	BuildRunning BuildStatus = "Running"
	// BuildCompleted means the build has finished execution successfully.
	BuildCompleted BuildStatus = "Completed"
	// BuildFailed means that something went wrong with the build. Could be a
	// misconfiguration in the .wharf-ci.yml file, or perhaps a scripting error
	// in some build step.
	BuildFailed BuildStatus = "Failed"
)

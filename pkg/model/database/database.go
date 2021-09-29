// Package database contains plain old Go types used by GORM as database models
// with GORM-specific Go tags.
package database

import (
	"time"

	"gopkg.in/guregu/null.v4"
)

// Structs naming conventions in this file:
//  - Go struct field names:  {type}Fields
//  - SQL column names:       {type}Columns
//
// Fields are added on-demand to these structs.

// Constraint convention in this file:
//
// When applying constraints using gorm tags, like:
//  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
// we apply it to both referencing fields.
//
// Examples:
//  - Build.Params and BuildParam.Build
//  - Project.Branches and Branch.Project
//
// One seems to take precedence, but to make sure and to keep the code
// consistent we add it to both referencing fields.

// ProviderFields holds the Go struct field names for each field.
// Useful in GORM .Where() statements to only select certain fields or in GORM
// Preload statements to select the correct field to preload.
var ProviderFields = struct {
	Name    string
	URL     string
	TokenID string
}{
	Name:    "Name",
	URL:     "URL",
	TokenID: "TokenID",
}

// Provider holds metadata about a connection to a remote provider. Some of
// importance are the URL field of where to find the remote, and the token field
// used to authenticate.
type Provider struct {
	ProviderID uint   `gorm:"primaryKey"`
	Name       string `gorm:"size:20;not null" enum:"azuredevops,gitlab,github"`
	URL        string `gorm:"size:500;not null"`
	TokenID    uint   `gorm:"nullable;default:NULL;index:provider_idx_token_id"`
	Token      *Token `gorm:"foreignKey:TokenID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
}

// TokenFields holds the Go struct field names for each field.
// Useful in GORM .Where() statements to only select certain fields or in GORM
// Preload statements to select the correct field to preload.
var TokenFields = struct {
	Token    string
	UserName string
}{
	Token:    "Token",
	UserName: "UserName",
}

// Token holds credentials for a remote provider.
type Token struct {
	TokenID  uint   `gorm:"primaryKey"`
	Token    string `gorm:"size:500; not null"`
	UserName string `gorm:"size:500"`
}

// ProjectFields holds the Go struct field names for each field.
// Useful in GORM .Where() statements to only select certain fields or in GORM
// Preload statements to select the correct field to preload.
var ProjectFields = struct {
	ProjectID string
	Name      string
	GroupName string
	TokenID   string
	Token     string
	Provider  string
	Branches  string
}{
	ProjectID: "ProjectID",
	Name:      "Name",
	GroupName: "GroupName",
	TokenID:   "TokenID",
	Token:     "Token",
	Provider:  "Provider",
	Branches:  "Branches",
}

// Project holds data about an imported project. A lot of the data is expected
// to be populated with data from the remote provider, such as the description
// and avatar.
type Project struct {
	ProjectID       uint      `gorm:"primaryKey"`
	Name            string    `gorm:"size:500;not null"`
	GroupName       string    `gorm:"size:500"`
	Description     string    `gorm:"size:500"`
	AvatarURL       string    `gorm:"size:500"`
	TokenID         uint      `gorm:"nullable;default:NULL;index:project_idx_token_id"`
	Token           *Token    `gorm:"foreignKey:TokenID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	ProviderID      uint      `gorm:"nullable;default:NULL;index:project_idx_provider_id"`
	Provider        *Provider `gorm:"foreignKey:ProviderID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	BuildDefinition string    `sql:"type:text"`
	Branches        []Branch  `gorm:"foreignKey:ProjectID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	GitURL          string    `gorm:"nullable;default:NULL"`
}

// BranchFields holds the Go struct field names for each field.
// Useful in GORM .Where() statements to only select certain fields or in GORM
// Preload statements to select the correct field to preload.
var BranchFields = struct {
	ProjectID string
	Name      string
	Default   string
	TokenID   string
}{
	ProjectID: "ProjectID",
	Name:      "Name",
	Default:   "Default",
	TokenID:   "TokenID",
}

// BranchColumns holds the DB column names for each field.
// Useful in GORM .Order() statements to order the results based on a specific
// column, which does not support the regular Go field names.
var BranchColumns = struct {
	BranchID string
}{
	BranchID: "branch_id",
}

// Branch is a single branch in the VCS that can be targeted during builds.
// For example a Git branch.
type Branch struct {
	BranchID  uint     `gorm:"primaryKey"`
	ProjectID uint     `gorm:"not null;index:branch_idx_project_id"`
	Project   *Project `gorm:"foreignKey:ProjectID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Name      string   `gorm:"not null"`
	Default   bool     `gorm:"not null"`
	TokenID   uint     `gorm:"nullable;default:NULL;index:branch_idx_token_id"`
	Token     Token    `gorm:"foreignKey:TokenID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
}

// BuildFields holds the Go struct field names for each field.
// Useful in GORM .Where() statements to only select certain fields or in GORM
// Preload statements to select the correct field to preload.
var BuildFields = struct {
	Params              string
	TestResultSummaries string
}{
	Params:              "Params",
	TestResultSummaries: "TestResultSummaries",
}

// BuildColumns holds the DB column names for each field.
// Useful in GORM .Order() statements to order the results based on a specific
// column, which does not support the regular Go field names.
var BuildColumns = struct {
	BuildID     string
	StatusID    string
	ScheduledOn string
	StartedOn   string
	CompletedOn string
	Environment string
	Stage       string
	IsInvalid   string
}{
	BuildID:     "build_id",
	StatusID:    "status_id",
	ScheduledOn: "scheduled_on",
	StartedOn:   "started_on",
	CompletedOn: "completed_on",
	Environment: "environment",
	Stage:       "stage",
	IsInvalid:   "is_invalid",
}

// Build holds data about the state of a build. Which parameters was used to
// start it, what status it holds, et.al.
type Build struct {
	BuildID             uint                `gorm:"primaryKey"`
	StatusID            BuildStatus         `gorm:"not null"`
	ProjectID           uint                `gorm:"not null;index:build_idx_project_id"`
	Project             *Project            `gorm:"foreignKey:ProjectID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ScheduledOn         null.Time           `gorm:"nullable;default:NULL"`
	StartedOn           null.Time           `gorm:"nullable;default:NULL"`
	CompletedOn         null.Time           `gorm:"nullable;default:NULL"`
	GitBranch           string              `gorm:"size:300;default:'';not null"`
	Environment         null.String         `gorm:"nullable;size:40" swaggertype:"string"`
	Stage               string              `gorm:"size:40;default:'';not null"`
	Params              []BuildParam        `gorm:"foreignKey:BuildID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	IsInvalid           bool                `gorm:"not null;default:false"`
	TestResultSummaries []TestResultSummary `gorm:"foreignKey:BuildID"`
}

// BuildStatus is an enum of different states for a build.
type BuildStatus int

const (
	// BuildScheduling means the build has been registered, but no code
	// execution has begun yet. This is usually quite an ephemeral state.
	BuildScheduling BuildStatus = iota
	// BuildRunning means the build is executing right now. The execution
	// engine has load in the target code paths and repositories.
	BuildRunning
	// BuildCompleted means the build has finished execution successfully.
	BuildCompleted
	// BuildFailed means that something went wrong with the build. Could be a
	// misconfiguration in the .wharf-ci.yml file, or perhaps a scripting error
	// in some build step.
	BuildFailed
)

// BuildParam holds the name and value of an input parameter fed into a build.
type BuildParam struct {
	BuildParamID uint   `gorm:"primaryKey"`
	BuildID      uint   `gorm:"not null;index:buildparam_idx_build_id"`
	Build        *Build `gorm:"foreignKey:BuildID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Name         string `gorm:"not null"`
	Value        string `gorm:"nullable"`
}

// Log is a single logged line for a build.
type Log struct {
	LogID     uint      `gorm:"primaryKey"`
	BuildID   uint      `gorm:"not null;index:log_idx_build_id"`
	Build     *Build    `gorm:"foreignKey:BuildID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Message   string    `sql:"type:text"`
	Timestamp time.Time `gorm:"not null"`
}

// Param holds the definition of an input parameter for a project.
type Param struct {
	ParamID      int    `gorm:"primaryKey"`
	Name         string `gorm:"not null"`
	Type         string `gorm:"not null"`
	Value        string
	DefaultValue string
}

// ArtifactColumns holds the DB column names for each field.
// Useful in GORM .Order() statements to order the results based on a specific
// column, which does not support the regular Go field names.
var ArtifactColumns = struct {
	ArtifactID string
	FileName   string
}{
	ArtifactID: "artifact_id",
	FileName:   "file_name",
}

// Artifact holds the binary data as well as metadata about that binary such as
// the file name and which build it belongs to.
type Artifact struct {
	ArtifactID uint   `gorm:"primaryKey"`
	BuildID    uint   `gorm:"not null;index:artifact_idx_build_id"`
	Build      *Build `gorm:"foreignKey:BuildID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Name       string `gorm:"not null"`
	FileName   string `gorm:"nullable"`
	Data       []byte `gorm:"nullable"`
}

// TestResultSummary contains data about a single test result file.
type TestResultSummary struct {
	TestResultSummaryID uint      `gorm:"primaryKey"`
	FileName            string    `gorm:"nullable"`
	ArtifactID          uint      `gorm:"not null;index:testresultsummary_idx_artifact_id"`
	Artifact            *Artifact `gorm:"foreignKey:ArtifactID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	BuildID             uint      `gorm:"not null;index:testresultsummary_idx_build_id"`
	Build               *Build    `gorm:"foreignKey:BuildID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Total               uint      `gorm:"not null"`
	Failed              uint      `gorm:"not null"`
	Passed              uint      `gorm:"not null"`
	Skipped             uint      `gorm:"not null"`
}

// TestResultStatus is an enum of different states a test result can be in.
type TestResultStatus string

const (
	// TestResultStatusSuccess means the test succeeded.
	TestResultStatusSuccess TestResultStatus = "Success"
	// TestResultStatusFailed means the test failed.
	TestResultStatusFailed TestResultStatus = "Failed"
	// TestResultStatusSkipped means the test was skipped.
	TestResultStatusSkipped TestResultStatus = "Skipped"
)

// TestResultDetail contains data about a single test in a test result file.
type TestResultDetail struct {
	TestResultDetailID uint             `gorm:"primaryKey"`
	ArtifactID         uint             `gorm:"not null;index:testresultdetail_idx_artifact_id"`
	Artifact           *Artifact        `gorm:"foreignKey:ArtifactID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	BuildID            uint             `gorm:"not null;index:testresultdetail_idx_build_id"`
	Build              *Build           `gorm:"foreignKey:BuildID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Name               string           `gorm:"not null"`
	Message            null.String      `gorm:"nullable"`
	StartedOn          null.Time        `gorm:"nullable;default:NULL;"`
	CompletedOn        null.Time        `gorm:"nullable;default:NULL;"`
	Status             TestResultStatus `gorm:"not null"`
}

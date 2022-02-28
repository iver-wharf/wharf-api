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

// TimeMetadata contains fields that GORM will recognize and update
// automatically for us.
//
// Docs: https://gorm.io/docs/models.html#Creating-Updating-Time-Unix-Milli-Nano-Seconds-Tracking
type TimeMetadata struct {
	CreatedAt *time.Time `gorm:"nullable"`
	UpdatedAt *time.Time `gorm:"nullable"`
}

// SafeSQLName represents a value that is safe to use as an SQL table or column
// name without the need of escaping.
//
// It is merely semantical and has no validation attached. Values of this type
// should never be constructed from user input.
type SafeSQLName string

// ProviderFields holds the Go struct field names for each field.
// Useful in GORM .Where() statements to only select certain fields or in GORM
// Preload statements to select the correct field to preload.
var ProviderFields = struct {
	ProviderID string
	Name       string
	URL        string
	TokenID    string
}{
	ProviderID: "ProviderID",
	Name:       "Name",
	URL:        "URL",
	TokenID:    "TokenID",
}

// ProviderColumns holds the DB column names for each field.
// Useful in GORM .Order() statements to order the results based on a specific
// column, which does not support the regular Go field names.
var ProviderColumns = struct {
	ProviderID SafeSQLName
	Name       SafeSQLName
	URL        SafeSQLName
	TokenID    SafeSQLName
}{
	ProviderID: "provider_id",
	Name:       "name",
	URL:        "url",
	TokenID:    "token_id",
}

// Provider holds metadata about a connection to a remote provider. Some of
// importance are the URL field of where to find the remote, and the token field
// used to authenticate.
type Provider struct {
	TimeMetadata
	ProviderID uint   `gorm:"primaryKey"`
	Name       string `gorm:"size:20;not null"`
	URL        string `gorm:"size:500;not null"`
	TokenID    uint   `gorm:"nullable;default:NULL;index:provider_idx_token_id"`
	Token      *Token `gorm:"foreignKey:TokenID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
}

// TokenFields holds the Go struct field names for each field.
// Useful in GORM .Where() statements to only select certain fields or in GORM
// Preload statements to select the correct field to preload.
var TokenFields = struct {
	TokenID  string
	Token    string
	UserName string
}{
	TokenID:  "TokenID",
	Token:    "Token",
	UserName: "UserName",
}

// TokenColumns holds the DB column names for each field.
// Useful in GORM .Order() statements to order the results based on a specific
// column, which does not support the regular Go field names.
var TokenColumns = struct {
	TokenID  SafeSQLName
	Token    SafeSQLName
	UserName SafeSQLName
}{
	TokenID:  "token_id",
	Token:    "token",
	UserName: "user_name",
}

// Token holds credentials for a remote provider.
type Token struct {
	TimeMetadata
	TokenID  uint   `gorm:"primaryKey"`
	Value    string `gorm:"size:500;not null"`
	UserName string `gorm:"size:500;not null;default:''"`
}

// ProjectFields holds the Go struct field names for each field.
// Useful in GORM .Where() statements to only select certain fields or in GORM
// Preload statements to select the correct field to preload.
var ProjectFields = struct {
	ProjectID       string
	RemoteProjectID string
	Name            string
	GroupName       string
	Description     string
	AvatarURL       string
	TokenID         string
	Token           string
	ProviderID      string
	Provider        string
	BuildDefinition string
	Branches        string
	GitURL          string
	Overrides       string
}{
	ProjectID:       "ProjectID",
	Name:            "Name",
	GroupName:       "GroupName",
	Description:     "Description",
	AvatarURL:       "AvatarURL",
	TokenID:         "TokenID",
	Token:           "Token",
	ProviderID:      "ProviderID",
	Provider:        "Provider",
	BuildDefinition: "BuildDefinition",
	Branches:        "Branches",
	GitURL:          "GitURL",
	Overrides:       "Overrides",
}

// ProjectColumns holds the DB column names for each field.
// Useful in GORM .Order() statements to order the results based on a specific
// column, which does not support the regular Go field names.
var ProjectColumns = struct {
	ProjectID       SafeSQLName
	RemoteProjectID SafeSQLName
	Name            SafeSQLName
	GroupName       SafeSQLName
	Description     SafeSQLName
	TokenID         SafeSQLName
	GitURL          SafeSQLName
}{
	ProjectID:       "project_id",
	RemoteProjectID: "remote_project_id",
	Name:            "name",
	GroupName:       "group_name",
	Description:     "description",
	TokenID:         "token_id",
	GitURL:          "git_url",
}

// Project holds data about an imported project. A lot of the data is expected
// to be populated with data from the remote provider, such as the description
// and avatar.
type Project struct {
	TimeMetadata
	ProjectID       uint      `gorm:"primaryKey"`
	RemoteProjectID string    `gorm:"not null;default:''"`
	Name            string    `gorm:"size:500;not null"`
	GroupName       string    `gorm:"size:500;not null;default:''"`
	Description     string    `gorm:"size:500;not null;default:''"`
	AvatarURL       string    `gorm:"size:500;not null;default:''"`
	TokenID         *uint     `gorm:"nullable;default:NULL;index:project_idx_token_id"`
	Token           *Token    `gorm:"foreignKey:TokenID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	ProviderID      *uint     `gorm:"nullable;default:NULL;index:project_idx_provider_id"`
	Provider        *Provider `gorm:"foreignKey:ProviderID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	BuildDefinition string    `gorm:"not null;default:''"`
	Branches        []Branch  `gorm:"foreignKey:ProjectID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	GitURL          string    `gorm:"not null;default:''"`

	Overrides ProjectOverrides `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// ProjectOverrides holds data about a project's overridden values.
type ProjectOverrides struct {
	ProjectOverridesID uint   `gorm:"primaryKey"`
	ProjectID          uint   `gorm:"foreignKey:ProjectID;uniqueIndex:project_overrides_idx_project_id"`
	Description        string `gorm:"size:500;not null;default:''"`
	AvatarURL          string `gorm:"size:500;not null;default:''"`
	GitURL             string `gorm:"not null;default:''"`
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
	BranchID SafeSQLName
	Name     SafeSQLName
}{
	BranchID: "branch_id",
	Name:     "name",
}

// Branch is a single branch in the VCS that can be targeted during builds.
// For example a Git branch.
type Branch struct {
	TimeMetadata
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
	ProjectID           string
	StatusID            string
	GitBranch           string
	Environment         string
	Stage               string
	WorkerID            string
	IsInvalid           string
	Params              string
	TestResultSummaries string
}{
	ProjectID:           "ProjectID",
	StatusID:            "StatusID",
	GitBranch:           "GitBranch",
	Environment:         "Environment",
	Stage:               "Stage",
	WorkerID:            "WorkerID",
	IsInvalid:           "IsInvalid",
	Params:              "Params",
	TestResultSummaries: "TestResultSummaries",
}

// BuildColumns holds the DB column names for each field.
// Useful in GORM .Order() statements to order the results based on a specific
// column, which does not support the regular Go field names.
var BuildColumns = struct {
	BuildID     SafeSQLName
	StatusID    SafeSQLName
	ScheduledOn SafeSQLName
	StartedOn   SafeSQLName
	CompletedOn SafeSQLName
	GitBranch   SafeSQLName
	Environment SafeSQLName
	Stage       SafeSQLName
	IsInvalid   SafeSQLName
}{
	BuildID:     "build_id",
	StatusID:    "status_id",
	ScheduledOn: "scheduled_on",
	StartedOn:   "started_on",
	CompletedOn: "completed_on",
	GitBranch:   "git_branch",
	Environment: "environment",
	Stage:       "stage",
	IsInvalid:   "is_invalid",
}

// BuildSizes holds the DB column size limits.
// Useful when validating the fields attempting to insert values into the
// database.
var BuildSizes = struct {
	EngineID int
}{
	EngineID: 32,
}

// Build holds data about the state of a build. Which parameters was used to
// start it, what status it holds, et.al.
type Build struct {
	TimeMetadata
	BuildID             uint                `gorm:"primaryKey"`
	StatusID            BuildStatus         `gorm:"not null"`
	ProjectID           uint                `gorm:"not null;index:build_idx_project_id"`
	Project             *Project            `gorm:"foreignKey:ProjectID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ScheduledOn         null.Time           `gorm:"nullable;default:NULL"`
	StartedOn           null.Time           `gorm:"nullable;default:NULL"`
	CompletedOn         null.Time           `gorm:"nullable;default:NULL"`
	GitBranch           string              `gorm:"size:300;not null;default:''"`
	Environment         null.String         `gorm:"nullable;size:40" swaggertype:"string"`
	Stage               string              `gorm:"size:40;not null;default:''"`
	WorkerID            string              `gorm:"size:40;not null;default:'';index:build_idx_worker_id"`
	Params              []BuildParam        `gorm:"foreignKey:BuildID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	IsInvalid           bool                `gorm:"not null;default:false"`
	TestResultSummaries []TestResultSummary `gorm:"foreignKey:BuildID"`
	EngineID            string              `gorm:"size:32;not null;default:''"`
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

// IsValid returns false if the underlying type is an unknown enum value.
// 	BuildScheduling.IsValid()   // => true
// 	(BuildStatus(-1)).IsValid() // => false
func (buildStatus BuildStatus) IsValid() bool {
	return buildStatus >= BuildScheduling && buildStatus <= BuildFailed
}

// BuildParamFields holds the Go struct field names for each field.
// Useful in GORM .Where() statements to only select certain fields or in GORM
// Preload statements to select the correct field to preload.
var BuildParamFields = struct {
	Value string
}{
	Value: "Value",
}

// BuildParam holds the name and value of an input parameter fed into a build.
type BuildParam struct {
	BuildParamID uint   `gorm:"primaryKey"`
	BuildID      uint   `gorm:"not null;index:buildparam_idx_build_id"`
	Build        *Build `gorm:"foreignKey:BuildID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Name         string `gorm:"not null"`
	Value        string `gorm:"not null;default:''"`
}

// Log is a single logged line for a build.
type Log struct {
	LogID     uint      `gorm:"primaryKey"`
	BuildID   uint      `gorm:"not null;index:log_idx_build_id"`
	Build     *Build    `gorm:"foreignKey:BuildID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Message   string    `sql:"type:text"`
	Timestamp time.Time `gorm:"not null"`
}

// ParamFields holds the Go struct field names for each field.
// Useful in GORM .Where() statements to only select certain fields or in GORM
// Preload statements to select the correct field to preload.
var ParamFields = struct {
	Value        string
	DefaultValue string
}{
	Value:        "Value",
	DefaultValue: "DefaultValue",
}

// Param holds the definition of an input parameter for a project.
type Param struct {
	ParamID      int    `gorm:"primaryKey"`
	Name         string `gorm:"not null"`
	Type         string `gorm:"not null"`
	Value        string `gorm:"not null;default:''"`
	DefaultValue string `gorm:"not null;default:''"`
}

// ArtifactColumns holds the DB column names for each field.
// Useful in GORM .Order() statements to order the results based on a specific
// column, which does not support the regular Go field names.
var ArtifactColumns = struct {
	ArtifactID SafeSQLName
	Name       SafeSQLName
	FileName   SafeSQLName
}{
	ArtifactID: "artifact_id",
	Name:       "name",
	FileName:   "file_name",
}

// ArtifactFields holds the Go struct field names for each field.
// Useful in GORM .Where() statements to only select certain fields or in GORM
// Preload statements to select the correct field to preload.
var ArtifactFields = struct {
	BuildID  string
	Name     string
	FileName string
}{
	BuildID:  "BuildID",
	Name:     "Name",
	FileName: "FileName",
}

// Artifact holds the binary data as well as metadata about that binary such as
// the file name and which build it belongs to.
type Artifact struct {
	TimeMetadata
	ArtifactID uint   `gorm:"primaryKey"`
	BuildID    uint   `gorm:"not null;index:artifact_idx_build_id"`
	Build      *Build `gorm:"foreignKey:BuildID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Name       string `gorm:"not null"`
	FileName   string `gorm:"not null;default:''"`
	Data       []byte `gorm:"nullable"`
}

// TestResultSummaryFields holds the Go struct field names for each field.
// Useful in GORM .Where() statements to only select certain fields or in GORM
// Preload statements to select the correct field to preload.
var TestResultSummaryFields = struct {
	FileName string
}{
	FileName: "FileName",
}

// TestResultSummary contains data about a single test result file.
type TestResultSummary struct {
	TimeMetadata
	TestResultSummaryID uint      `gorm:"primaryKey"`
	FileName            string    `gorm:"not null;default:''"`
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
	TimeMetadata
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

package main

import (
	"time"

	"gopkg.in/guregu/null.v4"
)

// Consts conventions in this file:
//  - Go struct field name:           {type}Field{FieldName}
//  - Association struct field names: {type}Assoc{FieldName}
//  - JSON property names:            {type}JSON{FieldName}
//  - SQL column names:               {type}Column{FieldName}

const (
	providerFieldName      = "Name"
	providerFieldURL       = "URL"
	providerFieldUploadURL = "UploadURL"
	providerFieldTokenID   = "TokenID"
)

type Provider struct {
	ProviderID uint   `gorm:"primaryKey" json:"providerId"`
	Name       string `gorm:"size:20;not null" json:"name" enum:"azuredevops,gitlab,github"`
	URL        string `gorm:"size:500;not null" json:"url"`
	UploadURL  string `gorm:"size:500" json:"uploadUrl"`
	TokenID    uint   `gorm:"nullable;default:NULL;index:provider_idx_token_id" json:"tokenId"`
	Token      *Token `gorm:"foreignKey:TokenID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
}

const (
	tokenFieldToken    = "Token"
	tokenFieldUserName = "UserName"
)

type Token struct {
	TokenID  uint   `gorm:"primaryKey" json:"tokenId"`
	Token    string `gorm:"size:500; not null" json:"token" format:"password"`
	UserName string `gorm:"size:500" json:"userName"`
}

const (
	projectFieldProjectID = "ProjectID"
	projectFieldTokenID   = "TokenID"
	projectFieldName      = "Name"
	projectFieldGroupName = "GroupName"
	projectAssocProvider  = "Provider"
	projectAssocBranches  = "Branches"
	projectAssocToken     = "Token"
)

type Project struct {
	ProjectID       uint      `gorm:"primaryKey" json:"projectId"`
	Name            string    `gorm:"size:500;not null" json:"name"`
	GroupName       string    `gorm:"size:500" json:"groupName"`
	Description     string    `gorm:"size:500" json:"description"`
	AvatarURL       string    `gorm:"size:500" json:"avatarUrl"`
	TokenID         uint      `gorm:"nullable;default:NULL;index:project_idx_token_id" json:"tokenId"`
	Token           *Token    `gorm:"foreignKey:TokenID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	ProviderID      uint      `gorm:"nullable;default:NULL;index:project_idx_provider_id" json:"providerId"`
	Provider        *Provider `gorm:"foreignKey:ProviderID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"provider"`
	BuildDefinition string    `sql:"type:text" json:"buildDefinition"`
	Branches        []Branch  `gorm:"foreignKey:ProjectID" json:"branches"`
	GitURL          string    `gorm:"nullable;default:NULL" json:"gitUrl"`
}

const (
	branchFieldProjectID = "ProjectID"
	branchFieldName      = "Name"
	branchFieldDefault   = "Default"
	branchFieldTokenID   = "TokenID"
)

type Branch struct {
	BranchID  uint     `gorm:"primaryKey" json:"branchId"`
	ProjectID uint     `gorm:"not null;index:branch_idx_project_id" json:"projectId"`
	Project   *Project `gorm:"foreignKey:ProjectID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Name      string   `gorm:"not null" json:"name"`
	Default   bool     `gorm:"not null" json:"default"`
	TokenID   uint     `gorm:"nullable;default:NULL;index:branch_idx_token_id" json:"tokenId"`
	Token     Token    `gorm:"foreignKey:TokenID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
}

const (
	buildColumnBuildID = "build_id"
	buildAssocParams   = "Params"
	buildColumnName    = "name"
)

var buildJSONToColumns = map[string]string{
	"buildId":     buildColumnBuildID,
	"environment": "environment",
	"finishedOn":  "completed_on",
	"scheduledOn": "scheduled_on",
	"stage":       "stage",
	"startedOn":   "started_on",
	"statusId":    "status_id",
	"isInvalid":   "is_invalid",
}

type Build struct {
	BuildID             uint                `gorm:"primaryKey" json:"buildId"`
	StatusID            BuildStatus         `gorm:"not null" json:"statusId"`
	ProjectID           uint                `gorm:"not null;index:build_idx_project_id" json:"projectId"`
	Project             *Project            `gorm:"foreignKey:ProjectID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	ScheduledOn         *time.Time          `gorm:"nullable;default:NULL" json:"scheduledOn" format:"date-time"`
	StartedOn           *time.Time          `gorm:"nullable;default:NULL" json:"startedOn" format:"date-time"`
	CompletedOn         *time.Time          `gorm:"nullable;default:NULL" json:"finishedOn" format:"date-time"`
	GitBranch           string              `gorm:"size:300;default:'';not null" json:"gitBranch"`
	Environment         null.String         `gorm:"nullable;size:40" json:"environment" swaggertype:"string"`
	Stage               string              `gorm:"size:40;default:'';not null" json:"stage"`
	Params              []BuildParam        `gorm:"foreignKey:BuildID" json:"params"`
	IsInvalid           bool                `gorm:"not null;default:false" json:"isInvalid"`
	TestResultSummaries []TestResultSummary `gorm:"foreignKey:BuildID" json:"testResultSummaries"`
}

type BuildParam struct {
	BuildParamID uint   `gorm:"primaryKey" json:"-"`
	BuildID      uint   `gorm:"not null;index:buildparam_idx_build_id" json:"buildId"`
	Build        *Build `gorm:"foreignKey:BuildID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Name         string `gorm:"not null" json:"name"`
	Value        string `gorm:"nullable" json:"value"`
}

type Log struct {
	LogID     uint      `gorm:"primaryKey" json:"logId"`
	BuildID   uint      `gorm:"not null;index:log_idx_build_id" json:"buildId"`
	Build     *Build    `gorm:"foreignKey:BuildID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Message   string    `sql:"type:text" json:"message"`
	Timestamp time.Time `gorm:"not null" json:"timestamp" format:"date-time"`
}

type Param struct {
	ParamID      int    `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"not null" json:"name"`
	Type         string `gorm:"not null" json:"type"`
	Value        string `json:"value"`
	DefaultValue string `json:"defaultValue"`
}

const (
	artifactColumnArtifactID = "artifact_id"
	artifactColumnFileName   = "file_name"
)

type Artifact struct {
	ArtifactID uint   `gorm:"primaryKey" json:"artifactId"`
	BuildID    uint   `gorm:"not null;index:artifact_idx_build_id" json:"buildId"`
	Build      *Build `gorm:"foreignKey:BuildID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Name       string `gorm:"not null" json:"name"`
	FileName   string `gorm:"nullable" json:"fileName"`
	Data       []byte `gorm:"nullable" json:"-"`
}

// TestResultSummary contains data about a single test result file.
type TestResultSummary struct {
	TestResultSummaryID uint      `gorm:"primaryKey" json:"testResultSummaryId"`
	FileName            string    `gorm:"nullable" json:"fileName"`
	ArtifactID          uint      `gorm:"not null;index:testresultsummary_idx_artifact_id" json:"artifactId"`
	Artifact            *Artifact `gorm:"foreignKey:ArtifactID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	BuildID             uint      `gorm:"not null;index:testresultsummary_idx_build_id" json:"buildId"`
	Build               *Build    `gorm:"foreignKey:BuildID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Total               uint      `gorm:"not null" json:"total"`
	Failed              uint      `gorm:"not null" json:"failed"`
	Passed              uint      `gorm:"not null" json:"passed"`
	Skipped             uint      `gorm:"not null" json:"skipped"`
}

type testResultStatus string

const (
	testResultStatusSuccess testResultStatus = "Success"
	testResultStatusFailed  testResultStatus = "Failed"
	testResultStatusNoTests testResultStatus = "No tests"
	testResultStatusSkipped testResultStatus = "Skipped"
)

// TestResultDetail contains data about a single test in a test result file.
type TestResultDetail struct {
	TestResultDetailID uint             `gorm:"primaryKey" json:"testResultDetailId"`
	ArtifactID         uint             `gorm:"not null;index:testresultdetail_idx_artifact_id" json:"artifactId"`
	Artifact           *Artifact        `gorm:"foreignKey:ArtifactID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	BuildID            uint             `gorm:"not null;index:testresultdetail_idx_build_id" json:"buildId"`
	Build              *Build           `gorm:"foreignKey:BuildID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Name               string           `gorm:"not null" json:"name"`
	Message            null.String      `gorm:"nullable" json:"message" swaggertype:"string"`
	StartedOn          *time.Time       `gorm:"nullable;default:NULL;" json:"startedOn" format:"date-time"`
	CompletedOn        *time.Time       `gorm:"nullable;default:NULL;" json:"completedOn" format:"date-time"`
	Status             testResultStatus `gorm:"not null" enums:"Failed,Passed,Skipped" json:"status"`
}

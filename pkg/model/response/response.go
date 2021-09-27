// Package response contains plain old Go types returned by wharf-web in the
// HTTP responses, with Swaggo-specific Go tags.
package response

import "gopkg.in/guregu/null.v4"

// Artifact holds the binary data as well as metadata about that binary such as
// the file name and which build it belongs to.
type Artifact struct {
	ArtifactID uint   `json:"artifactId"`
	BuildID    uint   `json:"buildId"`
	Name       string `json:"name"`
	FileName   string `json:"fileName"`
}

// ArtifactMetadata contains the file name and artifact ID of an Artifact.
type ArtifactMetadata struct {
	FileName   string `json:"fileName"`
	ArtifactID uint   `json:"artifactId"`
}

// Branch holds details about a project's branch.
type Branch struct {
	BranchID  uint   `json:"branchId"`
	ProjectID uint   `json:"projectId"`
	Name      string `json:"name"`
	Default   bool   `json:"default"`
	TokenID   uint   `json:"tokenId"`
}

// Project holds details about a project.
type Project struct {
}

// TestStatus is an enum of different states a test run or test summary can be
// in.
type TestStatus string

const (
	// TestStatusSuccess means the test run or test summary passed, or in the
	// case that there are multiple tests then that there are no failing tests
	// and at least one successful test.
	TestStatusSuccess TestStatus = "Success"

	// TestStatusFailed means the test run or test summary failed, or in the
	// case that there are multiple tests then that at least one test failed.
	TestStatusFailed TestStatus = "Failed"

	// TestStatusNoTests means the test run or test summary is inconclusive,
	// where there are neither any passing nor failing tests.
	TestStatusNoTests TestStatus = "No tests"
)

// TestResultDetail contains data about a single test in a test result file.
type TestResultDetail struct {
	TestResultDetailID uint             `json:"testResultDetailId"`
	ArtifactID         uint             `json:"artifactId"`
	BuildID            uint             `json:"buildId"`
	Name               string           `json:"name"`
	Message            null.String      `json:"message" swaggertype:"string"`
	StartedOn          null.Time        `json:"startedOn" format:"date-time"`
	CompletedOn        null.Time        `json:"completedOn" format:"date-time"`
	Status             TestResultStatus `json:"status" enum:"Failed,Passed,Skipped"`
}

// TestResultListSummary contains data about several test result files.
type TestResultListSummary struct {
	BuildID uint `json:"buildId"`
	Total   uint `json:"total"`
	Failed  uint `json:"failed"`
	Passed  uint `json:"passed"`
	Skipped uint `json:"skipped"`
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

// TestResultSummary contains data about a single test result file.
type TestResultSummary struct {
	TestResultSummaryID uint   `json:"testResultSummaryId"`
	FileName            string `json:"fileName"`
	ArtifactID          uint   `json:"artifactId"`
	BuildID             uint   `json:"buildId"`
	Total               uint   `json:"total"`
	Failed              uint   `json:"failed"`
	Passed              uint   `json:"passed"`
	Skipped             uint   `json:"skipped"`
}

// TestsResults holds how many builds has passed and failed. A test result has
// the status of "Failed" if there are any failed tests, "Success" if there are
// any passing tests and no failed tests, and "No tests" if there are no failed
// nor passing tests.
type TestsResults struct {
	Passed uint       `json:"passed"`
	Failed uint       `json:"failed"`
	Status TestStatus `json:"status" enums:"Success,Failed,No tests"`
}

// Token holds credentials for a remote provider.
type Token struct {
	TokenID  uint   `json:"tokenId"`
	Token    string `json:"token" format:"password"`
	UserName string `json:"userName"`
}

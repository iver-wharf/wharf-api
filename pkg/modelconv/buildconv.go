package modelconv

import (
	"strconv"

	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/request"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/response"
)

// DBBuildParamsToResponses converts a slice of database build parameters to a
// slice of response build parameters.
func DBBuildParamsToResponses(dbParams []database.BuildParam) []response.BuildParam {
	resParams := make([]response.BuildParam, len(dbParams))
	for i, dbParam := range dbParams {
		resParams[i] = DBBuildParamToResponse(dbParam)
	}
	return resParams
}

// DBBuildParamToResponse converts a database build parameter to a response
// build parameter.
func DBBuildParamToResponse(dbParam database.BuildParam) response.BuildParam {
	return response.BuildParam{
		BuildID: dbParam.BuildID,
		Name:    dbParam.Name,
		Value:   dbParam.Value,
	}
}

// DBBuildsToResponses converts a slice of database builds to a slice of
// response builds.
func DBBuildsToResponses(dbBuilds []database.Build) []response.Build {
	resBuilds := make([]response.Build, len(dbBuilds))
	for i, dbBuild := range dbBuilds {
		resBuilds[i] = DBBuildToResponse(dbBuild)
	}
	return resBuilds
}

// DBBuildToResponse converts a database build to a response build.
func DBBuildToResponse(dbBuild database.Build) response.Build {
	var (
		failed  uint
		passed  uint
		skipped uint
	)
	for _, summary := range dbBuild.TestResultSummaries {
		failed += summary.Failed
		passed += summary.Passed
		skipped += summary.Skipped
	}
	resListSummary := response.TestResultListSummary{
		BuildID: dbBuild.BuildID,
		Total:   failed + passed + skipped,
		Passed:  passed,
		Skipped: skipped,
		Failed:  failed,
	}
	return response.Build{
		TimeMetadata:          DBTimeMetadataToResponse(dbBuild.TimeMetadata),
		BuildID:               dbBuild.BuildID,
		StatusID:              int(dbBuild.StatusID),
		Status:                DBBuildStatusToResponse(dbBuild.StatusID),
		ProjectID:             dbBuild.ProjectID,
		ScheduledOn:           dbBuild.ScheduledOn,
		StartedOn:             dbBuild.StartedOn,
		CompletedOn:           dbBuild.CompletedOn,
		GitBranch:             dbBuild.GitBranch,
		Environment:           dbBuild.Environment,
		Stage:                 dbBuild.Stage,
		Params:                DBBuildParamsToResponses(dbBuild.Params),
		IsInvalid:             dbBuild.IsInvalid,
		TestResultSummaries:   DBTestResultSummariesToResponses(dbBuild.TestResultSummaries),
		TestResultListSummary: resListSummary,
	}
}

// DBBuildToResponseBuildReferenceWrapper converts a database build to a
// response build reference wrapper.
func DBBuildToResponseBuildReferenceWrapper(dbBuild database.Build) response.BuildReferenceWrapper {
	return response.BuildReferenceWrapper{
		BuildReference: strconv.FormatUint(uint64(dbBuild.BuildID), 10),
	}
}

// DBBuildStatusToResponse converts a database build status to a response
// build status.
func DBBuildStatusToResponse(dbStatus database.BuildStatus) response.BuildStatus {
	switch dbStatus {
	case database.BuildScheduling:
		return response.BuildScheduling
	case database.BuildRunning:
		return response.BuildRunning
	case database.BuildCompleted:
		return response.BuildCompleted
	case database.BuildFailed:
		return response.BuildFailed
	default:
		return response.BuildScheduling
	}
}

// ReqBuildStatusToDatabase converts a request build status to a database
// build status.
func ReqBuildStatusToDatabase(reqStatus request.BuildStatus) (database.BuildStatus, bool) {
	switch reqStatus {
	case request.BuildScheduling:
		return database.BuildScheduling, true
	case request.BuildRunning:
		return database.BuildRunning, true
	case request.BuildCompleted:
		return database.BuildCompleted, true
	case request.BuildFailed:
		return database.BuildFailed, true
	default:
		return database.BuildScheduling, false
	}
}

package modelconv

import (
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
)

// DBTestResultSummariesToResponses converts a slice of database
// test result summaries to a slice of response test result summaries.
func DBTestResultSummariesToResponses(dbSummaries []database.TestResultSummary) []response.TestResultSummary {
	resSummaries := make([]response.TestResultSummary, len(dbSummaries))
	for i, dbSummary := range dbSummaries {
		resSummaries[i] = DBTestResultSummaryToResponse(dbSummary)
	}
	return resSummaries
}

// DBTestResultSummaryToResponse converts a database test result summary to a
// response test result summary.
func DBTestResultSummaryToResponse(dbSummary database.TestResultSummary) response.TestResultSummary {
	return response.TestResultSummary{
		TimeMetadata:        DBTimeMetadataToResponse(dbSummary.TimeMetadata),
		TestResultSummaryID: dbSummary.TestResultSummaryID,
		FileName:            dbSummary.FileName,
		ArtifactID:          dbSummary.ArtifactID,
		BuildID:             dbSummary.BuildID,
		Total:               dbSummary.Total,
		Failed:              dbSummary.Failed,
		Passed:              dbSummary.Passed,
		Skipped:             dbSummary.Skipped,
	}
}

// DBTestResultDetailsToResponses converts a slice of database
// test result details to a slice of response test result details.
func DBTestResultDetailsToResponses(dbDetails []database.TestResultDetail) []response.TestResultDetail {
	resDetails := make([]response.TestResultDetail, len(dbDetails))
	for i, dbDetail := range dbDetails {
		resDetails[i] = DBTestResultDetailToResponse(dbDetail)
	}
	return resDetails
}

// DBTestResultDetailToResponse converts a database test result detail to a
// response test result detail.
func DBTestResultDetailToResponse(dbDetail database.TestResultDetail) response.TestResultDetail {
	return response.TestResultDetail{
		TimeMetadata:       DBTimeMetadataToResponse(dbDetail.TimeMetadata),
		TestResultDetailID: dbDetail.TestResultDetailID,
		ArtifactID:         dbDetail.ArtifactID,
		BuildID:            dbDetail.BuildID,
		Name:               dbDetail.Name,
		Message:            dbDetail.Message,
		StartedOn:          dbDetail.StartedOn,
		CompletedOn:        dbDetail.CompletedOn,
		Status:             DBTestResultStatusToResponse(dbDetail.Status),
	}
}

// DBTestResultStatusToResponse converts a database test result status to a
// response test result status.
func DBTestResultStatusToResponse(dbStatus database.TestResultStatus) response.TestResultStatus {
	switch dbStatus {
	case database.TestResultStatusSuccess:
		return response.TestResultStatusSuccess
	case database.TestResultStatusSkipped:
		return response.TestResultStatusSkipped
	case database.TestResultStatusFailed:
		return response.TestResultStatusFailed
	default:
		return response.TestResultStatus(dbStatus)
	}
}

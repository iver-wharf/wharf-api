package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/internal/ctxparser"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"github.com/iver-wharf/wharf-core/pkg/problem"
	"gorm.io/gorm"
)

type buildTestResultModule struct {
	Database *gorm.DB
}

func (m buildTestResultModule) Register(r gin.IRouter) {
	testResult := r.Group("/test-result")
	{
		testResult.POST("/", m.createBuildTestResultHandler)

		testResult.GET("/detail", m.getBuildAllTestResultDetailListHandler)

		testResult.GET("/summary", m.getBuildAllTestResultSummaryListHandler)
		testResult.GET("/summary/:artifactId", m.getBuildTestResultSummaryHandler)
		testResult.GET("/summary/:artifactId/detail", m.getBuildTestResultDetailListHandler)

		testResult.GET("/list-summary", m.getBuildAllTestResultListSummaryHandler)
	}
}

// createBuildTestResultHandler godoc
// @id createBuildTestResult
// @summary Post test result data
// @tags test-result
// @accept multipart/form-data
// @param buildId path int true "Build ID"
// @param files formData file true "Test result file"
// @success 201 {object} []response.ArtifactMetadata "Added new test result data and created summaries"
// @failure 400 {object} problem.Response "Bad request"
// @failure 502 {object} problem.Response "Database unreachable or bad gateway"
// @router /build/{buildId}/test-result [post]
func (m buildTestResultModule) createBuildTestResultHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildId")
	if !ok {
		return
	}

	files, err := ctxparser.ParseMultipartFormDataFiles(c, "files")
	if err != nil {
		ginutil.WriteMultipartFormReadError(c, err,
			fmt.Sprintf("Failed reading multipart-form's file data from request body when uploading"+
				" new test result for build with ID %d.", buildID))
		return
	}

	dbArtifacts, ok := createArtifacts(c, m.Database, files, buildID)
	if !ok {
		return
	}

	var dbAllDetails []database.TestResultDetail

	dbAllSummaries := make([]database.TestResultSummary, 0, len(dbArtifacts))
	resArtifactMetadataList := make([]response.ArtifactMetadata, 0, len(dbArtifacts))

	for _, dbArtifact := range dbArtifacts {
		dbSummary, dbDetails, err := getTestSummaryAndDetails(dbArtifact.Data, dbArtifact.ArtifactID, buildID)
		if err != nil {
			log.Warn().
				WithError(err).
				WithString("filename", dbArtifact.FileName).
				WithUint("build", buildID).
				WithUint("artifact", dbArtifact.ArtifactID).
				Message("Failed to unmarshal; invalid/unsupported TRX/XML format.")

			ginutil.WriteProblemError(c, err,
				problem.Response{
					Type:   "/prob/api/test-results-parse",
					Status: http.StatusBadRequest,
					Title:  "Unexpected response format.",
					Detail: fmt.Sprintf(
						"Failed parsing test result ID %d, for build with ID %d in"+
							" database. Invalid/unsupported TRX/XML format.", dbSummary.ArtifactID, buildID),
				})
			return
		}

		dbSummary.FileName = dbArtifact.FileName
		dbAllSummaries = append(dbAllSummaries, dbSummary)
		dbAllDetails = append(dbAllDetails, dbDetails...)

		resArtifactMetadataList = append(resArtifactMetadataList, response.ArtifactMetadata{
			FileName:   dbArtifact.FileName,
			ArtifactID: dbArtifact.ArtifactID,
		})
	}

	if err := m.Database.CreateInBatches(dbAllSummaries, 10).Error; err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed saving test result summaries for build with ID %d in database.",
			buildID))
		return
	}

	err = m.Database.
		CreateInBatches(dbAllDetails, 100).
		Error
	if err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed saving test result details for build with ID %d in database.",
			buildID))
	}

	c.JSON(http.StatusOK, resArtifactMetadataList)
}

// getBuildAllTestResultDetailListHandler godoc
// @id getBuildAllTestResultDetailList
// @summary Get all test result details for specified build
// @tags test-result
// @param buildId path int true "Build ID"
// @success 200 {object} []response.TestResultDetail
// @failure 400 {object} problem.Response "Bad request"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildId}/test-result/detail [get]
func (m buildTestResultModule) getBuildAllTestResultDetailListHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildId")
	if !ok {
		return
	}

	var dbDetails []database.TestResultDetail
	err := m.Database.
		Where(&database.TestResultDetail{BuildID: buildID}).
		Find(&dbDetails).
		Error

	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching test result details for build with ID %d from database.",
			buildID))
		return
	}

	resDetails := dbTestResultDetailsToResponses(dbDetails)
	c.JSON(http.StatusOK, resDetails)
}

// getBuildAllTestResultSummaryListHandler godoc
// @id getBuildAllTestResultSummaryList
// @summary Get all test result summaries for specified build
// @tags test-result
// @param buildId path int true "Build ID"
// @success 200 {object} []response.TestResultSummary
// @failure 400 {object} problem.Response "Bad Request"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildId}/test-result/summary [get]
func (m buildTestResultModule) getBuildAllTestResultSummaryListHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildId")
	if !ok {
		return
	}

	var dbSummaries []database.TestResultSummary
	err := m.Database.
		Where(&database.TestResultSummary{BuildID: buildID}).
		Find(&dbSummaries).
		Error

	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching test result summaries from build with ID %d from database.",
			buildID))
		return
	}

	resSummaries := make([]response.TestResultSummary, len(dbSummaries))
	for i, dbSummary := range dbSummaries {
		resSummaries[i] = dbTestResultSummaryToResponse(dbSummary)
	}

	c.JSON(http.StatusOK, resSummaries)
}

// getBuildTestResultSummaryHandler godoc
// @id getBuildTestResultSummary
// @summary Get test result summary for specified test
// @tags test-result
// @param buildId path int true "Build ID"
// @param artifactId path int true "Artifact ID"
// @success 200 {object} response.TestResultSummary
// @failure 400 {object} problem.Response "Bad Request"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildId}/test-result/summary/{artifactId} [get]
func (m buildTestResultModule) getBuildTestResultSummaryHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildId")
	if !ok {
		return
	}

	artifactID, ok := ginutil.ParseParamUint(c, "artifactId")
	if !ok {
		return
	}

	var dbSummary database.TestResultSummary
	err := m.Database.
		Where(&database.TestResultSummary{BuildID: buildID, ArtifactID: artifactID}).
		Find(&dbSummary).
		Error

	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching test result summary from test with ID %d for build with ID %d from database.",
			artifactID, buildID))
		return
	}

	resSummary := dbTestResultSummaryToResponse(dbSummary)
	c.JSON(http.StatusOK, resSummary)
}

// getBuildTestResultDetailListHandler godoc
// @id getBuildTestResultDetailList
// @summary Get all test result details for specified test
// @tags test-result
// @param buildId path int true "Build ID"
// @param artifactId path int true "Artifact ID"
// @success 200 {object} []response.TestResultDetail
// @failure 400 {object} problem.Response "Bad Request"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildId}/test-result/summary/{artifactId}/detail [get]
func (m buildTestResultModule) getBuildTestResultDetailListHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildId")
	if !ok {
		return
	}

	artifactID, ok := ginutil.ParseParamUint(c, "artifactId")
	if !ok {
		return
	}

	var dbDetails []database.TestResultDetail
	err := m.Database.
		Where(&database.TestResultDetail{BuildID: buildID, ArtifactID: artifactID}).
		Find(&dbDetails).
		Error

	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching test result details from test with ID %d for build with ID %d from database.",
			artifactID, buildID))
		return
	}

	resDetails := dbTestResultDetailsToResponses(dbDetails)
	c.JSON(http.StatusOK, resDetails)
}

// getBuildAllTestResultListSummaryHandler godoc
// @id getBuildAllTestResultListSummary
// @summary Get test result list summary of all tests for specified build
// @tags test-result
// @param buildId path int true "Build ID"
// @success 200 {object} response.TestResultListSummary
// @failure 400 {object} problem.Response "Bad Request"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildId}/test-result/list-summary [get]
func (m buildTestResultModule) getBuildAllTestResultListSummaryHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildId")
	if !ok {
		return
	}

	var dbListSummary struct {
		Failed  uint
		Passed  uint
		Skipped uint
	}

	err := m.Database.
		Model(&TestResultSummary{}).
		Where(&TestResultSummary{BuildID: buildID}).
		Select("sum(failed) as Failed, sum(passed) as Passed, sum(skipped) as Skipped").
		Scan(&dbListSummary).
		Error

	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching test result summaries for build with ID %d from database.",
			buildID))
		return
	}

	resListSummary := response.TestResultListSummary{
		BuildID: buildID,
		Total:   dbListSummary.Failed + dbListSummary.Passed + dbListSummary.Skipped,
		Passed:  dbListSummary.Passed,
		Skipped: dbListSummary.Skipped,
		Failed:  dbListSummary.Failed,
	}

	c.JSON(http.StatusOK, resListSummary)
}

type xmlInnerString struct {
	InnerXML string `xml:",innerxml"`
}

type trxTestRun struct {
	XMLName xml.Name `xml:"TestRun"`
	Results struct {
		UnitTestResults []struct {
			TestName  string `xml:"testName,attr"`
			Duration  string `xml:"duration,attr"`
			StartTime string `xml:"startTime,attr"`
			EndTime   string `xml:"endTime,attr"`
			Outcome   string `xml:"outcome,attr"`
			Output    struct {
				ErrorInfo struct {
					Message    xmlInnerString `xml:"Message"`
					StackTrace xmlInnerString `xml:"StackTrace"`
				} `xml:"ErrorInfo"`
			} `xml:"Output"`
		} `xml:"UnitTestResult"`
	} `xml:"Results"`

	ResultSummary struct {
		Counters struct {
			Total               uint `xml:"total,attr"`
			Executed            uint `xml:"executed,attr"`
			Passed              uint `xml:"passed,attr"`
			Failed              uint `xml:"failed,attr"`
			Error               uint `xml:"error,attr"`
			Timeout             uint `xml:"timeout,attr"`
			Aborted             uint `xml:"aborted,attr"`
			Inconclusive        uint `xml:"inconclusive,attr"`
			PassedButRunAborted uint `xml:"passedButRunAborted,attr"`
			NotRunnable         uint `xml:"notRunnable,attr"`
			NotExecuted         uint `xml:"notExecuted,attr"`
			Disconnected        uint `xml:"disconnected,attr"`
			Warning             uint `xml:"warning,attr"`
			Completed           uint `xml:"completed,attr"`
			InProgress          uint `xml:"inProgress,attr"`
			Pending             uint `xml:"pending,attr"`
		} `xml:"Counters"`
	} `xml:"ResultSummary"`
}

func getTestSummaryAndDetails(data []byte, artifactID, buildID uint) (database.TestResultSummary, []database.TestResultDetail, error) {
	var testRun trxTestRun
	if err := xml.Unmarshal(data, &testRun); err != nil {
		return database.TestResultSummary{}, nil, err
	}

	dbDetails := make([]database.TestResultDetail, len(testRun.Results.UnitTestResults))
	for idx, utr := range testRun.Results.UnitTestResults {
		detail := &dbDetails[idx]
		detail.ArtifactID = artifactID
		detail.BuildID = buildID
		detail.Name = utr.TestName
		if utr.Outcome == "Passed" {
			detail.Status = database.TestResultStatusSuccess
		} else if utr.Outcome == "Failed" {
			detail.Status = database.TestResultStatusFailed
		} else if utr.Outcome == "NotExecuted" {
			detail.Status = database.TestResultStatusSkipped
		}
		if detail.Status != database.TestResultStatusSuccess {
			detail.Message.SetValid(fmt.Sprintf("%s\n%s",
				utr.Output.ErrorInfo.Message.InnerXML,
				utr.Output.ErrorInfo.StackTrace.InnerXML))
		}

		parseTimeFailedEvent := log.Warn().
			WithUint("build", buildID).
			WithUint("artifact", artifactID).
			WithString("test", detail.Name)

		startTime, err := time.Parse(time.RFC3339, utr.StartTime)
		if err != nil {
			parseTimeFailedEvent.
				WithError(err).
				Message("Failed to parse StartTime for test.")
		} else {
			detail.StartedOn.SetValid(startTime)
		}

		endTime, err := time.Parse(time.RFC3339, utr.EndTime)
		if err != nil {
			parseTimeFailedEvent.
				WithError(err).
				Message("Failed to parse CompletedOn for test.")
		} else {
			detail.CompletedOn.SetValid(endTime)
		}
	}

	counters := testRun.ResultSummary.Counters
	dbSummary := database.TestResultSummary{
		ArtifactID: artifactID,
		BuildID:    buildID,
		Failed:     counters.Failed,
		Passed:     counters.Passed,
		Skipped:    counters.NotExecuted,
		Total:      counters.Total,
	}

	return dbSummary, dbDetails, nil
}

func dbTestResultSummariesToResponses(dbSummaries []database.TestResultSummary) []response.TestResultSummary {
	resSummaries := make([]response.TestResultSummary, len(dbSummaries))
	for i, dbSummary := range dbSummaries {
		resSummaries[i] = dbTestResultSummaryToResponse(dbSummary)
	}
	return resSummaries
}

func dbTestResultSummaryToResponse(dbSummary database.TestResultSummary) response.TestResultSummary {
	return response.TestResultSummary{
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

func dbTestResultDetailsToResponses(dbDetails []database.TestResultDetail) []response.TestResultDetail {
	resDetails := make([]response.TestResultDetail, len(dbDetails))
	for i, dbDetail := range dbDetails {
		resDetails[i] = dbTestResultDetailToResponse(dbDetail)
	}
	return resDetails
}

func dbTestResultDetailToResponse(dbDetail database.TestResultDetail) response.TestResultDetail {
	return response.TestResultDetail{
		TestResultDetailID: dbDetail.TestResultDetailID,
		ArtifactID:         dbDetail.ArtifactID,
		BuildID:            dbDetail.BuildID,
		Name:               dbDetail.Name,
		Message:            dbDetail.Message,
		StartedOn:          dbDetail.StartedOn,
		CompletedOn:        dbDetail.CompletedOn,
		Status:             dbTestResultStatusToResponse(dbDetail.Status),
	}
}

func dbTestResultStatusToResponse(dbStatus database.TestResultStatus) response.TestResultStatus {
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

package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/internal/ctxparser"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"github.com/iver-wharf/wharf-core/pkg/problem"
	"gorm.io/gorm"
)

type testResultModule struct {
	Database *gorm.DB
}

func (m testResultModule) Register(g *gin.RouterGroup) {
	g.POST("/test-result-data", m.postTestResultDataHandler)
	g.GET("/test-result-details", m.getBuildAllTestResultDetailsHandler)
	g.GET("/test-result-details/:testResultDetailId", m.getBuildTestResultDetailsHandler)
	g.GET("/test-results-summary", m.getBuildTestResultsSummaryHandler)
}

// postTestResultDataHandler godoc
// @summary Post test result data
// @tags artifact
// @accept multipart/form-data
// @param buildid path int true "Build ID"
// @param file formData file true "Test result file"
// @success 201 {object} TestResultListSummary "Added new test result data and created summary"
// @failure 400 {object} problem.Response "Bad request"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/test-result-data [post]
func (m testResultModule) postTestResultDataHandler(c *gin.Context) {
	buildID, files, ok := ctxparser.ParamBuildIDAndFiles(c)
	if !ok {
		return
	}

	artifacts, ok := createArtifacts(c, m.Database, files, buildID)
	if !ok {
		return
	}

	summaries := make([]TestResultSummary, 0, len(artifacts))
	lotsOfDetails := make([]TestResultDetail, 0)

	listSummary := TestResultListSummary{
		BuildID: buildID}
	for _, artifact := range artifacts {
		summary, details, err := getTestSummaryAndDetails(artifact.Data, artifact.ArtifactID, buildID)
		if err != nil {
			log.Warn().
				WithError(err).
				WithString("filename", artifact.FileName).
				WithUint("build", buildID).
				WithUint("artifact", artifact.ArtifactID).
				Message("Failed to unmarshal; invalid/unsupported TRX/XML format.")

			ginutil.WriteProblemError(c, err,
				problem.Response{
					Type:   "/prob/api/test-results-parse",
					Status: http.StatusBadGateway,
					Title:  "Unexpected response format.",
					Detail: fmt.Sprintf(
						"Failed parsing test result ID %d, for build with ID %d in"+
							" database. Invalid/unsupported TRX/XML format.", summary.ArtifactID, buildID),
				})
			return
		}

		listSummary.Failed += summary.Failed
		listSummary.Passed += summary.Passed
		listSummary.Skipped += summary.Skipped

		summary.FileName = artifact.FileName
		summaries = append(summaries, summary)
		lotsOfDetails = append(lotsOfDetails, details...)
	}

	listSummary.Total = listSummary.Failed + listSummary.Passed + listSummary.Skipped

	if err := m.Database.CreateInBatches(summaries, 10).Error; err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed saving test result summaries for build with ID %d in database.",
			buildID))
		return
	}

	err := m.Database.
		CreateInBatches(lotsOfDetails, 100).
		Error
	if err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed saving test result details for build with ID %d in database.",
			buildID))
	}

	c.JSON(http.StatusOK, listSummary)
}

// getBuildAllTestResultDetailsHandler godoc
// @summary Get all test result details for specified build
// @tags artifact
// @param buildid path int true "Build ID"
// @success 200 {object} []TestResultDetail
// @failure 400 {object} problem.Response "Bad request"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/test-result-details [get]
func (m testResultModule) getBuildAllTestResultDetailsHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}

	details := []TestResultDetail{}
	err := m.Database.
		Where(&TestResultDetail{BuildID: buildID}).
		Find(&details).
		Error

	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching test result details for build with ID %d from database.",
			buildID))
		return
	}

	c.JSON(http.StatusOK, details)
}

// getBuildTestResultDetailsHandler godoc
// @summary Get test result details for specified test
// @tags artifact
// @param buildid path int true "Build ID"
// @param artifactId path int true "Artifact ID"
// @success 200 {object} []TestResultDetail
// @router /build/{buildid}/test-result-details/{artifactId} [get]
func (m testResultModule) getBuildTestResultDetailsHandler(c *gin.Context) {
	artifactID, buildID, ok := ctxparser.ParamArtifactAndBuildID(c)
	if !ok {
		return
	}

	details := []TestResultDetail{}
	err := m.Database.
		Where(&TestResultDetail{BuildID: buildID, ArtifactID: artifactID}).
		Find(&details).
		Error

	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching test result details from test with ID %d for build with ID %d from database.",
			artifactID, buildID))
		return
	}

	c.JSON(http.StatusOK, details)
}

// getBuildTestResultsSummaryHandler godoc
// @summary Get test result summary of all tests for specified build
// @tags artifact
// @param buildid path int true "Build ID"
// @success 200 {object} TestResultListSummary
// @failure 400 {object} problem.Response "Bad Request"
// @failure 502 {object} problem.Response "Bad Gateway"
// @router /build/{buildid}/test-results-summary [get]
func (m testResultModule) getBuildTestResultsSummaryHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}

	summaries := []TestResultSummary{}
	err := m.Database.
		Preload("Artifact").
		Where(&TestResultSummary{BuildID: buildID}).
		Find(&summaries).
		Error

	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching test result summaries for build with ID %d from database.",
			buildID))
		return
	}

	listSummary := TestResultListSummary{
		BuildID: buildID}

	for _, v := range summaries {
		listSummary.Failed += v.Failed
		listSummary.Passed += v.Passed
		listSummary.Skipped += v.Skipped
	}

	listSummary.Total = listSummary.Failed + listSummary.Passed + listSummary.Skipped

	c.JSON(http.StatusOK, listSummary)
}

type trxTestRun struct {
	XMLName xml.Name `xml:"TestRun"`

	Results struct {
		XMLName         xml.Name `xml:"Results"`
		UnitTestResults []struct {
			XMLName   xml.Name `xml:"UnitTestResult"`
			TestName  string   `xml:"testName,attr"`
			Duration  string   `xml:"duration,attr"`
			StartTime string   `xml:"startTime,attr"`
			EndTime   string   `xml:"endTime,attr"`
			Outcome   string   `xml:"outcome,attr"`
			Output    struct {
				XMLName   xml.Name `xml:"Output"`
				ErrorInfo struct {
					XMLName xml.Name `xml:"ErrorInfo"`
					Message struct {
						InnerXML string `xml:",innerxml"`
					} `xml:"Message"`
					StackTrace struct {
						InnerXML string `xml:",innerxml"`
					} `xml:"StackTrace"`
				} `xml:"ErrorInfo"`
			} `xml:"Output"`
		} `xml:"UnitTestResult"`
	} `xml:"Results"`

	ResultSummary struct {
		XMLName  xml.Name `xml:"ResultSummary"`
		Counters struct {
			XMLName             xml.Name `xml:"Counters"`
			Total               uint     `xml:"total,attr"`
			Executed            uint     `xml:"executed,attr"`
			Passed              uint     `xml:"passed,attr"`
			Failed              uint     `xml:"failed,attr"`
			Error               uint     `xml:"error,attr"`
			Timeout             uint     `xml:"timeout,attr"`
			Aborted             uint     `xml:"aborted,attr"`
			Inconclusive        uint     `xml:"inconclusive,attr"`
			PassedButRunAborted uint     `xml:"passedButRunAborted,attr"`
			NotRunnable         uint     `xml:"notRunnable,attr"`
			NotExecuted         uint     `xml:"notExecuted,attr"`
			Disconnected        uint     `xml:"disconnected,attr"`
			Warning             uint     `xml:"warning,attr"`
			Completed           uint     `xml:"completed,attr"`
			InProgress          uint     `xml:"inProgress,attr"`
			Pending             uint     `xml:"pending,attr"`
		} `xml:"Counters"`
	} `xml:"ResultSummary"`
}

// getTestSummaryAndDetails currently only supports the TRX/XML format.
func getTestSummaryAndDetails(data []byte, artifactID, buildID uint) (TestResultSummary, []TestResultDetail, error) {
	var testRun trxTestRun
	if err := xml.Unmarshal(data, &testRun); err != nil {
		return TestResultSummary{}, []TestResultDetail{}, err
	}

	details := make([]TestResultDetail, len(testRun.Results.UnitTestResults))
	for idx, utr := range testRun.Results.UnitTestResults {
		detail := &details[idx]
		detail.ArtifactID = artifactID
		detail.BuildID = buildID
		detail.Name = utr.TestName
		if utr.Outcome == "Passed" {
			detail.Status = TestResultStatusSuccess
		} else if utr.Outcome == "Failed" {
			detail.Status = TestResultStatusFailed
		} else if utr.Outcome == "NotExecuted" {
			detail.Status = TestResultStatusSkipped
		}
		if detail.Status != TestResultStatusSuccess {
			detail.Message.SetValid(fmt.Sprintf("%s\n%s",
				utr.Output.ErrorInfo.Message.InnerXML,
				utr.Output.ErrorInfo.StackTrace.InnerXML))
		}

		if startTime, err := time.Parse(time.RFC3339, utr.StartTime); err == nil {
			detail.StartedOn.SetValid(startTime)
		}

		if endTime, err := time.Parse(time.RFC3339, utr.EndTime); err == nil {
			detail.CompletedOn.SetValid(endTime)
		}
	}

	counters := testRun.ResultSummary.Counters
	summary := TestResultSummary{
		ArtifactID: artifactID,
		BuildID:    buildID,
		Failed:     counters.Failed,
		Passed:     counters.Passed,
		Skipped:    counters.NotExecuted,
		Total:      counters.Total,
	}

	return summary, details, nil
}

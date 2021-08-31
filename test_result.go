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

type buildTestResultModule struct {
	Database *gorm.DB
}

// TestResultListSummary contains data about several test result files.
type TestResultListSummary struct {
	BuildID uint `json:"buildId"`
	Total   uint `json:"total"`
	Failed  uint `json:"failed"`
	Passed  uint `json:"passed"`
	Skipped uint `json:"skipped"`
}

// ArtifactList contains a slice of artifact file names and a field for how many
// entries there are. Used as response model when uploading test result data
// been uploaded.
type ArtifactList struct {
	Entries []artifactListItem `json:"entries"`
	Count   uint               `json:"count"`
}

type artifactListItem struct {
	FileName   string `json:"fileName"`
	ArtifactID uint   `json:"artifactId"`
}

func (m buildTestResultModule) Register(r gin.IRouter) {
	testResult := r.Group("/test-result")
	{
		testResult.POST("/", m.postBuildTestResultDataHandler)

		testResult.GET("/detail", m.getBuildAllTestResultDetailsHandler)

		testResult.GET("/summary", m.getBuildAllTestResultSummariesHandler)
		testResult.GET(
			fmt.Sprintf("/summary/:%s", ctxparser.ArtifactIDParamName),
			m.getBuildTestResultSummaryHandler)

		testResult.GET(
			fmt.Sprintf("/summary/:%s/detail", ctxparser.ArtifactIDParamName),
			m.getBuildTestResultDetailsHandler)

		testResult.GET("/list-summary", m.getBuildTestResultListSummaryHandler)
	}
}

// postBuildTestResultDataHandler godoc
// @summary Post test result data
// @tags test-result
// @accept multipart/form-data
// @param buildid path int true "Build ID"
// @param file formData file true "Test result file"
// @success 201 {object} ArtifactList "Added new test result data and created summaries"
// @failure 400 {object} problem.Response "Bad request"
// @failure 502 {object} problem.Response "Database unreachable or bad gateway"
// @router /build/{buildid}/test-result [post]
func (m buildTestResultModule) postBuildTestResultDataHandler(c *gin.Context) {
	p := ctxparser.ParamSetBuildID{}
	if ok := ctxparser.ShouldBindUri(c, &p); !ok {
		return
	}

	files, ok := ctxparser.ParseMultipartFormData(c, p.BuildID)
	if !ok {
		return
	}

	artifacts, ok := createArtifacts(c, m.Database, files, p.BuildID)
	if !ok {
		return
	}

	artifactList := ArtifactList{}

	summaries := make([]TestResultSummary, 0, len(artifacts))
	lotsOfDetails := make([]TestResultDetail, 0)

	for _, artifact := range artifacts {
		summary, details, err := getTestSummaryAndDetails(artifact.Data, artifact.ArtifactID, p.BuildID)
		if err != nil {
			log.Warn().
				WithError(err).
				WithString("filename", artifact.FileName).
				WithUint("build", p.BuildID).
				WithUint("artifact", artifact.ArtifactID).
				Message("Failed to unmarshal; invalid/unsupported TRX/XML format.")

			ginutil.WriteProblemError(c, err,
				problem.Response{
					Type:   "/prob/api/test-results-parse",
					Status: http.StatusBadGateway,
					Title:  "Unexpected response format.",
					Detail: fmt.Sprintf(
						"Failed parsing test result ID %d, for build with ID %d in"+
							" database. Invalid/unsupported TRX/XML format.", summary.ArtifactID, p.BuildID),
				})
			return
		}

		summary.FileName = artifact.FileName
		summaries = append(summaries, summary)
		lotsOfDetails = append(lotsOfDetails, details...)

		artifactList.Entries = append(artifactList.Entries, artifactListItem{
			FileName:   artifact.FileName,
			ArtifactID: artifact.ArtifactID,
		})
	}

	if err := m.Database.CreateInBatches(summaries, 10).Error; err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed saving test result summaries for build with ID %d in database.",
			p.BuildID))
		return
	}

	if err := m.Database.CreateInBatches(lotsOfDetails, 100).Error; err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed saving test result details for build with ID %d in database.",
			p.BuildID))
		return
	}

	artifactList.Count = uint(len(artifactList.Entries))

	c.JSON(http.StatusOK, artifactList)
}

// getBuildAllTestResultDetailsHandler godoc
// @summary Get all test result details for specified build
// @tags test-result
// @param buildid path int true "Build ID"
// @success 200 {object} []TestResultDetail
// @failure 400 {object} problem.Response "Bad request"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/test-result/detail [get]
func (m buildTestResultModule) getBuildAllTestResultDetailsHandler(c *gin.Context) {
	p := ctxparser.ParamSetBuildID{}
	if ok := ctxparser.ShouldBindUri(c, &p); !ok {
		return
	}

	details := []TestResultDetail{}
	err := m.Database.
		Where(&TestResultDetail{BuildID: p.BuildID}).
		Find(&details).
		Error

	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching test result details for build with ID %d from database.",
			p.BuildID))
		return
	}

	c.JSON(http.StatusOK, details)
}

// getBuildAllTestResultSummariesHandler godoc
// @summary Get all test result summaries for specified build
// @tags test-result
// @param buildid path int true "Build ID"
// @success 200 {object} []TestResultSummary
// @failure 400 {object} problem.Response "Bad Request"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/test-result/summary [get]
func (m buildTestResultModule) getBuildAllTestResultSummariesHandler(c *gin.Context) {
	p := ctxparser.ParamSetBuildID{}
	if ok := ctxparser.ShouldBindUri(c, &p); !ok {
		return
	}

	summaries := []TestResultSummary{}
	err := m.Database.
		Where(&TestResultSummary{BuildID: p.BuildID}).
		Find(&summaries).
		Error

	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching test result summaries from build with ID %d from database.",
			p.BuildID))
		return
	}

	c.JSON(http.StatusOK, summaries)
}

// getBuildTestResultSummaryHandler godoc
// @summary Get test result summary for specified test
// @tags test-result
// @param buildid path int true "Build ID"
// @param artifactId path int true "Artifact ID"
// @success 200 {object} TestResultSummary
// @failure 400 {object} problem.Response "Bad Request"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/test-result/summary/{artifactId} [get]
func (m buildTestResultModule) getBuildTestResultSummaryHandler(c *gin.Context) {
	p := struct {
		ctxparser.ParamSetArtifactID
		ctxparser.ParamSetBuildID
	}{}
	if ok := ctxparser.ShouldBindUri(c, &p); !ok {
		return
	}

	summary := TestResultSummary{}
	err := m.Database.
		Where(&TestResultSummary{BuildID: p.BuildID, ArtifactID: p.ArtifactID}).
		Find(&summary).
		Error

	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching test result summary from test with ID %d for build with ID %d from database.",
			p.ArtifactID, p.BuildID))
		return
	}

	c.JSON(http.StatusOK, summary)
}

// getBuildTestResultDetailsHandler godoc
// @summary Get all test result details for specified test
// @tags test-result
// @param buildid path int true "Build ID"
// @param artifactId path int true "Artifact ID"
// @success 200 {object} []TestResultDetail
// @failure 400 {object} problem.Response "Bad Request"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/test-result/summary/{artifactId}/detail [get]
func (m buildTestResultModule) getBuildTestResultDetailsHandler(c *gin.Context) {
	p := struct {
		ctxparser.ParamSetArtifactID
		ctxparser.ParamSetBuildID
	}{}
	if ok := ctxparser.ShouldBindUri(c, &p); !ok {
		return
	}

	details := []TestResultDetail{}
	err := m.Database.
		Where(&TestResultDetail{BuildID: p.BuildID, ArtifactID: p.ArtifactID}).
		Find(&details).
		Error

	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching test result details from test with ID %d for build with ID %d from database.",
			p.ArtifactID, p.BuildID))
		return
	}

	c.JSON(http.StatusOK, details)
}

// getBuildTestResultListSummaryHandler godoc
// @summary Get test result list summary of all tests for specified build
// @tags test-result
// @param buildid path int true "Build ID"
// @success 200 {object} TestResultListSummary
// @failure 400 {object} problem.Response "Bad Request"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/test-result/list-summary [get]
func (m buildTestResultModule) getBuildTestResultListSummaryHandler(c *gin.Context) {
	p := ctxparser.ParamSetBuildID{}
	if ok := ctxparser.ShouldBindUri(c, &p); !ok {
		return
	}

	listSummary := TestResultListSummary{BuildID: p.BuildID}

	err := m.Database.
		Model(&TestResultSummary{}).
		Select("sum(failed) as Failed, sum(passed) as Passed, sum(skipped) as Skipped").
		Where(&listSummary).
		Scan(&listSummary).
		Error

	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching test result summaries for build with ID %d from database.",
			p.BuildID))
		return
	}

	listSummary.Total = listSummary.Failed + listSummary.Passed + listSummary.Skipped

	c.JSON(http.StatusOK, listSummary)
}

type xmlInnerString struct {
	InnerXML string `xml:",innerxml"`
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
					XMLName    xml.Name       `xml:"ErrorInfo"`
					Message    xmlInnerString `xml:"Message"`
					StackTrace xmlInnerString `xml:"StackTrace"`
				}
			}
		}
	}

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
		}
	}
}

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

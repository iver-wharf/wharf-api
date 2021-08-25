package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"github.com/iver-wharf/wharf-core/pkg/problem"
	"gorm.io/gorm"
)

// TestResultListSummary contains data about several test result files.
type TestResultListSummary struct {
	BuildID   uint                `json:"buildId"`
	Total     uint                `json:"total"`
	Failed    uint                `json:"failed"`
	Passed    uint                `json:"passed"`
	Skipped   uint                `json:"skipped"`
	Summaries []TestResultSummary `json:"summaries"`
}

type artifactModule struct {
	Database *gorm.DB
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

// TestsResults holds how many builds has passed and failed. A test result has
// the status of "Failed" if there are any failed tests, "Success" if there are
// any passing tests and no failed tests, and "No tests" if there are no failed
// nor passing tests.
type TestsResults struct {
	Passed uint       `json:"passed"`
	Failed uint       `json:"failed"`
	Status TestStatus `json:"status" enums:"Success,Failed,No tests"`
}

func (m artifactModule) Register(g *gin.RouterGroup) {
	g.GET("/artifacts", m.getBuildArtifactsHandler)
	g.GET("/artifact/:artifactId", m.getBuildArtifactHandler)
	g.POST("/artifact", m.postBuildArtifactHandler)
	g.PUT("/test-result-data", m.postTestResultDataHandler)
	g.GET("/test-result-details", m.getBuildAllTestResultDetailsHandler)
	g.GET("/test-result-details/:artifactId", m.getBuildTestResultDetailsHandler)
	g.GET("/test-results-summary", m.getBuildTestResultsSummaryHandler)
	// deprecated
	g.GET("/tests-results", m.getBuildTestsResultsHandler)
}

// getBuildArtifactsHandler godoc
// @summary Get list of build artifacts
// @tags artifact
// @param buildid path int true "Build ID"
// @success 200 {array} Artifact
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/artifacts [get]
func (m artifactModule) getBuildArtifactsHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}

	artifacts := []Artifact{}
	err := m.Database.
		Where(&Artifact{BuildID: uint(buildID)}).
		Find(&artifacts).
		Error
	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching artifacts for build with ID %d from database.",
			buildID))
		return
	}

	c.JSON(http.StatusOK, artifacts)
}

// getBuildArtifactHandler godoc
// @summary Get build artifact
// @tags artifact
// @param buildid path int true "Build ID"
// @param artifactId path int true "Artifact ID"
// @success 200 {file} string "OK"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Artifact not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/artifact/{artifactId} [get]
func (m artifactModule) getBuildArtifactHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}
	artifactID, ok := ginutil.ParseParamUint(c, "artifactId")
	if !ok {
		return
	}

	var artifact Artifact
	err := m.Database.
		Where(&Artifact{
			BuildID:    buildID,
			ArtifactID: artifactID}).
		Order(artifactColumnArtifactID + " DESC").
		First(&artifact).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ginutil.WriteDBNotFound(c, fmt.Sprintf(
			"Artifact with ID %d was not found on build with ID %d.",
			artifactID, buildID))
		return
	} else if err != nil {
		ginutil.WriteBodyReadError(c, err, fmt.Sprintf(
			"Failed fetching artifact with ID %d on build with ID %d.",
			artifactID, buildID))
		return
	}

	extension := filepath.Ext(artifact.FileName)
	mimeType := mime.TypeByExtension(extension)
	disposition := fmt.Sprintf("attachment; filename=\"%s\"", artifact.FileName)

	c.Header("Content-Disposition", disposition)
	c.Data(http.StatusOK, mimeType, artifact.Data)
}

// getBuildTestsResultsHandler godoc
// @deprecated
// @summary Get build tests results from .trx files. Deprecated, /build/{buildid}/test-results-summary should be used instead.
// @tags artifact
// @param buildid path int true "Build ID"
// @success 200 {object} testsResults
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/tests-results [get]
func (m artifactModule) getBuildTestsResultsHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}

	testRunFiles := []Artifact{}

	err := m.Database.
		Where(&Artifact{BuildID: uint(buildID)}).
		Where(artifactColumnFileName+" LIKE ?", "%.trx").
		Find(&testRunFiles).
		Error
	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching test result artifacts for build with ID %d from database.",
			buildID))
		return
	}

	var results TestsResults
	var run testRun

	for _, testRunFile := range testRunFiles {
		xml.Unmarshal(testRunFile.Data, &run)
		results.Passed += run.ResultSummary.Counters.Passed
		results.Failed += run.ResultSummary.Counters.Failed
	}

	if results.Failed == 0 && results.Passed == 0 {
		results.Status = TestStatusNoTests
	} else if results.Failed == 0 {
		results.Status = TestStatusSuccess
	} else {
		results.Status = TestStatusFailed
	}

	c.JSON(http.StatusOK, results)
}

// postBuildArtifactHandler godoc
// @summary Post build artifact
// @tags artifact
// @accept multipart/form-data
// @param buildid path int true "Build ID"
// @param file formData file true "Build artifact file"
// @success 201 {object} string "Added new artifacts"
// @failure 400 {object} problem.Response "Bad request"
// @failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @failure 404 {object} problem.Response "Artifact not found"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/artifact [post]
func (m artifactModule) postBuildArtifactHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}
	files, ok := parseMultipartFormData(c, buildID)
	if !ok {
		return
	}
	_, ok = m.createArtifacts(c, files, buildID)
	if !ok {
		return
	}
	c.Status(http.StatusCreated)
}

type file struct {
	name     string
	fileName string
	data     []byte
}

// postTestResultDataHandler godoc
// @summary Post test result data
// @tags artifact
// @accept multipart/form-data
// @param buildid path int true "Build ID"
// @param file formData file true "Test result artifact file"
// @success 201 {object} TestResultListSummary "Added new test result data and created summary"
// @failure 400 {object} problem.Response "Bad request"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/test-result-data [post]
func (m artifactModule) postTestResultDataHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}

	files, ok := parseMultipartFormData(c, buildID)
	if !ok {
		return
	}

	artifacts, ok := m.createArtifacts(c, files, buildID)
	if !ok {
		return
	}

	summaries := make([]TestResultSummary, 0, len(artifacts))
	lotsOfDetails := make([]TestResultDetail, 0)

	for _, artifact := range artifacts {
		summary, details, err := getTestSummaryAndDetails(artifact.Data, artifact.ArtifactID, buildID)
		if err != nil {
			logEvent := log.Warn().
				WithError(err).
				WithString("filename", artifact.Name).
				WithUint("build", buildID).
				WithUint("artifact", artifact.ArtifactID)
			if syntaxErr := err.(*xml.SyntaxError); syntaxErr != nil || strings.HasPrefix(err.Error(), "xml:") {
				logEvent.Message("Failed to unmarshal; invalid/unsupported TRX/XML format.")

				ginutil.WriteProblemError(c, syntaxErr,
					problem.Response{
						Type:   "/prob/unexpected-response-format",
						Status: http.StatusBadGateway,
						Title:  "Unexpected response format.",
						Detail: fmt.Sprintf(
							"Failed parsing test result ID %d, for build with ID %d in"+
								" database. Invalid/unsupported TRX/XML format.", summary.ArtifactID, buildID),
					})
			} else {
				logEvent.Message("Failed to unmarshal; the structure used to unmarshal might be malformed.")

				ginutil.WriteProblemError(c, syntaxErr,
					problem.Response{
						Type:   "/prob/bad-code",
						Status: http.StatusInternalServerError,
						Title:  "Bad code.",
						Detail: fmt.Sprintf(
							"Failed parsing test result ID %d, for build with ID %d in database. The structure"+
								" used to unmarshal the data in the wharf-api might be malformed. This really shouldn't happen.",
							summary.ArtifactID, buildID),
					})
			}
			return
		}

		summary.FileName = artifact.FileName
		summaries = append(summaries, summary)
		lotsOfDetails = append(lotsOfDetails, details...)
	}

	summaryList := TestResultListSummary{
		Summaries: summaries,
		BuildID:   buildID}
	for _, summary := range summaryList.Summaries {
		err := m.Database.
			Create(summary).
			Error
		if err != nil {
			ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
				"Failed saving test result summary from artifact with ID %d, for build"+
					" with ID %d in database.",
				summary.ArtifactID, buildID))
			return
		}

		summaryList.Failed += summary.Failed
		summaryList.Passed += summary.Passed
		summaryList.Skipped += summary.Skipped
	}

	summaryList.Total =
		summaryList.Failed +
			summaryList.Passed +
			summaryList.Skipped

	err := m.Database.
		CreateInBatches(lotsOfDetails, 100).
		Error
	if err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed saving test result details for build with ID %d in database.",
			buildID))
	}

	c.JSON(http.StatusOK, summaryList)
}

// getBuildAllTestResultDetailsHandler godoc
// @summary Get all test result details for specified build
// @tags artifact
// @param buildid path int true "Build ID"
// @success 200 {object} []TestResultDetail
// @failure 400 {object} problem.Response "Bad request"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/test-result-details [get]
func (m artifactModule) getBuildAllTestResultDetailsHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}

	details := []TestResultDetail{}
	err := m.Database.
		Where(&TestResultDetail{BuildID: uint(buildID)}).
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
func (m artifactModule) getBuildTestResultDetailsHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}

	artifactID, ok := ginutil.ParseParamUint(c, "artifactId")
	if !ok {
		return
	}

	details := []TestResultDetail{}
	err := m.Database.
		Where(&TestResultDetail{BuildID: uint(buildID), ArtifactID: artifactID}).
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
func (m artifactModule) getBuildTestResultsSummaryHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, "buildid")
	if !ok {
		return
	}

	summaries := []TestResultSummary{}
	err := m.Database.
		Preload("Artifact").
		Where(&TestResultSummary{BuildID: uint(buildID)}).
		Find(&summaries).
		Error

	if err != nil {
		ginutil.WriteDBReadError(c, err, fmt.Sprintf(
			"Failed fetching test result summaries for build with ID %d from database.",
			buildID))
		return
	}

	summaryList := TestResultListSummary{
		BuildID:   buildID,
		Summaries: summaries}

	for _, v := range summaries {
		summaryList.Failed += v.Failed
		summaryList.Passed += v.Passed
		summaryList.Skipped += v.Skipped
	}

	summaryList.Total = summaryList.Failed + summaryList.Passed + summaryList.Skipped

	c.JSON(http.StatusOK, summaryList)
}

func parseMultipartFormData(c *gin.Context, buildID uint) ([]file, bool) {
	files := make([]file, 0)

	form, err := c.MultipartForm()
	if err != nil {
		ginutil.WriteMultipartFormReadError(c, err,
			fmt.Sprintf("Failed reading multipart-form content from request body when uploading new"+
				" artifact for build with ID %d.", buildID))
		return nil, false
	}

	for k := range form.File {
		if fhs := form.File[k]; len(fhs) > 0 {
			data, ok := readMultipartFileData(c, buildID, fhs[0])
			if !ok {
				return files, false
			}

			files = append(files, file{
				name:     k,
				fileName: fhs[0].Filename,
				data:     data,
			})
		}
	}

	return files, true
}

func readMultipartFileData(c *gin.Context, buildID uint, fh *multipart.FileHeader) ([]byte, bool) {
	if fh == nil {
		return nil, false
	}

	f, err := fh.Open()
	if err != nil {
		ginutil.WriteMultipartFormReadError(c, err,
			fmt.Sprintf("Failed with starting to read file content from multipart form request body when"+
				" uploading new artifact for build with ID %d.", buildID))
		return nil, false
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		ginutil.WriteMultipartFormReadError(c, err,
			fmt.Sprintf("Failed reading file content from multipart form request body when uploading new"+
				" artifact for build with ID %d.", buildID))
		return nil, false
	}
	return data, true
}

type testRun struct {
	XMLName       xml.Name      `xml:"TestRun"`
	Results       results       `xml:"Results"`
	ResultSummary resultSummary `xml:"ResultSummary"`
}

type results struct {
	XMLName         xml.Name         `xml:"Results"`
	UnitTestResults []unitTestResult `xml:"UnitTestResult"`
}

type unitTestResult struct {
	XMLName   xml.Name `xml:"UnitTestResult"`
	TestName  string   `xml:"testName,attr"`
	Duration  string   `xml:"duration,attr"`
	StartTime string   `xml:"startTime,attr"`
	EndTime   string   `xml:"endTime,attr"`
	Outcome   string   `xml:"outcome,attr"`
	Output    output   `xml:"Output"`
}

type output struct {
	XMLName   xml.Name  `xml:"Output"`
	ErrorInfo errorInfo `xml:"ErrorInfo"`
}

type errorInfo struct {
	XMLName    xml.Name   `xml:"ErrorInfo"`
	Message    message    `xml:"Message"`
	StackTrace stackTrace `xml:"StackTrace"`
}

type message struct {
	InnerXML string `xml:",innerxml"`
}

type stackTrace struct {
	InnerXML string `xml:",innerxml"`
}

type resultSummary struct {
	XMLName  xml.Name `xml:"ResultSummary"`
	Counters counters `xml:"Counters"`
}

type counters struct {
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

// getTestSummaryAndDetails currently only supports the TRX/XML format.
func getTestSummaryAndDetails(data []byte, artifactID, buildID uint) (TestResultSummary, []TestResultDetail, error) {
	var myTestRun testRun
	if err := xml.Unmarshal(data, &myTestRun); err != nil {
		return TestResultSummary{}, []TestResultDetail{}, err
	}

	details := make([]TestResultDetail, len(myTestRun.Results.UnitTestResults))
	for idx, utr := range myTestRun.Results.UnitTestResults {
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
			detail.StartedOn = &startTime
		}

		if endTime, err := time.Parse(time.RFC3339, utr.EndTime); err == nil {
			detail.CompletedOn = &endTime
		}
	}

	counters := myTestRun.ResultSummary.Counters
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

func (m artifactModule) createArtifacts(c *gin.Context, files []file, buildID uint) ([]Artifact, bool) {
	artifacts := make([]Artifact, len(files))
	for idx, f := range files {
		artifact := &artifacts[idx]
		artifact.Data = f.data
		artifact.Name = f.name
		artifact.FileName = f.fileName
		artifact.BuildID = buildID

		err := m.Database.Create(artifact).Error
		if err != nil {
			ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
				"Failed saving artifact with name %q for build with ID %d in database.",
				artifact.FileName, buildID))
			return artifacts, false
		}

		log.Debug().
			WithString("filename", artifact.Name).
			WithUint("build", buildID).
			WithUint("artifact", artifact.ArtifactID).
			Message("File saved as artifact")
	}
	return artifacts, true
}

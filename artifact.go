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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"gorm.io/gorm"
)

const (
	buildIDParamName    = "buildid"
	artifactIDParamName = "artifactId"

	estimatedTestDetailsPerFile = 20
)

type ArtifactModule struct {
	Database *gorm.DB
}

// SummaryOfTestResultSummaries contains data about several test result files.
type SummaryOfTestResultSummaries struct {
	BuildID   uint                `json:"buildId"`
	Total     uint                `json:"total"`
	Failed    uint                `json:"failed"`
	Passed    uint                `json:"passed"`
	Skipped   uint                `json:"skipped"`
	Summaries []TestResultSummary `json:"summaries"`
}

func (m ArtifactModule) Register(g *gin.RouterGroup) {
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
func (m ArtifactModule) getBuildArtifactsHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, buildIDParamName)
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
func (m ArtifactModule) getBuildArtifactHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, buildIDParamName)
	if !ok {
		return
	}
	artifactID, ok := ginutil.ParseParamUint(c, artifactIDParamName)
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
func (m ArtifactModule) getBuildTestsResultsHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, buildIDParamName)
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

	var myTestsResults testsResults
	var myTestRun testRun

	for _, testRunFile := range testRunFiles {
		xml.Unmarshal(testRunFile.Data, &myTestRun)
		myTestsResults.Passed += myTestRun.ResultSummary.Counters.Passed
		myTestsResults.Failed += myTestRun.ResultSummary.Counters.Failed
	}

	if myTestsResults.Failed == 0 && myTestsResults.Passed == 0 {
		myTestsResults.Status = testStatusNoTests
	} else if myTestsResults.Failed == 0 {
		myTestsResults.Status = testStatusSuccess
	} else {
		myTestsResults.Status = testStatusFailed
	}

	c.JSON(http.StatusOK, myTestsResults)
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
func (m ArtifactModule) postBuildArtifactHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, buildIDParamName)
	if !ok {
		return
	}
	files, ok := parseMultipartFormData(c)
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
// @success 201 {object} SummaryOfTestResultSummaries "Added new test result data and created summary"
// @failure 400 {object} problem.Response "Bad request"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/test-result-data [post]
func (m ArtifactModule) postTestResultDataHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, buildIDParamName)
	if !ok {
		return
	}

	files, ok := parseMultipartFormData(c)
	if !ok {
		return
	}

	artifacts, ok := m.createArtifacts(c, files, buildID)
	if !ok {
		return
	}

	summaries := make([]TestResultSummary, 0, len(artifacts))
	details := make([]TestResultDetail, 0, estimatedTestDetailsPerFile*len(artifacts))

	for _, artifact := range artifacts {
		detail, summary, err := getTestSummaryAndDetails(artifact.Data, artifact.ArtifactID, buildID)
		if err != nil {
			log.Warn().
				WithError(err).
				WithString("filename", artifact.Name).
				WithUint("build", buildID).
				WithUint("artifact", artifact.ArtifactID).
				Message("Failed to unmarshal; invalid XML format")
			return
		}

		summary.FileName = artifact.FileName
		summaries = append(summaries, summary)
		details = append(details, detail...)
	}

	summaryOfSummaries := SummaryOfTestResultSummaries{
		Failed:    0,
		Passed:    0,
		Skipped:   0,
		Total:     0,
		Summaries: make([]TestResultSummary, 0, len(summaries)),
		BuildID:   buildID}
	for _, summary := range summaries {
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

		summaryOfSummaries.Failed += summary.Failed
		summaryOfSummaries.Passed += summary.Passed
		summaryOfSummaries.Skipped += summary.Skipped
		summaryOfSummaries.Summaries = append(
			summaryOfSummaries.Summaries,
			summary)
	}

	summaryOfSummaries.Total =
		summaryOfSummaries.Failed +
			summaryOfSummaries.Passed +
			summaryOfSummaries.Skipped

	err := m.Database.
		CreateInBatches(details, 100).
		Error
	if err != nil {
		ginutil.WriteDBWriteError(c, err, fmt.Sprintf(
			"Failed saving test result details for build with ID %d in database.",
			buildID))
	}

	c.JSON(http.StatusOK, summaryOfSummaries)
}

// getBuildAllTestResultDetailsHandler godoc
// @summary Get all test result details for specified build
// @tags artifact
// @param buildid path int true "Build ID"
// @success 200 {object} []TestResultDetail
// @failure 400 {object} problem.Response "Bad request"
// @failure 502 {object} problem.Response "Database is unreachable"
// @router /build/{buildid}/test-result-details [get]
func (m ArtifactModule) getBuildAllTestResultDetailsHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, buildIDParamName)
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
func (m ArtifactModule) getBuildTestResultDetailsHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, buildIDParamName)
	if !ok {
		return
	}

	artifactID, ok := ginutil.ParseParamUint(c, artifactIDParamName)
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
// @success 200 {object} SummaryOfTestResultSummaries
// @failure 400 {object} problem.Response "Bad Request"
// @failure 502 {object} problem.Response "Bad Gateway"
// @router /build/{buildid}/test-results-summary [get]
func (m ArtifactModule) getBuildTestResultsSummaryHandler(c *gin.Context) {
	buildID, ok := ginutil.ParseParamUint(c, buildIDParamName)
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

	summary := SummaryOfTestResultSummaries{
		BuildID:   buildID,
		Summaries: summaries}

	for _, v := range summaries {
		summary.Failed += v.Failed
		summary.Passed += v.Passed
		summary.Skipped += v.Skipped
	}

	summary.Total = summary.Failed + summary.Passed + summary.Skipped

	c.JSON(http.StatusOK, summary)
}

func parseMultipartFormData(c *gin.Context) ([]file, bool) {
	files := make([]file, 0)

	form, err := c.MultipartForm()
	if err != nil {
		ginutil.WriteMultipartFormReadError(c, err,
			"Failed reading multipart-form content from request body when uploading new artifact for build.")
		return nil, false
	}

	for k := range form.File {
		if fhs := form.File[k]; len(fhs) > 0 {
			data, ok := readMultipartFileData(c, fhs[0])
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

func readMultipartFileData(c *gin.Context, fh *multipart.FileHeader) ([]byte, bool) {
	if fh == nil {
		return nil, false
	}

	f, err := fh.Open()
	if err != nil {
		ginutil.WriteMultipartFormReadError(c, err,
			"Failed with starting to read file content from multipart form request body when uploading new artifact for build.")
		return nil, false
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		ginutil.WriteMultipartFormReadError(c, err,
			"Failed reading file content from multipart form request body when uploading new artifact for build.")
		return nil, false
	}
	return data, true
}

type testStatus string

const (
	testStatusSuccess testStatus = "Success"
	testStatusFailed  testStatus = "Failed"
	testStatusNoTests testStatus = "No tests"
)

type testsResults struct {
	Passed uint       `json:"passed"`
	Failed uint       `json:"failed"`
	Status testStatus `json:"status" enums:"Success,Failed,No tests"`
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

// getTestSummaryAndDetails currently only supports the TRX format.
func getTestSummaryAndDetails(data []byte, artifactID, buildID uint) ([]TestResultDetail, TestResultSummary, error) {
	var myTestRun testRun
	if err := xml.Unmarshal(data, &myTestRun); err != nil {
		return []TestResultDetail{}, TestResultSummary{}, err
	}

	details := make([]TestResultDetail, len(myTestRun.Results.UnitTestResults))
	for idx, utr := range myTestRun.Results.UnitTestResults {
		detail := &details[idx]
		detail.ArtifactID = artifactID
		detail.BuildID = buildID
		detail.Name = utr.TestName
		if utr.Outcome == "Passed" {
			detail.Status = testResultStatusSuccess
		} else if utr.Outcome == "Failed" {
			detail.Status = testResultStatusFailed
		} else if utr.Outcome == "NotExecuted" {
			detail.Status = testResultStatusSkipped
		}
		if detail.Status != testResultStatusSuccess {
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

	return details, summary, nil
}

func (m ArtifactModule) createArtifacts(c *gin.Context, files []file, buildID uint) ([]Artifact, bool) {
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

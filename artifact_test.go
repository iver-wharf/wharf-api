package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-core/pkg/problem"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// Implements io.Reader
type mockReader struct{}

func (m mockReader) Read(p []byte) (n int, err error) {
	return 0, nil
}

type mockSuccessDB struct{}

var artifactOne = Artifact{1, 1, nil, "file", "filename.ext", []byte{0, 2, 3, 4}}
var artifactTwo = Artifact{2, 1, nil, "file", "filename_2.ext", []byte{1, 3, 3, 7}}
var artifactThree = Artifact{3, 2, nil, "file", "filename_2.ext", []byte{1, 3, 3, 7}}

var trxSuccessfulTest = Artifact{4, 1, nil, "file", "test_passed.trx", []byte(`<TestRun><ResultSummary><Counters passed="5" failed="0"></Counters></ResultSummary></TestRun>`)}
var trxFailedTest = Artifact{5, 2, nil, "file", "test_failed.trx", []byte(`<TestRun><ResultSummary><Counters passed="0" failed="5"></Counters></ResultSummary></TestRun>`)}
var trxNoTestsTest = Artifact{6, 3, nil, "file", "test_no_tests.trx", []byte(`<TestRun><ResultSummary><Counters passed="0" failed="0"></Counters></ResultSummary></TestRun>`)}
var trxInvalidXML = Artifact{7, 4, nil, "file", "test_invalid_xml.trx", []byte(`<TestRun<ResultSummary><Counters passed=0" failed="0"></Counters>/ResultSummary>TestRun>`)}

func (m mockSuccessDB) GetArtifacts(buildID uint) ([]Artifact, error) {
	switch buildID {
	case 1:
		return []Artifact{
			artifactOne,
			artifactTwo,
			trxSuccessfulTest,
		}, nil
	case 2:
		return []Artifact{
			artifactThree,
		}, nil
	default:
		return []Artifact{}, nil
	}
}

func (m mockSuccessDB) GetArtifact(buildID, artifactID uint) (Artifact, error) {
	switch buildID {
	case 1:
		switch artifactID {
		case 1:
			return artifactOne, nil
		case 2:
			return artifactTwo, nil
		}
	case 2:
		switch artifactID {
		case 3:
			return artifactThree, nil
		}
	}
	return Artifact{}, gorm.ErrRecordNotFound
}

func (m mockSuccessDB) GetTRXFilesFromArtifacts(buildID uint) ([]Artifact, error) {
	switch buildID {
	case 1:
		return []Artifact{trxSuccessfulTest}, nil
	case 2:
		return []Artifact{trxFailedTest}, nil
	case 3:
		return []Artifact{trxNoTestsTest}, nil
	}
	return nil, nil
}

func (m mockSuccessDB) CreateArtifact(artifact *Artifact) error {
	return nil
}

type mockFailDB struct{}

func (m mockFailDB) GetArtifacts(buildID uint) ([]Artifact, error) {
	return nil, fmt.Errorf("failed to get artifacts from build with ID %d", buildID)
}

func (m mockFailDB) GetArtifact(buildID, artifactID uint) (Artifact, error) {
	if artifactID == 3 {
		return Artifact{}, gorm.ErrRecordNotFound
	}
	return Artifact{}, fmt.Errorf("failed to get artifact with ID %d from build with ID %d", artifactID, buildID)
}

func (m mockFailDB) GetTRXFilesFromArtifacts(buildID uint) ([]Artifact, error) {
	switch buildID {
	case 4:
		return []Artifact{trxInvalidXML}, nil
	}
	return nil, fmt.Errorf("failed to get trx files from build with ID %d", buildID)
}

func (m mockFailDB) CreateArtifact(artifact *Artifact) error {
	return fmt.Errorf("failed to create artifacts")
}

var successModule = artifactModule{mockSuccessDB{}}
var failModule = artifactModule{mockFailDB{}}

func TestGetBuildArtifactsHandler(t *testing.T) {
	testCases := []struct {
		m               artifactModule
		name            string
		wantContentType string
		wantStatusCode  int
		params          gin.Params
	}{
		{
			m:               successModule,
			name:            "Success when good request",
			wantContentType: gin.MIMEJSON,
			wantStatusCode:  http.StatusOK,
			params:          gin.Params{{Key: "buildid", Value: "1"}},
		},
		{
			m:               failModule,
			name:            "Fail when no buildid param",
			wantContentType: problem.HTTPContentType,
			wantStatusCode:  http.StatusBadRequest,
			params:          nil,
		},
		{
			m:               failModule,
			name:            "Fail when db read error",
			wantContentType: problem.HTTPContentType,
			wantStatusCode:  http.StatusBadGateway,
			params:          gin.Params{{Key: "buildid", Value: "1"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, r := makeContext(tc.params)

			tc.m.getBuildArtifactsHandler(c)

			res := r.Result()
			assert.Contains(t, res.Header.Get("Content-Type"), tc.wantContentType)
			assert.Equal(t, tc.wantStatusCode, res.StatusCode)
		})
	}
}

func TestGetBuildArtifactHandler(t *testing.T) {
	testCases := []struct {
		m               artifactModule
		name            string
		wantContentType string
		wantDisposition string
		wantStatusCode  int
		params          gin.Params
	}{
		{
			m:               successModule,
			name:            "Success when good request",
			wantContentType: "",
			wantDisposition: fmt.Sprintf("attachment; filename=\"%s\"", artifactTwo.FileName),
			wantStatusCode:  http.StatusOK,
			params: gin.Params{
				{Key: "buildid", Value: "1"},
				{Key: "artifactId", Value: "2"}},
		},
		{
			m:               failModule,
			name:            "Fail when no buildid param",
			wantContentType: problem.HTTPContentType,
			wantDisposition: "",
			wantStatusCode:  http.StatusBadRequest,
			params:          nil,
		},
		{
			m:               failModule,
			name:            "Fail when invalid buildid param",
			wantContentType: problem.HTTPContentType,
			wantDisposition: "",
			wantStatusCode:  http.StatusBadRequest,
			params: gin.Params{
				{Key: "buildid", Value: "-1"}},
		},
		{
			m:               failModule,
			name:            "Fail when no artifactId param",
			wantContentType: problem.HTTPContentType,
			wantDisposition: "",
			wantStatusCode:  http.StatusBadRequest,
			params: gin.Params{
				{Key: "buildid", Value: "1"}},
		},
		{
			m:               failModule,
			name:            "Fail when invalid artifactId param",
			wantContentType: problem.HTTPContentType,
			wantDisposition: "",
			wantStatusCode:  http.StatusBadRequest,
			params: gin.Params{
				{Key: "buildid", Value: "1"},
				{Key: "artifactId", Value: "-1"}},
		},
		{
			m:               failModule,
			name:            "Fail when db read error",
			wantContentType: problem.HTTPContentType,
			wantDisposition: "",
			wantStatusCode:  http.StatusBadGateway,
			params: gin.Params{
				{Key: "buildid", Value: "1"},
				{Key: "artifactId", Value: "2"}},
		},
		{
			m:               failModule,
			name:            "Fail when db not found error",
			wantContentType: problem.HTTPContentType,
			wantDisposition: "",
			wantStatusCode:  http.StatusNotFound,
			params: gin.Params{
				{Key: "buildid", Value: "1"},
				{Key: "artifactId", Value: "3"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, r := makeContext(tc.params)

			tc.m.getBuildArtifactHandler(c)

			res := r.Result()
			assert.Equal(t, tc.wantDisposition, res.Header.Get("Content-Disposition"))
			assert.Equal(t, tc.wantStatusCode, res.StatusCode)
		})
	}
}

func TestGetBuildTestsResultsHandler(t *testing.T) {
	testCases := []struct {
		m               artifactModule
		name            string
		wantContentType string
		wantStatusCode  int
		wantTestStatus  TestStatus
		params          gin.Params
	}{
		{
			m:               successModule,
			name:            "Status is TestStatusSuccess when success",
			wantContentType: gin.MIMEJSON,
			wantStatusCode:  http.StatusOK,
			wantTestStatus:  TestStatusSuccess,
			params: gin.Params{
				{Key: "buildid", Value: "1"}},
		},
		{
			m:               successModule,
			name:            "Status is TestStatusFailed when failed",
			wantContentType: gin.MIMEJSON,
			wantStatusCode:  http.StatusOK,
			wantTestStatus:  TestStatusFailed,
			params: gin.Params{
				{Key: "buildid", Value: "2"}},
		},
		{
			m:               successModule,
			name:            "Status is TestStatusNoTests when no tests",
			wantContentType: gin.MIMEJSON,
			wantStatusCode:  http.StatusOK,
			wantTestStatus:  TestStatusNoTests,
			params: gin.Params{
				{Key: "buildid", Value: "3"}},
		},
		{
			m:               failModule,
			name:            "Fail when no buildid param",
			wantContentType: problem.HTTPContentType,
			wantStatusCode:  http.StatusBadRequest,
			params:          nil,
		},
		{
			m:               failModule,
			name:            "Fail when db read error",
			wantContentType: problem.HTTPContentType,
			wantStatusCode:  http.StatusBadGateway,
			params: gin.Params{
				{Key: "buildid", Value: "1"}},
		},
		{
			m:               failModule,
			name:            "Fail when invalid xml",
			wantContentType: problem.HTTPContentType,
			wantStatusCode:  http.StatusBadRequest,
			params: gin.Params{
				{Key: "buildid", Value: "4"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, r := makeContext(tc.params)

			tc.m.getBuildTestsResultsHandler(c)

			res := r.Result()

			assert.Contains(t, res.Header.Get("Content-Type"), tc.wantContentType)
			assert.Equal(t, tc.wantStatusCode, res.StatusCode)

			data, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("failed reading response body: %q", string(data))
			}

			if tc.wantTestStatus != "" {
				var results TestsResults
				if err := json.Unmarshal(data, &results); err != nil {
					t.Fatalf("failed to unmarshal json: %q", string(data))
				}
				assert.Equal(t, tc.wantTestStatus, results.Status)
			}
		})
	}
}

// Helper functions

func makeContext(p gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.ReleaseMode)
	r := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(r)
	c.Request = httptest.NewRequest("GET", "/mock/", mockReader{})
	addParams(c, p)
	return c, r
}

func addParams(c *gin.Context, p gin.Params) {
	c.Params = append(c.Params, p...)
}

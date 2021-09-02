package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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

func (m mockSuccessDB) GetArtifacts(buildID uint) ([]Artifact, error) {
	return []Artifact{
		artifactOne,
		artifactTwo,
	}, nil
}

func (m mockSuccessDB) GetArtifact(buildID, artifactID uint) (Artifact, error) {
	return artifactTwo, nil
}

func (m mockSuccessDB) GetTRXFilesFromArtifacts(buildID uint) ([]Artifact, error) {
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
		params          gin.Params
		wantContentType string
		wantStatus      int
	}{
		{
			successModule,
			"Good request should succeed",
			gin.Params{{Key: "buildid", Value: "1"}},
			"application/json; charset=utf-8",
			http.StatusOK,
		},
		{
			failModule,
			"Fail when no buildid param",
			nil,
			"application/problem+json",
			http.StatusBadRequest,
		},
		{
			failModule,
			"Fail when db read error",
			gin.Params{{Key: "buildid", Value: "1"}},
			"application/problem+json",
			http.StatusBadGateway,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := makeContext(tc.params)

			tc.m.getBuildArtifactsHandler(c)

			assert.Equal(t, tc.wantContentType, c.Writer.Header().Get("Content-Type"))
			assert.Equal(t, tc.wantStatus, c.Writer.Status())
		})
	}
}

func TestGetBuildArtifactHandlerSuccess(t *testing.T) {
	testCases := []struct {
		m               artifactModule
		name            string
		params          []gin.Param
		wantContentType string
		wantDisposition string
		wantStatus      int
	}{
		{
			successModule,
			"Good request should succeed",
			gin.Params{
				{Key: "buildid", Value: "1"},
				{Key: "artifactId", Value: "2"},
			},
			"",
			fmt.Sprintf("attachment; filename=\"%s\"", artifactTwo.FileName),
			http.StatusOK,
		},
		{
			failModule,
			"Fail when no buildid param",
			nil,
			"application/problem+json",
			"",
			http.StatusBadRequest,
		},
		{
			failModule,
			"Fail when invalid buildid param",
			gin.Params{{Key: "buildid", Value: "-1"}},
			"application/problem+json",
			"",
			http.StatusBadRequest,
		},
		{
			failModule,
			"Fail when no artifactId param",
			gin.Params{{Key: "buildid", Value: "1"}},
			"application/problem+json",
			"",
			http.StatusBadRequest,
		},
		{
			failModule,
			"Fail when invalid artifactId param",
			gin.Params{
				{Key: "buildid", Value: "1"},
				{Key: "artifactId", Value: "-1"}},
			"application/problem+json",
			"",
			http.StatusBadRequest,
		},
		{
			failModule,
			"Fail when db read error",
			gin.Params{
				{Key: "buildid", Value: "1"},
				{Key: "artifactId", Value: "2"}},
			"application/problem+json",
			"",
			http.StatusBadGateway,
		},
		{
			failModule,
			"Fail when db not found error",
			gin.Params{
				{Key: "buildid", Value: "1"},
				{Key: "artifactId", Value: "3"}},
			"application/problem+json",
			"",
			http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := makeContext(tc.params)

			tc.m.getBuildArtifactHandler(c)

			assert.Equal(t, tc.wantDisposition, c.Writer.Header().Get("Content-Disposition"))
			assert.Equal(t, tc.wantStatus, c.Writer.Status())
		})
	}
}

// Helper functions

func makeContext(p gin.Params) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/mock/", mockReader{})
	addParams(c, p)
	return c
}

func addParams(c *gin.Context, p gin.Params) {
	c.Params = append(c.Params, p...)
}

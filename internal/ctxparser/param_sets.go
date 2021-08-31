package ctxparser

const (
	ArtifactIDParamName = "artifactId"
	BuildIDParamName    = "buildid"
)

type ParamSetArtifactID struct {
	ArtifactID uint `uri:"artifactId" binding:"required"`
}

type ParamSetBuildID struct {
	BuildID uint `uri:"buildid" binding:"required"`
}

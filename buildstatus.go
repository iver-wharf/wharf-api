package main

import "strconv"

// BuildStatus is an enum of different states for a build.
type BuildStatus int

const (
	// BuildScheduling means the build has been registered, but no code
	// execution has begun yet. This is usually quite an ephemeral state.
	BuildScheduling BuildStatus = iota
	// BuildRunning means the build is executing right now. The execution
	// engine has load in the target code paths and repositories.
	BuildRunning
	// BuildCompleted means the build has finished execution successfully.
	BuildCompleted
	// BuildFailed means that something went wrong with the build. Could be a
	// misconfiguration in the .wharf-ci.yml file, or perhaps a scripting error
	// in some build step.
	BuildFailed
)

func (bs BuildStatus) String() string {
	switch bs {
	case BuildScheduling:
		return "Scheduling"
	case BuildRunning:
		return "Running"
	case BuildCompleted:
		return "Completed"
	case BuildFailed:
		return "Failed"
	default:
		return strconv.Itoa(int(bs))
	}
}

var toID = map[string]BuildStatus{
	"Scheduling": BuildScheduling,
	"Running":    BuildRunning,
	"Completed":  BuildCompleted,
	"Failed":     BuildFailed,
}

func parseBuildStatus(name string) (status BuildStatus, ok bool) {
	status, ok = toID[name]
	return
}

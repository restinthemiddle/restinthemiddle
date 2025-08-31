package version

import "fmt"

// Version information - should be set via build flags (see Dockerfile and Makefile).
var (
	Version   = "dev"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

// Info returns formatted version information.
func Info() string {
	return fmt.Sprintf("restinthemiddle %s (built %s, commit %s)", Version, BuildDate, GitCommit)
}

// Short returns just the version string.
func Short() string {
	return Version
}

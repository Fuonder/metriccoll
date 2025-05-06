package buildinfo

import (
	"fmt"
	"runtime"
	"time"
)

type BuildInfo struct {
	// BuildVersion is the version of build from Git tags.
	BuildVersion string `json:"version"`
	// BuildCommit is the short representation of git commit hash.
	BuildCommit string `json:"commit_id"`
	// BuildDate is the build date.
	BuildDate time.Time `json:"time"`
	// Compiler is the version of Go compiler used for building.
	Compiler string `json:"compiler"`
}

func NewBuildInfo(Version string, Commit string, Date string, GeneratedBuild *BuildInfo) *BuildInfo {
	if GeneratedBuild != nil {
		if Version != "N/A" {
			GeneratedBuild.BuildVersion = Version
		}
		if Commit != "N/A" {
			GeneratedBuild.BuildCommit = Commit
		}
		if Date != "N/A" {
			GeneratedBuild.BuildDate = MustParseTime(Date)
		}
		return GeneratedBuild
	}
	bInfo := &BuildInfo{
		BuildVersion: Version,
		BuildCommit:  Commit,
		BuildDate:    MustParseTime(Date),
		Compiler:     runtime.Version(),
	}
	return bInfo
}

func (b *BuildInfo) String() string {
	return fmt.Sprintf(
		"Build Version : %s\nBuild Commit  : %s\nBuild Date    : %s\nCompiler      : %s\n",
		b.BuildVersion, b.BuildCommit, b.BuildDate.Format(time.RFC3339), b.Compiler,
	)
}

func MustParseTime(val string) time.Time {
	t, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return time.Now().UTC()
	}
	return t
}

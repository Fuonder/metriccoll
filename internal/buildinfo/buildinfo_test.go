package buildinfo

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"runtime"
	"testing"
	"time"
)

func TestMustParseTime(t *testing.T) {
	refTime := time.Date(2023, 5, 1, 15, 4, 5, 0, time.UTC)

	tests := []struct {
		name      string
		input     string
		want      time.Time
		approxNow bool
	}{
		{
			name:      "valid RFC3339 datetime",
			input:     "2023-05-01T15:04:05Z",
			want:      refTime,
			approxNow: false,
		},
		{
			name:      "invalid date",
			input:     "invalid-date",
			approxNow: true,
		},
		{
			name:      "empty string",
			input:     "",
			approxNow: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MustParseTime(tt.input)
			if tt.approxNow {
				assert.WithinDuration(t,
					time.Now().UTC(),
					got,
					2*time.Second,
					"Ожидается текущее время в UTC")
			} else {
				assert.True(t,
					tt.want.Equal(got),
					"Ожидалось: %v, Получено: %v",
					tt.want,
					got)
			}
		})
	}
}

func TestNewBuildInfo(t *testing.T) {
	refDateStr := "2024-01-01T12:00:00Z"
	refDate := MustParseTime(refDateStr)

	tests := []struct {
		name           string
		version        string
		commit         string
		date           string
		generated      *BuildInfo
		expectedResult BuildInfo
	}{
		{
			name:      "empty_GeneratedBuild_use_ldflags",
			version:   "v1.2.3",
			commit:    "abc1234",
			date:      refDateStr,
			generated: nil,
			expectedResult: BuildInfo{
				BuildVersion: "v1.2.3",
				BuildCommit:  "abc1234",
				BuildDate:    refDate,
				Compiler:     runtime.Version(),
			},
		},
		{
			name:      "empty_GeneratedBuild_empty_ldflags",
			version:   "N/A",
			commit:    "N/A",
			date:      "N/A",
			generated: nil,
			expectedResult: BuildInfo{
				BuildVersion: "N/A",
				BuildCommit:  "N/A",
				Compiler:     runtime.Version(),
			},
		},
		{
			name:    "use_generated_build_empty_ldflags",
			version: "N/A",
			commit:  "N/A",
			date:    "N/A",
			generated: &BuildInfo{
				BuildVersion: "gen-v1",
				BuildCommit:  "gen123",
				BuildDate:    refDate,
				Compiler:     "go1.generated",
			},
			expectedResult: BuildInfo{
				BuildVersion: "gen-v1",
				BuildCommit:  "gen123",
				BuildDate:    refDate,
				Compiler:     "go1.generated",
			},
		},
		{
			name:    "ldflags_override_generated_build",
			version: "v2.0.0",
			commit:  "override123",
			date:    refDateStr,
			generated: &BuildInfo{
				BuildVersion: "old-v",
				BuildCommit:  "old-c",
				BuildDate:    time.Now(),
				Compiler:     "should-not-change",
			},
			expectedResult: BuildInfo{
				BuildVersion: "v2.0.0",
				BuildCommit:  "override123",
				BuildDate:    refDate,
				Compiler:     "should-not-change",
			},
		},
		{
			name:      "invalid_date",
			version:   "v3",
			commit:    "abc",
			date:      "invalid-date",
			generated: nil,
			expectedResult: BuildInfo{
				BuildVersion: "v3",
				BuildCommit:  "abc",
				Compiler:     runtime.Version(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewBuildInfo(tt.version, tt.commit, tt.date, tt.generated)

			require.NotNil(t, result)
			assert.Equal(t, tt.expectedResult.BuildVersion, result.BuildVersion)
			assert.Equal(t, tt.expectedResult.BuildCommit, result.BuildCommit)
			assert.Equal(t, tt.expectedResult.Compiler, result.Compiler)

			if tt.name == "invalid_date" || tt.name == "empty_GeneratedBuild_empty_ldflags" {
				diff := time.Since(result.BuildDate)
				assert.True(t, diff < 2*time.Second, "BuildDate should be current UTC")
			} else {
				assert.True(t, tt.expectedResult.BuildDate.Equal(result.BuildDate), "BuildDate mismatch")
			}
		})
	}
}

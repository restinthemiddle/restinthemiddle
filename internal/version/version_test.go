package version

import (
	"strings"
	"testing"
)

func TestShort(t *testing.T) {
	tests := []struct {
		name            string
		version         string
		expectedVersion string
	}{
		{
			name:            "returns dev version",
			version:         "dev",
			expectedVersion: "dev",
		},
		{
			name:            "returns semantic version",
			version:         "v1.2.3",
			expectedVersion: "v1.2.3",
		},
		{
			name:            "returns git describe version",
			version:         "v1.2.3-5-g1abc234",
			expectedVersion: "v1.2.3-5-g1abc234",
		},
		{
			name:            "returns dirty version",
			version:         "v1.2.3-dirty",
			expectedVersion: "v1.2.3-dirty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			originalVersion := Version

			// Set test values
			Version = tt.version

			// Test
			result := Short()

			// Restore original values
			Version = originalVersion

			// Assert
			if result != tt.expectedVersion {
				t.Errorf("Short() = %q, want %q", result, tt.expectedVersion)
			}
		})
	}
}

func TestInfo(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		buildDate   string
		gitCommit   string
		expectedFmt string
	}{
		{
			name:        "returns formatted info with default values",
			version:     "dev",
			buildDate:   "unknown",
			gitCommit:   "unknown",
			expectedFmt: "restinthemiddle dev (built unknown, commit unknown)",
		},
		{
			name:        "returns formatted info with real values",
			version:     "v1.2.3",
			buildDate:   "2025-08-31T10:30:00Z",
			gitCommit:   "abc123def456",
			expectedFmt: "restinthemiddle v1.2.3 (built 2025-08-31T10:30:00Z, commit abc123def456)",
		},
		{
			name:        "returns formatted info with git describe version",
			version:     "v1.2.3-5-g1abc234",
			buildDate:   "2025-08-31T10:30:00Z",
			gitCommit:   "1abc234567890",
			expectedFmt: "restinthemiddle v1.2.3-5-g1abc234 (built 2025-08-31T10:30:00Z, commit 1abc234567890)",
		},
		{
			name:        "returns formatted info with dirty version",
			version:     "v1.2.3-dirty",
			buildDate:   "2025-08-31T10:30:00Z",
			gitCommit:   "abc123def456-dirty",
			expectedFmt: "restinthemiddle v1.2.3-dirty (built 2025-08-31T10:30:00Z, commit abc123def456-dirty)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			originalVersion := Version
			originalBuildDate := BuildDate
			originalGitCommit := GitCommit

			// Set test values
			Version = tt.version
			BuildDate = tt.buildDate
			GitCommit = tt.gitCommit

			// Test
			result := Info()

			// Restore original values
			Version = originalVersion
			BuildDate = originalBuildDate
			GitCommit = originalGitCommit

			// Assert
			if result != tt.expectedFmt {
				t.Errorf("Info() = %q, want %q", result, tt.expectedFmt)
			}
		})
	}
}

func TestInfoFormat(t *testing.T) {
	// Save original values
	originalVersion := Version
	originalBuildDate := BuildDate
	originalGitCommit := GitCommit

	// Set test values
	Version = "v1.2.3"
	BuildDate = "2025-08-31T10:30:00Z"
	GitCommit = "abc123"

	// Test
	result := Info()

	// Restore original values
	Version = originalVersion
	BuildDate = originalBuildDate
	GitCommit = originalGitCommit

	// Assert format structure
	if !strings.HasPrefix(result, "restinthemiddle ") {
		t.Errorf("Info() should start with 'restinthemiddle ', got: %q", result)
	}

	if !strings.Contains(result, "built") {
		t.Errorf("Info() should contain 'built', got: %q", result)
	}

	if !strings.Contains(result, "commit") {
		t.Errorf("Info() should contain 'commit', got: %q", result)
	}

	expectedParts := []string{"v1.2.3", "2025-08-31T10:30:00Z", "abc123"}
	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Info() should contain %q, got: %q", part, result)
		}
	}
}

// TestPackageVariables tests that the package variables are properly initialized.
func TestPackageVariables(t *testing.T) {
	// Test default values exist (even if they might be overridden at build time)
	if Version == "" {
		t.Error("Version should have a default value")
	}

	if BuildDate == "" {
		t.Error("BuildDate should have a default value")
	}

	if GitCommit == "" {
		t.Error("GitCommit should have a default value")
	}
}

// BenchmarkInfo benchmarks the Info function.
func BenchmarkInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Info()
	}
}

// BenchmarkShort benchmarks the Short function.
func BenchmarkShort(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Short()
	}
}

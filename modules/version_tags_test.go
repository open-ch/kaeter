package modules

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionsWithTagsIntegration(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	versionsPath := filepath.Join(tmpDir, "versions.yaml")

	// Create initial versions.yaml with tags
	initialYAML := `id: test:module
type: Makefile
versioning: SemVer
versions:
  0.0.0: 2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091
  1.0.0: 2020-01-01T00:00:00Z|abc123def456|production,stable
  1.1.0: 2020-06-01T00:00:00Z|def789ghi012|beta,experimental
`

	err := os.WriteFile(versionsPath, []byte(initialYAML), 0600)
	require.NoError(t, err)

	// Read the versions file
	versions, err := ReadFromFile(versionsPath)
	require.NoError(t, err)

	// Verify the versions were parsed correctly
	assert.Equal(t, "test:module", versions.ID)
	assert.Equal(t, 3, len(versions.ReleasedVersions))

	// Check version without tags (backward compatible)
	v1 := versions.ReleasedVersions[0]
	assert.Equal(t, "0.0.0", v1.Number.String())
	assert.Nil(t, v1.Tags)

	// Check version with production and stable tags
	v2 := versions.ReleasedVersions[1]
	assert.Equal(t, "1.0.0", v2.Number.String())
	assert.Equal(t, []string{"production", "stable"}, v2.Tags)

	// Check version with beta and experimental tags
	v3 := versions.ReleasedVersions[2]
	assert.Equal(t, "1.1.0", v3.Number.String())
	assert.Equal(t, []string{"beta", "experimental"}, v3.Tags)

	// Add a new version without tags
	refTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	newVersion, err := versions.AddRelease(&refTime, BumpMinor, "", "newcommit123")
	require.NoError(t, err)
	assert.Equal(t, "1.2.0", newVersion.Number.String())
	assert.Nil(t, newVersion.Tags)

	// Save and reload to test round-trip
	err = versions.SaveToFile(versionsPath)
	require.NoError(t, err)

	reloaded, err := ReadFromFile(versionsPath)
	require.NoError(t, err)

	// Verify all versions survived round-trip
	assert.Equal(t, 4, len(reloaded.ReleasedVersions))

	// Verify tags were preserved
	assert.Nil(t, reloaded.ReleasedVersions[0].Tags)
	assert.Equal(t, []string{"production", "stable"}, reloaded.ReleasedVersions[1].Tags)
	assert.Equal(t, []string{"beta", "experimental"}, reloaded.ReleasedVersions[2].Tags)
	assert.Nil(t, reloaded.ReleasedVersions[3].Tags)

	// Read the file content to verify format
	content, err := os.ReadFile(versionsPath)
	require.NoError(t, err)

	contentStr := string(content)
	// Verify tags are in the file
	assert.Contains(t, contentStr, "|production,stable")
	assert.Contains(t, contentStr, "|beta,experimental")
	// Verify versions without tags don't have trailing pipe
	assert.Contains(t, contentStr, "0.0.0: 2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091\n")
	assert.Contains(t, contentStr, "1.2.0: 2021-01-01T00:00:00Z|newcommit123\n")
}

func TestVersionsWithTagsBackwardCompatibility(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	versionsPath := filepath.Join(tmpDir, "versions.yaml")

	// Create a versions.yaml WITHOUT tags (old format)
	oldFormatYAML := `id: test:module
type: Makefile
versioning: SemVer
versions:
  0.0.0: 2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091
  1.0.0: 2020-01-01T00:00:00Z|abc123def456
`

	err := os.WriteFile(versionsPath, []byte(oldFormatYAML), 0600)
	require.NoError(t, err)

	// Read the old format file
	versions, err := ReadFromFile(versionsPath)
	require.NoError(t, err)

	// Verify versions were parsed correctly without tags
	assert.Equal(t, 2, len(versions.ReleasedVersions))
	assert.Nil(t, versions.ReleasedVersions[0].Tags)
	assert.Nil(t, versions.ReleasedVersions[1].Tags)

	// Add a new version and save
	refTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = versions.AddRelease(&refTime, BumpMinor, "", "newcommit789")
	require.NoError(t, err)

	err = versions.SaveToFile(versionsPath)
	require.NoError(t, err)

	// Read again and verify old versions still don't have tags
	reloaded, err := ReadFromFile(versionsPath)
	require.NoError(t, err)

	assert.Equal(t, 3, len(reloaded.ReleasedVersions))
	assert.Nil(t, reloaded.ReleasedVersions[0].Tags)
	assert.Nil(t, reloaded.ReleasedVersions[1].Tags)
	assert.Nil(t, reloaded.ReleasedVersions[2].Tags)

	// Read file content and verify no extra pipes were added
	content, err := os.ReadFile(versionsPath)
	require.NoError(t, err)

	contentStr := string(content)
	// Should not have trailing pipes for versions without tags
	assert.NotContains(t, contentStr, "675156f77a931aa40ceb115b763d9d1230b26091|")
	assert.NotContains(t, contentStr, "abc123def456|")
	assert.NotContains(t, contentStr, "newcommit789|")
}

func TestManuallyAddTagsToVersion(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	versionsPath := filepath.Join(tmpDir, "versions.yaml")

	// Create initial versions.yaml
	initialYAML := `id: test:module
type: Makefile
versioning: SemVer
versions:
  1.0.0: 2020-01-01T00:00:00Z|abc123
`

	err := os.WriteFile(versionsPath, []byte(initialYAML), 0600)
	require.NoError(t, err)

	// Read, modify tags, and save
	versions, err := ReadFromFile(versionsPath)
	require.NoError(t, err)

	// Add tags to existing version
	versions.ReleasedVersions[0].Tags = []string{"production", "hotfix"}

	err = versions.SaveToFile(versionsPath)
	require.NoError(t, err)

	// Reload and verify tags were added
	reloaded, err := ReadFromFile(versionsPath)
	require.NoError(t, err)

	assert.Equal(t, []string{"production", "hotfix"}, reloaded.ReleasedVersions[0].Tags)

	// Verify file content
	content, err := os.ReadFile(versionsPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "|production,hotfix")
}

package modules

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/mocks"
)

const sampleSemVerVersion = `id: testGroup:testModule
type: Makefile
versioning: SemVer
versions:
  0.0.0: 2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091
  1.1.1: 2019-04-01T16:06:07Z|934b40f6862a2dc28f4045bd57d1832dfde10e55
  1.2.0: 2019-04-02T16:06:07Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e55
  1.2.0-1: 2019-04-02T16:06:07Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e66
  v2.0.0: 2020-01-01T00:00:00Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e77
  v2.0.0-1+2: 2020-01-01T01:00:00Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e88
`

const sampleAnyStringVersion = `# Auto-generated file: please edit with care.
# Identifies this module within the fat repo.
id: testGroup:testModule
# The underlying tool to which building and releasing is handed off
type: Makefile
# Should this module be versioned with semantic or calendar versioning?
versioning: AnyStringVer
# Version identifiers have the following format:
# <version string>: <RFC3339 formatted timestamp>|<commit ID>
versions:
  0.0.0: 2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091
  AnyString: 2019-04-01T16:06:07Z|934b40f6862a2dc28f4045bd57d1832dfde10e55
  a-zA-Z0-9.+_~@: 2019-04-02T16:06:07Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e55
  whyNot: 2020-01-01T00:00:00Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e66
  1.0.0-1: 2019-04-01T16:06:07Z|100156f77a931aa40ceb115b763d9d1230b26091
  1.0.0-2: 2019-04-01T16:06:07Z|100b40f6862a2dc28f4045bd57d1832dfde10e55
  1.1.0-1: 2020-01-01T00:00:00Z|110b40f6862a2dc28f4045bd57d1832dfde10e66
`

const templateMetadataVersion = `# Auto-generated file: please edit with care.
# Identifies this module within the fat repo.
id: testGroup:testModule
# The underlying tool to which building and releasing is handed off
type: Makefile
# Should this module be versioned with semantic or calendar versioning?
versioning: SemVer
%s
# Version identifiers have the following format:
# <version string>: <RFC3339 formatted timestamp>|<commit ID>
versions:
  0.0.0: 2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091
`

func parseYamlOnly(t *testing.T, rawYAML string) *Versions {
	var deser Versions
	err := yaml.Unmarshal([]byte(rawYAML), &deser)
	assert.NoError(t, err)
	return &deser
}

func parseVersions(t *testing.T, rawYAML string) *Versions {
	parsed, err := unmarshalVersions([]byte(rawYAML))
	assert.NoError(t, err)
	return parsed
}

func TestGetVersionsFilePath(t *testing.T) {
	tests := []struct {
		name               string
		subPath            string
		mockFiles          map[string]string
		expectedPathEnding string
		expectError        bool
	}{
		{
			name:        "Path without a versions yaml uses default",
			expectError: true,
		},
		{
			name:        "Fails for path that doesn't exist",
			subPath:     "not-a-folder/",
			expectError: true,
		},
		{
			name:        "Fail if input is not a directory",
			subPath:     "strange.yaml",
			mockFiles:   map[string]string{"strange.yaml": "# this not a directory"},
			expectError: true,
		},
		{
			name:               "Finds version.yaml file",
			mockFiles:          map[string]string{"versions.yaml": "# Empty"},
			expectedPathEnding: "versions.yaml",
		},
		{
			name:               "Finds version.yml file",
			mockFiles:          map[string]string{"versions.yml": "# Empty"},
			expectedPathEnding: "versions.yml",
		},
		{
			name:        "Fails if both yaml and yml present",
			mockFiles:   map[string]string{"versions.yml": "# Empty", "versions.yaml": "# Empty"},
			expectError: true,
		},
		{
			name:        "Fails if version.yaml not in given path but in sub folder",
			mockFiles:   map[string]string{"module/versions.yaml": "# Empty"},
			expectError: true,
		},
		{
			name: "Pick file at given path if multiple ones are found",
			mockFiles: map[string]string{
				"versions.yaml":         "# Empty",
				"module/versions.yaml":  "# Empty",
				"module2/versions.yaml": "# Empty",
			},
			expectedPathEnding: "versions.yaml",
		},
		{
			name: "Fails if multiple versions files found in different folders",
			mockFiles: map[string]string{
				"module/versions.yaml":  "# Empty",
				"module2/versions.yaml": "# Empty",
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder := t.TempDir()
			_ = mocks.CreateMockFolder(t, testFolder, "module")
			_ = mocks.CreateMockFolder(t, testFolder, "module2")
			for filename, content := range tc.mockFiles {
				mocks.CreateMockFile(t, testFolder, filename, content)
			}

			versionsAbsPath, err := GetVersionsFilePath(path.Join(testFolder, tc.subPath))

			if tc.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, path.Join(testFolder, tc.expectedPathEnding), versionsAbsPath)
		})
	}
}

// TestYamlBehavior just checks that some behavior of the lib is as expected
func TestYamlBehavior(t *testing.T) {
	// With goccy/go-yaml, we can directly unmarshal to our struct
	var lazily Versions
	err := yaml.Unmarshal([]byte(sampleSemVerVersion), &lazily)
	assert.NoError(t, err)
	assert.Equal(t, parseYamlOnly(t, sampleSemVerVersion), &lazily)
}

func TestYamlV3UnmarshalFromStruct(t *testing.T) {
	tests := []struct {
		name                string
		yaml                string
		expectedRawVersions yaml.MapSlice
	}{
		{
			name: "SemVer versions",
			yaml: sampleSemVerVersion,
			expectedRawVersions: yaml.MapSlice{
				{Key: "0.0.0", Value: "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091"},
				{Key: "1.1.1", Value: "2019-04-01T16:06:07Z|934b40f6862a2dc28f4045bd57d1832dfde10e55"},
				{Key: "1.2.0", Value: "2019-04-02T16:06:07Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e55"},
				{Key: "1.2.0-1", Value: "2019-04-02T16:06:07Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e66"},
				{Key: "v2.0.0", Value: "2020-01-01T00:00:00Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e77"},
				{Key: "v2.0.0-1+2", Value: "2020-01-01T01:00:00Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e88"},
			},
		},
		{
			name: "AnyString versions",
			yaml: sampleAnyStringVersion,
			expectedRawVersions: yaml.MapSlice{
				{Key: "0.0.0", Value: "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091"},
				{Key: "AnyString", Value: "2019-04-01T16:06:07Z|934b40f6862a2dc28f4045bd57d1832dfde10e55"},
				{Key: "a-zA-Z0-9.+_~@", Value: "2019-04-02T16:06:07Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e55"},
				{Key: "whyNot", Value: "2020-01-01T00:00:00Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e66"},
				{Key: "1.0.0-1", Value: "2019-04-01T16:06:07Z|100156f77a931aa40ceb115b763d9d1230b26091"},
				{Key: "1.0.0-2", Value: "2019-04-01T16:06:07Z|100b40f6862a2dc28f4045bd57d1832dfde10e55"},
				{Key: "1.1.0-1", Value: "2020-01-01T00:00:00Z|110b40f6862a2dc28f4045bd57d1832dfde10e66"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := parseYamlOnly(t, tc.yaml)
			assert.NotNil(t, raw.ReleasedVersions)

			// Convert the VersionSlice to key-value pairs for comparison
			rawVersions := make(yaml.MapSlice, 0, len(raw.ReleasedVersions))
			for _, versionData := range raw.ReleasedVersions {
				// Since we're testing raw parsing, we need to use the raw fields
				// Check if this is a raw version (not yet parsed)
				if rawVersion, ok := versionData.Number.(VersionString); ok {
					rawVersions = append(rawVersions, yaml.MapItem{
						Key:   rawVersion.Version,
						Value: versionData.CommitID, // Raw value stored in CommitID during unmarshaling
					})
				} else {
					// If already parsed, use the parsed values
					rawVersions = append(rawVersions, yaml.MapItem{
						Key:   versionData.Number.String(),
						Value: versionData.marshalReleaseData(),
					})
				}
			}

			assert.Equal(t, len(tc.expectedRawVersions), len(rawVersions))
			for i, rawVersion := range rawVersions {
				assert.Equal(t, tc.expectedRawVersions[i], rawVersion)
			}
		})
	}
}

func TestUnmarshalVersionsSemVer(t *testing.T) {
	raw := parseYamlOnly(t, sampleSemVerVersion)
	high, err := unmarshalVersions([]byte(sampleSemVerVersion))

	assert.NoError(t, err)
	assert.Equal(t, raw.ID, high.ID)
	assert.Equal(t, raw.VersioningType, high.VersioningType)
	assert.Equal(t, raw.ModuleType, high.ModuleType)
	assert.Equal(t, (*Metadata)(nil), high.Metadata)
	assert.Equal(t, 6, len(high.ReleasedVersions))

	// Check the ordering of the underlying metadata slice
	assert.Equal(t, "0.0.0", high.ReleasedVersions[0].Number.String())
	assert.Equal(t, "1.1.1", high.ReleasedVersions[1].Number.String())
	assert.Equal(t, "1.2.0", high.ReleasedVersions[2].Number.String())
	assert.Equal(t, "1.2.0-1", high.ReleasedVersions[3].Number.String())
	assert.Equal(t, "v2.0.0", high.ReleasedVersions[4].Number.String())
	assert.Equal(t, "v2.0.0-1+2", high.ReleasedVersions[5].Number.String())
}

func TestUnmarshalVersionsAnyStringVer(t *testing.T) {
	raw := parseYamlOnly(t, sampleAnyStringVersion)
	high, err := unmarshalVersions([]byte(sampleAnyStringVersion))

	assert.NoError(t, err)
	assert.Equal(t, raw.ID, high.ID)
	assert.Equal(t, raw.VersioningType, high.VersioningType)
	assert.Equal(t, raw.ModuleType, high.ModuleType)
	assert.Equal(t, (*Metadata)(nil), high.Metadata)
	assert.Equal(t, 7, len(high.ReleasedVersions))

	// Check the ordering of the underlying metadata slice
	assert.Equal(t, "0.0.0", high.ReleasedVersions[0].Number.String())
	assert.Equal(t, "AnyString", high.ReleasedVersions[1].Number.String())
	assert.Equal(t, "a-zA-Z0-9.+_~@", high.ReleasedVersions[2].Number.String())
	assert.Equal(t, "whyNot", high.ReleasedVersions[3].Number.String())
	assert.Equal(t, "1.0.0-1", high.ReleasedVersions[4].Number.String())
	assert.Equal(t, "1.0.0-2", high.ReleasedVersions[5].Number.String())
	assert.Equal(t, "1.1.0-1", high.ReleasedVersions[6].Number.String())
}

func TestUnmarshalMetadata(t *testing.T) {
	var tests = []struct {
		name             string
		rawMetadata      string
		expectedMetadata *Metadata
		expectedError    bool
	}{
		{
			name:             "Expected no metadata",
			rawMetadata:      "",
			expectedMetadata: (*Metadata)(nil),
		},
		{
			name:             "Expected empty metadata",
			rawMetadata:      `metadata: {}`,
			expectedMetadata: &Metadata{},
		},
		{
			name: "Expected empty annotations",
			rawMetadata: `metadata:
    annotations: {}`,
			expectedMetadata: &Metadata{Annotations: map[string]string{}},
		},
		{
			name: "Expected metadata with single annotation",
			rawMetadata: `metadata:
    annotations:
        open.ch/osix-package: "true"`,
			expectedMetadata: &Metadata{
				Annotations: map[string]string{"open.ch/osix-package": "true"},
			},
		},
		{
			name: "Expected metadata with multiple annotations",
			rawMetadata: `metadata:
    annotations:
        programmers: Lovelace,Turing,Ritchie,Stroustrup
        open.ch/osix-package: "true"`,
			expectedMetadata: &Metadata{
				Annotations: map[string]string{"programmers": "Lovelace,Turing,Ritchie,Stroustrup", "open.ch/osix-package": "true"},
			},
		},
	}

	for _, tc := range tests {
		versionsContent := fmt.Sprintf(templateMetadataVersion, tc.rawMetadata)

		ver, err := unmarshalVersions([]byte(versionsContent))

		assert.NoError(t, err)
		assert.Equal(t, tc.expectedMetadata, ver.Metadata, tc.name)
	}
}

func TestVersionsSemVer_Marshal(t *testing.T) {
	vers := parseVersions(t, sampleSemVerVersion)

	bytes, err := vers.Marshal()
	assert.NoError(t, err)

	t.Log(string(bytes))
	assert.Equal(t, sampleSemVerVersion, string(bytes))
}

func TestVersionsAnyStringVer_Marshal(t *testing.T) {
	vers := parseVersions(t, sampleAnyStringVersion)

	bytes, err := vers.Marshal()
	assert.NoError(t, err)

	t.Log(string(bytes))
	assert.Equal(t, sampleAnyStringVersion, string(bytes))
}

func TestAddRelease_SemVer(t *testing.T) {
	var tests = []struct {
		name                  string
		bumpType              SemVerBump
		gitRef                string
		versionInput          string
		hasError              bool
		expectedVersionNumber *VersionNumber
	}{
		{
			name:                  "UserSpecifiedVersion",
			bumpType:              BumpPatch,
			gitRef:                "someCommitId",
			versionInput:          "v5.6.7",
			expectedVersionNumber: &VersionNumber{*semver.MustParse("v5.6.7")},
		},
		{
			name:                  "UserSpecifiedVersion without v prefix",
			bumpType:              BumpPatch,
			gitRef:                "someCommitId",
			versionInput:          "5.6.7",
			expectedVersionNumber: NewVersion(5, 6, 7),
		},
		{
			name:                  "Minor semver bump to v2.1.0",
			bumpType:              BumpMinor,
			gitRef:                "someCommitId",
			versionInput:          "",
			expectedVersionNumber: &VersionNumber{*semver.MustParse("v2.1.0")},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			versions := parseVersions(t, sampleSemVerVersion)
			refTime, err := time.Parse(time.RFC3339, "2020-02-02T00:00:00Z")
			assert.NoError(t, err)

			_, err = versions.AddRelease(&refTime, tc.bumpType, tc.versionInput, tc.gitRef)

			if tc.hasError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, 7, len(versions.ReleasedVersions), "expecting an additional entry in the versions")

			last := versions.ReleasedVersions[len(versions.ReleasedVersions)-1]
			assert.Equal(t, VersionMetadata{
				Number:    tc.expectedVersionNumber,
				Timestamp: refTime,
				CommitID:  tc.gitRef,
			}, *last, "the new version should be appended at the end")

			// Now check that when marshaling we actually write the new value out to the YAML
			marshaled, err := versions.Marshal()
			expected := fmt.Sprintf("%s  %s: 2020-02-02T00:00:00Z|%s\n", sampleSemVerVersion, tc.expectedVersionNumber, tc.gitRef)
			assert.NoError(t, err)
			assert.Equal(t, expected, string(marshaled))
		})
	}
}

func TestAddRelease_AnyStringVer(t *testing.T) {
	var tests = []struct {
		name           string
		bumpType       SemVerBump
		gitRef         string
		versionInput   string
		hasError       bool
		customVersions *Versions
	}{
		{
			name:     "empty version & commit ref fails",
			hasError: true,
		},
		{
			name:         "empty commit ref fails",
			versionInput: "someVersion",
			hasError:     true,
		},
		{
			name:     "empty version fails for non semver module",
			gitRef:   "commitId",
			hasError: true,
		},
		{
			name:         "forbidden character should trigger error",
			gitRef:       "commitId",
			versionInput: "forbiddenChar!",
			hasError:     true,
		},

		{
			name:         "adding to faulty versions obj fails",
			gitRef:       "commitId",
			versionInput: "newVersion",
			hasError:     true,
			customVersions: &Versions{
				ID:               "dummy",
				ModuleType:       "Makefile",
				VersioningType:   "AnyStringVer",
				ReleasedVersions: []*VersionMetadata{},
			},
		},
		{
			name:         "valid version",
			gitRef:       "someCommitId",
			versionInput: "newVersion",
		},
		{
			name:         "adding existing version fails",
			gitRef:       "commitHash",
			versionInput: "1.1.0-1",
			hasError:     true,
		},
		{
			name:         "adding existing commit ref fails",
			gitRef:       "110b40f6862a2dc28f4045bd57d1832dfde10e66",
			versionInput: "1.1.0-2",
			hasError:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var versions *Versions
			if tc.customVersions != nil {
				versions = tc.customVersions
			} else {
				versions = parseVersions(t, sampleAnyStringVersion)
			}
			refTime, err := time.Parse(time.RFC3339, "2020-02-02T00:00:00Z")
			assert.NoError(t, err)

			_, err = versions.AddRelease(&refTime, tc.bumpType, tc.versionInput, tc.gitRef)

			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, 8, len(versions.ReleasedVersions), "expecting an additional entry in the versions")

				last := versions.ReleasedVersions[len(versions.ReleasedVersions)-1]
				assert.Equal(t, VersionMetadata{
					Number:    VersionString{tc.versionInput},
					Timestamp: refTime,
					CommitID:  tc.gitRef,
				}, *last, "the new version should be appended at the end")

				// Now check that when marshaling we actually write the new value out to the YAML
				marshaled, err := versions.Marshal()
				expected := fmt.Sprintf("%s  %s: 2020-02-02T00:00:00Z|%s\n", sampleAnyStringVersion, tc.versionInput, tc.gitRef)
				assert.NoError(t, err)
				assert.Equal(t, expected, string(marshaled))
			}
		})
	}
}

func TestReadFromFileSemVer(t *testing.T) {
	high, err := ReadFromFile("testdata/dummy-versions-semver.yaml")
	assert.NoError(t, err)

	assert.Equal(t, "testGroup:testModule", high.ID)
	assert.Equal(t, "SemVer", high.VersioningType)
	assert.Equal(t, "Makefile", high.ModuleType)
	assert.Equal(t, 4, len(high.ReleasedVersions))

	// Check the ordering of the underlying metadata slice
	assert.Equal(t, "0.0.0", high.ReleasedVersions[0].Number.String())
	assert.Equal(t, "1.1.1", high.ReleasedVersions[1].Number.String())
	assert.Equal(t, "1.2.0", high.ReleasedVersions[2].Number.String())
	assert.Equal(t, "2.0.0", high.ReleasedVersions[3].Number.String())
}

func TestReadFromFileAnyStringVer(t *testing.T) {
	high, err := ReadFromFile("testdata/dummy-versions-anystringver.yaml")
	assert.NoError(t, err)

	assert.Equal(t, "testGroup:testModuleAnyStringVer", high.ID)
	assert.Equal(t, "AnyStringVer", high.VersioningType)
	assert.Equal(t, "Makefile", high.ModuleType)
	assert.Equal(t, 8, len(high.ReleasedVersions))

	// Check the ordering of the underlying metadata slice
	assert.Equal(t, "someVersion", high.ReleasedVersions[0].Number.String())
	assert.Equal(t, "withQuotes", high.ReleasedVersions[1].Number.String())
	assert.Equal(t, "dont-@or+me", high.ReleasedVersions[2].Number.String())
	assert.Equal(t, "42", high.ReleasedVersions[3].Number.String())
	assert.Equal(t, "0.12pre6", high.ReleasedVersions[4].Number.String())
	assert.Equal(t, "2.11.25_5.4.84_3_linux_1", high.ReleasedVersions[5].Number.String())
	assert.Equal(t, "3.7.0_20200529", high.ReleasedVersions[6].Number.String())
	assert.Equal(t, "2.5", high.ReleasedVersions[7].Number.String())
}

func TestVersions_SaveToFile(t *testing.T) {
	vers := parseVersions(t, sampleSemVerVersion)
	testFile := "test-save-file.yml"

	err := vers.SaveToFile(testFile)
	defer os.Remove(testFile)
	assert.NoError(t, err)

	readBytes, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, sampleSemVerVersion, string(readBytes))
}

func TestCommentPreservation(t *testing.T) {
	// Read a file with comments
	high, err := ReadFromFile("testdata/dummy-versions-semver.yaml")
	assert.NoError(t, err)

	// Marshal it back to YAML
	marshaledYAML, err := high.Marshal()
	assert.NoError(t, err)

	// Log the result to see if comments are preserved
	t.Log("Marshaled YAML with potential comments:")
	t.Log(string(marshaledYAML))

	// The YAML should contain structured data
	assert.Contains(t, string(marshaledYAML), "# Auto-generated file: please edit with care.")
	assert.Contains(t, string(marshaledYAML), "# Identifies this module within the fat repo.")
	assert.Contains(t, string(marshaledYAML), "# The underlying tool to which building and releasing is handed off")
}

func TestCommentPreservationWithNewVersion(t *testing.T) {
	// Read a file with comments
	high, err := ReadFromFile("testdata/dummy-versions-semver.yaml")
	assert.NoError(t, err)

	// Add a new version
	refTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	newVersion, err := high.AddRelease(&refTime, BumpMinor, "", "newcommitid123")
	assert.NoError(t, err)
	assert.Equal(t, "2.1.0", newVersion.Number.String())

	// Marshal it back to YAML
	marshaledYAML, err := high.Marshal()
	assert.NoError(t, err)

	// Log the result to verify comments are preserved and new version is at the end
	t.Log("Marshaled YAML after adding new version:")
	t.Log(string(marshaledYAML))

	// Verify comments are still there
	assert.Contains(t, string(marshaledYAML), "# Auto-generated file: please edit with care.")
	assert.Contains(t, string(marshaledYAML), "# Identifies this module within the fat repo.")
	assert.Contains(t, string(marshaledYAML), "# The underlying tool to which building and releasing is handed off")

	// Verify new version is at the end
	assert.Contains(t, string(marshaledYAML), "2.1.0: 2023-01-01T00:00:00Z|newcommitid123")

	// Verify order is preserved by checking the last line with a version
	lines := strings.Split(string(marshaledYAML), "\n")
	var lastVersionLine string
	for _, line := range lines {
		if strings.Contains(line, ":") && strings.Contains(line, "|") && strings.HasPrefix(strings.TrimSpace(line), "2.1.0:") {
			lastVersionLine = line
		}
	}
	assert.Contains(t, lastVersionLine, "2.1.0:")
}

func TestVersionsCustomUnmarshaler(t *testing.T) {
	t.Run("SemVer versions unmarshaled correctly", func(t *testing.T) {
		yamlData := `id: testGroup:testModule
type: Makefile
versioning: SemVer
versions:
  0.0.0: 2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091
  1.1.1: 2019-04-01T16:06:07Z|934b40f6862a2dc28f4045bd57d1832dfde10e55
  v2.0.0: 2020-01-01T00:00:00Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e77
`

		versions, err := unmarshalVersions([]byte(yamlData))
		assert.NoError(t, err)

		// Verify basic fields
		assert.Equal(t, "testGroup:testModule", versions.ID)
		assert.Equal(t, "Makefile", versions.ModuleType)
		assert.Equal(t, "SemVer", versions.VersioningType)

		// Verify versions are properly parsed
		assert.Len(t, versions.ReleasedVersions, 3)

		// Check that version parsing worked correctly
		assert.Equal(t, "0.0.0", versions.ReleasedVersions[0].Number.String())
		assert.Equal(t, "1.1.1", versions.ReleasedVersions[1].Number.String())
		assert.Equal(t, "v2.0.0", versions.ReleasedVersions[2].Number.String())

		// Verify timestamps and commit IDs
		expectedTime1, _ := time.Parse(time.RFC3339, "2019-04-01T16:06:07Z")
		expectedTime2, _ := time.Parse(time.RFC3339, "2019-04-01T16:06:07Z")
		expectedTime3, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")

		assert.True(t, versions.ReleasedVersions[0].Timestamp.Equal(expectedTime1))
		assert.True(t, versions.ReleasedVersions[1].Timestamp.Equal(expectedTime2))
		assert.True(t, versions.ReleasedVersions[2].Timestamp.Equal(expectedTime3))

		assert.Equal(t, "675156f77a931aa40ceb115b763d9d1230b26091", versions.ReleasedVersions[0].CommitID)
		assert.Equal(t, "934b40f6862a2dc28f4045bd57d1832dfde10e55", versions.ReleasedVersions[1].CommitID)
		assert.Equal(t, "aa4b40f6862a2dc28f4045bd57d1832dfde10e77", versions.ReleasedVersions[2].CommitID)
	})

	t.Run("AnyStringVer versions unmarshaled correctly", func(t *testing.T) {
		yamlData := `id: testGroup:testModule
type: Makefile
versioning: AnyStringVer
versions:
  0.0.0: 2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091
  AnyString: 2019-04-01T16:06:07Z|934b40f6862a2dc28f4045bd57d1832dfde10e55
  a-zA-Z0-9.+_~@: 2019-04-02T16:06:07Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e55
`

		versions, err := unmarshalVersions([]byte(yamlData))
		assert.NoError(t, err)

		// Verify basic fields
		assert.Equal(t, "testGroup:testModule", versions.ID)
		assert.Equal(t, "Makefile", versions.ModuleType)
		assert.Equal(t, "AnyStringVer", versions.VersioningType)

		// Verify versions are properly parsed
		assert.Len(t, versions.ReleasedVersions, 3)

		// Check that version parsing worked correctly
		assert.Equal(t, "0.0.0", versions.ReleasedVersions[0].Number.String())
		assert.Equal(t, "AnyString", versions.ReleasedVersions[1].Number.String())
		assert.Equal(t, "a-zA-Z0-9.+_~@", versions.ReleasedVersions[2].Number.String())
	})

	t.Run("Numeric keys handled correctly", func(t *testing.T) {
		yamlData := `id: testGroup:testModule
type: Makefile
versioning: SemVer
versions:
  1.0: 2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091
  2.5: 2019-04-01T16:06:07Z|934b40f6862a2dc28f4045bd57d1832dfde10e55
`

		versions, err := unmarshalVersions([]byte(yamlData))
		assert.NoError(t, err)

		// Verify versions are properly parsed
		assert.Len(t, versions.ReleasedVersions, 2)

		// Check that numeric keys are converted to strings properly
		// Note: YAML parses 1.0 as integer 1, 2.5 as float 2.5
		assert.Equal(t, "1", versions.ReleasedVersions[0].Number.String())
		assert.Equal(t, "2.5", versions.ReleasedVersions[1].Number.String())
	})

	t.Run("Metadata and dependencies unmarshaled correctly", func(t *testing.T) {
		yamlData := `id: testGroup:testModule
type: Makefile
versioning: SemVer
versions:
  1.0.0: 2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091
metadata:
  annotations:
    key1: value1
    key2: value2
dependencies:
  - dep1
  - dep2
`

		versions, err := unmarshalVersions([]byte(yamlData))
		assert.NoError(t, err)

		// Verify metadata
		assert.NotNil(t, versions.Metadata)
		assert.NotNil(t, versions.Metadata.Annotations)
		assert.Equal(t, "value1", versions.Metadata.Annotations["key1"])
		assert.Equal(t, "value2", versions.Metadata.Annotations["key2"])

		// Verify dependencies
		assert.Len(t, versions.Dependencies, 2)
		assert.Equal(t, "dep1", versions.Dependencies[0])
		assert.Equal(t, "dep2", versions.Dependencies[1])
	})
}

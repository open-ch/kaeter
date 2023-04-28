package modules

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

const sampleSemVerVersion = `# Auto-generated file: please edit with care.

# Identifies this module within the fat repo.
id: testGroup:testModule
# The underlying tool to which building and releasing is handed off
type: Makefile
# Should this module be versioned with semantic or calendar versioning?
versioning: SemVer
# Version identifiers have the following format:
# <version string>: <RFC3339 formatted timestamp>|<commit ID>
versions:
    0.0.0: 2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091
    1.1.1: 2019-04-01T16:06:07Z|934b40f6862a2dc28f4045bd57d1832dfde10e55
    1.2.0: 2019-04-02T16:06:07Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e55
    2.0.0: 2020-01-01T00:00:00Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e66
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

func parseYamlOnly(t *testing.T, rawYAML string) *rawVersions {
	var deser rawVersions
	err := yaml.Unmarshal([]byte(rawYAML), &deser)
	assert.NoError(t, err)
	return &deser
}

func parseVersions(t *testing.T, rawYAML string) *Versions {
	parsed, err := UnmarshalVersions([]byte(rawYAML))
	assert.NoError(t, err)
	return parsed
}

// TestYamlV3Behavior just checks that some behavior of the lib is as expected
func TestYamlV3Behavior(t *testing.T) {
	// Unmarshall to a raw node
	var node yaml.Node
	err := yaml.Unmarshal([]byte(sampleSemVerVersion), &node)
	assert.NoError(t, err)
	// Check comments exist around
	assert.Equal(t, "# Auto-generated file: please edit with care.", node.HeadComment)

	var lazily rawVersions
	err = node.Decode(&lazily)
	assert.NoError(t, err)
	assert.Equal(t, parseYamlOnly(t, sampleSemVerVersion), &lazily)
}

func TestYamlV3UnmarshalFromStruct(t *testing.T) {
	tests := []struct {
		name                string
		yaml                string
		expectedRawVersions []rawKeyValuePair
	}{
		{
			name: "SemVer versions",
			yaml: sampleSemVerVersion,
			expectedRawVersions: []rawKeyValuePair{
				rawKeyValuePair{"0.0.0", "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091"},
				rawKeyValuePair{"1.1.1", "2019-04-01T16:06:07Z|934b40f6862a2dc28f4045bd57d1832dfde10e55"},
				rawKeyValuePair{"1.2.0", "2019-04-02T16:06:07Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e55"},
				rawKeyValuePair{"2.0.0", "2020-01-01T00:00:00Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e66"},
			},
		},
		{
			name: "AnyString versions",
			yaml: sampleAnyStringVersion,
			expectedRawVersions: []rawKeyValuePair{
				rawKeyValuePair{"0.0.0", "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091"},
				rawKeyValuePair{"AnyString", "2019-04-01T16:06:07Z|934b40f6862a2dc28f4045bd57d1832dfde10e55"},
				rawKeyValuePair{"a-zA-Z0-9.+_~@", "2019-04-02T16:06:07Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e55"},
				rawKeyValuePair{"whyNot", "2020-01-01T00:00:00Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e66"},

				rawKeyValuePair{"1.0.0-1", "2019-04-01T16:06:07Z|100156f77a931aa40ceb115b763d9d1230b26091"},
				rawKeyValuePair{"1.0.0-2", "2019-04-01T16:06:07Z|100b40f6862a2dc28f4045bd57d1832dfde10e55"},
				rawKeyValuePair{"1.1.0-1", "2020-01-01T00:00:00Z|110b40f6862a2dc28f4045bd57d1832dfde10e66"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := parseYamlOnly(t, tc.yaml)
			assert.NotNil(t, raw.RawReleasedVersions)

			rawVersions, err := raw.releasedVersionsMap()
			assert.NoError(t, err)

			assert.Equal(t, len(tc.expectedRawVersions), len(rawVersions))
			for i, rawVersion := range rawVersions {
				assert.Equal(t, tc.expectedRawVersions[i], rawVersion)
			}
		})
	}
}

func TestUnmarshalVersionsSemVer(t *testing.T) {
	raw := parseYamlOnly(t, sampleSemVerVersion)
	high, err := UnmarshalVersions([]byte(sampleSemVerVersion))

	assert.NoError(t, err)
	assert.Equal(t, raw.ID, high.ID)
	assert.Equal(t, raw.VersioningType, high.VersioningType)
	assert.Equal(t, raw.ModuleType, high.ModuleType)
	assert.Equal(t, (*Metadata)(nil), high.Metadata)
	assert.Equal(t, 4, len(high.ReleasedVersions))

	// Check the ordering of the underlying metadata slice
	assert.Equal(t, "0.0.0", high.ReleasedVersions[0].Number.String())
	assert.Equal(t, "1.1.1", high.ReleasedVersions[1].Number.String())
	assert.Equal(t, "1.2.0", high.ReleasedVersions[2].Number.String())
	assert.Equal(t, "2.0.0", high.ReleasedVersions[3].Number.String())
}

func TestUnmarshalVersionsAnyStringVer(t *testing.T) {
	raw := parseYamlOnly(t, sampleAnyStringVersion)
	high, err := UnmarshalVersions([]byte(sampleAnyStringVersion))

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
			name: "Expected metadata with multipe annotations",
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

		ver, err := UnmarshalVersions([]byte(versionsContent))

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

func TestVersionsSemVer_AddRelease(t *testing.T) {
	vers := parseVersions(t, sampleSemVerVersion)

	refTime, _ := time.Parse(time.RFC3339, "2020-02-02T00:00:00Z")

	assert.Equal(t, 4, len(vers.ReleasedVersions))

	_, err := vers.AddRelease(&refTime, BumpMinor, "", "someCommitId")
	assert.NoError(t, err)
	assert.Equal(t, 5, len(vers.ReleasedVersions))

	last := vers.ReleasedVersions[len(vers.ReleasedVersions)-1]
	assert.Equal(t, VersionMetadata{
		Number:    VersionNumber{2, 1, 0},
		Timestamp: refTime,
		CommitID:  "someCommitId",
	}, *last, "the new version should be appended at the end")

	// Now check that when marshaling we actually write the new value out to the YAML
	marshaled, err := vers.Marshal()
	expected := fmt.Sprintf("%s    2.1.0: 2020-02-02T00:00:00Z|someCommitId\n", sampleSemVerVersion)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(marshaled))
}

func TestVersionsSemVer_AddReleaseUserSpecifiedVersion(t *testing.T) {
	vers := parseVersions(t, sampleSemVerVersion)

	refTime, _ := time.Parse(time.RFC3339, "2020-02-02T00:00:00Z")

	assert.Equal(t, 4, len(vers.ReleasedVersions))

	_, err := vers.AddRelease(&refTime, BumpPatch, "5.6.7", "someCommitId")
	assert.NoError(t, err)
	assert.Equal(t, 5, len(vers.ReleasedVersions), "should not fail")

	last := vers.ReleasedVersions[len(vers.ReleasedVersions)-1]
	assert.Equal(t, VersionMetadata{
		Number:    &VersionNumber{5, 6, 7},
		Timestamp: refTime,
		CommitID:  "someCommitId",
	}, *last, "the new version should be appended at the end")

	// Now check that when marshaling we actually write the new value out to the YAML
	marshaled, err := vers.Marshal()
	expected := fmt.Sprintf("%s    5.6.7: 2020-02-02T00:00:00Z|someCommitId\n", sampleSemVerVersion)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(marshaled))
}

func TestVersionsSemVer_AddRelease_Failures(t *testing.T) {
	vers := parseVersions(t, sampleSemVerVersion)

	refTime, _ := time.Parse(time.RFC3339, "2020-02-02T00:00:00Z")

	_, err := vers.AddRelease(&refTime, BumpPatch, "", "")
	assert.Error(t, err)

	_, err = vers.AddRelease(&refTime, BumpPatch, "notParseableNumberVersion", "commitId")
	assert.Error(t, err)

	faulty := Versions{
		ID:               "dummy",
		ModuleType:       "Makefile",
		VersioningType:   "semver",
		ReleasedVersions: []*VersionMetadata{},
		documentNode:     nil,
	}
	_, err = faulty.AddRelease(&refTime, BumpPatch, "", "commitId")
	assert.Error(t, err)
}

// TODO refactor the add release tests above to use the test table style
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
				documentNode:     nil,
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
				expected := fmt.Sprintf("%s    %s: 2020-02-02T00:00:00Z|%s\n", sampleAnyStringVersion, tc.versionInput, tc.gitRef)
				assert.NoError(t, err)
				assert.Equal(t, expected, string(marshaled))
			}
		})
	}
}

func TestReadFromFileSemVer(t *testing.T) {
	high, err := ReadFromFile("test-data/dummy-versions-semver.yaml")
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
	high, err := ReadFromFile("test-data/dummy-versions-anystringver.yaml")
	assert.NoError(t, err)

	assert.Equal(t, "testGroup:testModuleAnyStringVer", high.ID)
	assert.Equal(t, "AnyStringVer", high.VersioningType)
	assert.Equal(t, "Makefile", high.ModuleType)
	assert.Equal(t, 7, len(high.ReleasedVersions))

	// Check the ordering of the underlying metadata slice
	assert.Equal(t, "someVersion", high.ReleasedVersions[0].Number.String())
	assert.Equal(t, "withQuotes", high.ReleasedVersions[1].Number.String())
	assert.Equal(t, "dont-@or+me", high.ReleasedVersions[2].Number.String())
	assert.Equal(t, "42", high.ReleasedVersions[3].Number.String())
	assert.Equal(t, "0.12pre6", high.ReleasedVersions[4].Number.String())
	assert.Equal(t, "2.11.25_5.4.84_3_linux_1", high.ReleasedVersions[5].Number.String())
	assert.Equal(t, "3.7.0_20200529", high.ReleasedVersions[6].Number.String())
}

func TestVersions_SaveToFile(t *testing.T) {
	vers := parseVersions(t, sampleSemVerVersion)
	testFile := "test-save-file.yml"

	err := vers.SaveToFile(testFile)
	defer os.Remove(testFile)
	assert.NoError(t, err)

	readBytes, err := ioutil.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, sampleSemVerVersion, string(readBytes))
}

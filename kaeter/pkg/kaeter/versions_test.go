package kaeter

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

const sampleVersion = `# Auto-generated file: please edit with care.

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

func getRaw(t *testing.T) *rawVersions {
	var deser rawVersions
	err := yaml.Unmarshal([]byte(sampleVersion), &deser)
	assert.NoError(t, err)
	return &deser
}

func getVersions(t *testing.T) *Versions {
	high, err := UnmarshalVersions([]byte(sampleVersion))
	assert.NoError(t, err)
	return high
}

// TestYamlV3Behavior just checks that some behavior of the lib is as expected
func TestYamlV3Behavior(t *testing.T) {
	// Unmarshall to a raw node
	var node yaml.Node
	err := yaml.Unmarshal([]byte(sampleVersion), &node)
	assert.NoError(t, err)
	// Check comments exist around
	assert.Equal(t, "# Auto-generated file: please edit with care.", node.HeadComment)

	var lazily rawVersions
	err = node.Decode(&lazily)
	assert.NoError(t, err)
	assert.Equal(t, getRaw(t), &lazily)
}

func TestRawUnmarshal(t *testing.T) {
	raw := getRaw(t)

	assert.NotNil(t, raw.RawReleasedVersions)

	versionsMap, err := raw.releasedVersionsMap()
	assert.NoError(t, err)

	assert.Equal(t, 4, len(versionsMap))
	assert.Equal(t, "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091", versionsMap["0.0.0"])
	assert.Equal(t, "2019-04-01T16:06:07Z|934b40f6862a2dc28f4045bd57d1832dfde10e55", versionsMap["1.1.1"])
	assert.Equal(t, "2019-04-02T16:06:07Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e55", versionsMap["1.2.0"])
	assert.Equal(t, "2020-01-01T00:00:00Z|aa4b40f6862a2dc28f4045bd57d1832dfde10e66", versionsMap["2.0.0"])
}

func TestUnmarshalVersions(t *testing.T) {
	raw := getRaw(t)
	high, err := UnmarshalVersions([]byte(sampleVersion))

	assert.NoError(t, err)
	assert.Equal(t, raw.ID, high.ID)
	assert.Equal(t, raw.VersioningType, high.VersioningType)
	assert.Equal(t, raw.ModuleType, high.ModuleType)
	assert.Equal(t, 4, len(high.ReleasedVersions))

	// Check the ordering of the underlying metadata slice
	assert.Equal(t, "0.0.0", high.ReleasedVersions[0].Number.GetVersionString())
	assert.Equal(t, "1.1.1", high.ReleasedVersions[1].Number.GetVersionString())
	assert.Equal(t, "1.2.0", high.ReleasedVersions[2].Number.GetVersionString())
	assert.Equal(t, "2.0.0", high.ReleasedVersions[3].Number.GetVersionString())

}

func TestVersions_Marshal(t *testing.T) {
	vers := getVersions(t)

	bytes, err := vers.Marshal()
	assert.NoError(t, err)

	fmt.Println(string(bytes))

	assert.Equal(t, sampleVersion, string(bytes))
}

func TestVersions_AddRelease(t *testing.T) {
	vers := getVersions(t)

	refTime, _ := time.Parse(time.RFC3339, "2020-02-02T00:00:00Z")

	assert.Equal(t, 4, len(vers.ReleasedVersions))

	_, err := vers.AddRelease(&refTime, false, true, "someCommitId")
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
	expected := fmt.Sprintf("%s    2.1.0: 2020-02-02T00:00:00Z|someCommitId\n", sampleVersion)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(marshaled))

}

func TestVersions_AddRelease_Failures(t *testing.T) {
	vers := getVersions(t)

	refTime, _ := time.Parse(time.RFC3339, "2020-02-02T00:00:00Z")

	_, err := vers.AddRelease(&refTime, true, true, "commitId")
	assert.Error(t, err)

	_, err = vers.AddRelease(&refTime, false, false, "")
	assert.Error(t, err)

	faulty := Versions{
		ID:               "dummy",
		ModuleType:       "Makefile",
		VersioningType:   "semver",
		ReleasedVersions: []*VersionMetadata{},
		documentNode:     nil,
	}
	_, err = faulty.AddRelease(&refTime, false, false, "commitId")
	assert.Error(t, err)

}

func TestReadFromFile(t *testing.T) {
	high, err := ReadFromFile("test-data/dummy-versions.yaml")
	assert.NoError(t, err)

	assert.Equal(t, "testGroup:testModule", high.ID)
	assert.Equal(t, "SemVer", high.VersioningType)
	assert.Equal(t, "Makefile", high.ModuleType)
	assert.Equal(t, 4, len(high.ReleasedVersions))

	// Check the ordering of the underlying metadata slice
	assert.Equal(t, "0.0.0", high.ReleasedVersions[0].Number.GetVersionString())
	assert.Equal(t, "1.1.1", high.ReleasedVersions[1].Number.GetVersionString())
	assert.Equal(t, "1.2.0", high.ReleasedVersions[2].Number.GetVersionString())
	assert.Equal(t, "2.0.0", high.ReleasedVersions[3].Number.GetVersionString())

}

func TestVersions_SaveToFile(t *testing.T) {
	vers := getVersions(t)
	testFile := "test-save-file.yml"

	err := vers.SaveToFile(testFile)
	assert.NoError(t, err)

	readBytes, err := ioutil.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, sampleVersion, string(readBytes))
	os.Remove(testFile)
}

func TestCompareVersionNumbers(t *testing.T) {
	// Reminder: cmpareVersionNumbers returns true if the first argument is smaller than the second
	assert.False(t,
		compareVersionNumbers(VersionNumber{0, 0, 0}, VersionNumber{0, 0, 0}))
	assert.True(t,
		compareVersionNumbers(VersionNumber{0, 0, 0}, VersionNumber{1, 0, 0}))
	assert.True(t,
		compareVersionNumbers(VersionNumber{0, 0, 0}, VersionNumber{0, 1, 0}))
	assert.True(t,
		compareVersionNumbers(VersionNumber{0, 0, 0}, VersionNumber{0, 0, 1}))

	assert.False(t,
		compareVersionNumbers(VersionNumber{10, 10, 10}, VersionNumber{1, 1, 1}))
	assert.False(t,
		compareVersionNumbers(VersionNumber{10, 10, 1}, VersionNumber{1, 1, 1}))
	assert.False(t,
		compareVersionNumbers(VersionNumber{10, 1, 1}, VersionNumber{1, 1, 1}))

	assert.True(t,
		compareVersionNumbers(VersionNumber{9, 1, 1}, VersionNumber{10, 0, 0}))
	assert.True(t,
		compareVersionNumbers(VersionNumber{9, 9, 1}, VersionNumber{9, 10, 0}))
	assert.True(t,
		compareVersionNumbers(VersionNumber{9, 9, 9}, VersionNumber{9, 9, 10}))

}

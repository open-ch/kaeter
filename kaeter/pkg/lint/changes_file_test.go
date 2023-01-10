package lint

import (
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createMockVersions(t *testing.T, rawVersions []string) []*kaeter.VersionMetadata {
	versions := make([]*kaeter.VersionMetadata, len(rawVersions))

	for i, v := range rawVersions {
		version, err := kaeter.UnmarshalVersionMetadata(v, "2006-01-02T15:04:05Z|deadbeef", kaeter.AnyStringVer)
		assert.NoError(t, err)
		versions[i] = version
	}

	return versions
}

func TestValidateCHANGESFile(t *testing.T) {
	tests := []struct {
		name        string
		versions    kaeter.Versions
		changesPath string
		valid       bool
	}{
		{
			name:        "no error when all OK",
			versions:    kaeter.Versions{ReleasedVersions: createMockVersions(t, []string{"v2.8", "v2.9"})},
			changesPath: "test-data/dummy-CHANGES",
			valid:       true,
		},
		{
			name:        "usernames are optional",
			versions:    kaeter.Versions{ReleasedVersions: createMockVersions(t, []string{"v2.8", "v2.9"})},
			changesPath: "test-data/dummy-CHANGES-nonames",
			valid:       true,
		},
		{
			name:        "fail if releases missing",
			versions:    kaeter.Versions{ReleasedVersions: createMockVersions(t, []string{"v1.2", "v1.3"})},
			changesPath: "test-data/dummy-CHANGES",
			valid:       false,
		},
		{
			name:        "fail if version is mentioned but not released",
			versions:    kaeter.Versions{ReleasedVersions: createMockVersions(t, []string{"v3.0"})},
			changesPath: "test-data/dummy-CHANGES-edgecase",
			valid:       false,
		},
		{
			name:        "fail if date is missing",
			versions:    kaeter.Versions{ReleasedVersions: createMockVersions(t, []string{"v2.9"})},
			changesPath: "test-data/dummy-CHANGES-nodate",
			valid:       false,
		},
		{
			name:        "fail if file can't be parsed",
			versions:    kaeter.Versions{},
			changesPath: "test-data/dummy-CHANGES-non-existant",
			valid:       false,
		},
	}

	for _, tt := range tests {
		err := validateCHANGESFile(tt.changesPath, &tt.versions)

		if tt.valid {
			assert.NoError(t, err, tt.name)
		} else {
			assert.Error(t, err, tt.name)
		}
	}
}

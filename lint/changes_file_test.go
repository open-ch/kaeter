package lint

import (
	"testing"

	"github.com/open-ch/kaeter/modules"

	"github.com/stretchr/testify/assert"
)

func createMockVersions(t *testing.T, rawVersions []string) []*modules.VersionMetadata {
	versions := make([]*modules.VersionMetadata, len(rawVersions))

	for i, v := range rawVersions {
		version, err := modules.UnmarshalVersionMetadata(v, "2006-01-02T15:04:05Z|deadbeef", modules.AnyStringVer)
		assert.NoError(t, err)
		versions[i] = version
	}

	return versions
}

func TestValidateCHANGESFile(t *testing.T) {
	tests := []struct {
		name        string
		versions    modules.Versions
		changesPath string
		valid       bool
	}{
		{
			name:        "no error when all OK",
			versions:    modules.Versions{ReleasedVersions: createMockVersions(t, []string{"v2.8", "v2.9"})},
			changesPath: "test-data/dummy-CHANGES",
			valid:       true,
		},
		{
			name:        "usernames are optional",
			versions:    modules.Versions{ReleasedVersions: createMockVersions(t, []string{"v2.8", "v2.9"})},
			changesPath: "test-data/dummy-CHANGES-nonames",
			valid:       true,
		},
		{
			name:        "fail if releases missing",
			versions:    modules.Versions{ReleasedVersions: createMockVersions(t, []string{"v1.2", "v1.3"})},
			changesPath: "test-data/dummy-CHANGES",
			valid:       false,
		},
		{
			name:        "fail if version is mentioned but not released",
			versions:    modules.Versions{ReleasedVersions: createMockVersions(t, []string{"v3.0"})},
			changesPath: "test-data/dummy-CHANGES-edgecase",
			valid:       false,
		},
		{
			name:        "fail if date is missing",
			versions:    modules.Versions{ReleasedVersions: createMockVersions(t, []string{"v2.9"})},
			changesPath: "test-data/dummy-CHANGES-nodate",
			valid:       false,
		},
		{
			name:        "fail if file can't be parsed",
			versions:    modules.Versions{},
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

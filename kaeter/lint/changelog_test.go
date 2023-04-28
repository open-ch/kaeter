package lint

import (
	"testing"
	"time"

	"github.com/open-ch/kaeter/kaeter/modules"

	"github.com/stretchr/testify/assert"
)

const sampleChangelog = `# CHANGELOG

## 1.2.0 - 26.5.20

- MDR-597 Proper _working_ initial release

## 1.1.0 - 26.5.20

Debugging release

## 1.0.0 - 18.5.20

Initial Release: cli stub for interfacing with Hashicorp Vault`

const sampleChangelogCalVer = `# CHANGELOG

## 20.05.3 - 26.5.20

 - MDR-597 Proper _working_ initial release

## 20.05.2 - 26.5.20

Debugging release

## 20.05.1 - 18.5.20

Initial Release: cli stub for interfacing with Hashicorp Vault
`

const sampleChangelogDashOneAnyVer = `# CHANGELOG

## 1.19.1-1 - 02.09.22 pfi

- Test release with a -1 in the version number.
`

func TestUnmarshalVersionString(t *testing.T) {
	tests := []struct {
		name                  string
		changelogLine         string
		expectedVersion       *modules.VersionNumber // for semver
		expectedVersionString *modules.VersionString // for anystringver
	}{
		{
			name:            "Regular semver",
			changelogLine:   "## 1.2.0 - 26.5.20",
			expectedVersion: &modules.VersionNumber{1, 2, 0},
		},
		{
			name:            "Date like semver",
			changelogLine:   "## 20.05.98 - 26.5.20",
			expectedVersion: &modules.VersionNumber{20, 5, 98},
		},
		{
			name:                  "anystringver",
			changelogLine:         "## someString - 26.5.20",
			expectedVersionString: &modules.VersionString{"someString"},
		},
		{
			name:                  "anystringver x.y.z+1 style",
			changelogLine:         "## 1.2.3+1 - 26.5.20",
			expectedVersionString: &modules.VersionString{"1.2.3+1"},
		},
		{
			name:                  "anystringver x.y.z-1 style",
			changelogLine:         "## 1.2.3-1 - 26.5.20",
			expectedVersionString: &modules.VersionString{"1.2.3-1"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			versionIdentifier, err := UnmarshalVersionString(test.changelogLine)

			assert.NoError(t, err)
			t.Logf("versionIdentifier: %s", versionIdentifier)
			if test.expectedVersion != nil {
				assert.IsType(t, &modules.VersionNumber{}, versionIdentifier)
				versionNumber := versionIdentifier.(*modules.VersionNumber)
				assert.Equal(t, test.expectedVersion.Major, versionNumber.Major)
				assert.Equal(t, test.expectedVersion.Minor, versionNumber.Minor)
				assert.Equal(t, test.expectedVersion.Micro, versionNumber.Micro)
			} else {
				assert.IsType(t, &modules.VersionString{}, versionIdentifier)
				versionString := versionIdentifier.(*modules.VersionString)
				assert.Equal(t, test.expectedVersionString, versionString)
			}
		})
	}
}

func TestUnmarshalTimestampString(t *testing.T) {
	tests := []struct {
		name          string
		changelogLine string
		expectedDate  time.Time
	}{
		{
			name:          "Date format parsing: 2.1.06",
			changelogLine: "## 1.2.0 - 9.5.20",
			expectedDate:  time.Date(2020, time.Month(5), 9, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "Date format parsing: 2.1.2006",
			changelogLine: "## 1.2.0 - 9.5.2020",
			expectedDate:  time.Date(2020, time.Month(5), 9, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "Date format parsing: 2.01.06",
			changelogLine: "## 1.2.0 - 9.05.20",
			expectedDate:  time.Date(2020, time.Month(5), 9, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "Date format parsing: 2.01.2006",
			changelogLine: "## 1.2.0 - 9.05.2020",
			expectedDate:  time.Date(2020, time.Month(5), 9, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "Date format parsing: 02.1.06",
			changelogLine: "## 1.2.0 - 09.5.20",
			expectedDate:  time.Date(2020, time.Month(5), 9, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "Date format parsing: 02.1.2006",
			changelogLine: "## 1.2.0 - 09.5.2020",
			expectedDate:  time.Date(2020, time.Month(5), 9, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "Date format parsing: 02.01.06",
			changelogLine: "## 1.2.0 - 09.05.20",
			expectedDate:  time.Date(2020, time.Month(5), 9, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "Date format parsing: 02.01.2006",
			changelogLine: "## 1.2.0 - 09.05.2020",
			expectedDate:  time.Date(2020, time.Month(5), 9, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "Parsing date after anystring ver",
			changelogLine: "## something - 9.5.22",
			expectedDate:  time.Date(2022, time.Month(5), 9, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "Parsing date after anystring ver including dashes",
			changelogLine: "## 1.2.3-alpha - 9.5.22",
			expectedDate:  time.Date(2022, time.Month(5), 9, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			timestamp, err := UnmarshalTimestampString(test.changelogLine)

			assert.NoError(t, err)
			assert.Equal(t, test.expectedDate.Day(), timestamp.Day())
			assert.Equal(t, test.expectedDate.Month(), timestamp.Month())
			assert.Equal(t, test.expectedDate.Year(), timestamp.Year())
		})
	}
}

func TestUnmarshalChangelogSemVer(t *testing.T) {
	changelog, err := UnmarshalChangelog(sampleChangelog)
	assert.NoError(t, err)

	entries := changelog.Entries
	assert.Len(t, entries, 3)

	assertDateMatches(t, &entries[0], 26, 5, 2020)
	assert.IsType(t, &modules.VersionNumber{}, entries[0].Version)
	assertVersionMatchesSemVer(t, &entries[0], "1.2.0")

	assertDateMatches(t, &entries[1], 26, 5, 2020)
	assert.IsType(t, &modules.VersionNumber{}, entries[1].Version)
	assertVersionMatchesSemVer(t, &entries[1], "1.1.0")

	assertDateMatches(t, &entries[2], 18, 5, 2020)
	assert.IsType(t, &modules.VersionNumber{}, entries[2].Version)
	assertVersionMatchesSemVer(t, &entries[2], "1.0.0")
}

func TestReadFromFileSemVer(t *testing.T) {
	changelog, err := ReadFromFile("test-data/dummy-changelog-SemVer")
	assert.NoError(t, err)

	entries := changelog.Entries
	assert.Len(t, entries, 3)

	assertDateMatches(t, &entries[0], 26, 5, 2020)
	assert.IsType(t, &modules.VersionNumber{}, entries[0].Version)
	assertVersionMatchesSemVer(t, &entries[0], "1.2.0")

	assertDateMatches(t, &entries[1], 26, 5, 2020)
	assert.IsType(t, &modules.VersionNumber{}, entries[1].Version)
	assertVersionMatchesSemVer(t, &entries[1], "1.1.0")

	assertDateMatches(t, &entries[2], 18, 5, 2020)
	assert.IsType(t, &modules.VersionNumber{}, entries[2].Version)
	assertVersionMatchesSemVer(t, &entries[2], "1.0.0")
}

func TestReadFromFileCalVer(t *testing.T) {
	changelog, err := ReadFromFile("test-data/dummy-changelog-CalVer")
	assert.NoError(t, err)

	entries := changelog.Entries
	assert.Len(t, entries, 3)

	assertDateMatches(t, &entries[0], 26, 5, 2020)
	assert.IsType(t, &modules.VersionNumber{}, entries[0].Version)
	assertVersionMatchesSemVer(t, &entries[0], "20.5.3")

	assertDateMatches(t, &entries[1], 26, 5, 2020)
	assert.IsType(t, &modules.VersionNumber{}, entries[1].Version)
	assertVersionMatchesSemVer(t, &entries[1], "20.5.2")

	assertDateMatches(t, &entries[2], 18, 5, 2020)
	assert.IsType(t, &modules.VersionNumber{}, entries[2].Version)
	assertVersionMatchesSemVer(t, &entries[2], "20.5.1")
}

func TestUnmarshalChangelogCalVer(t *testing.T) {
	changelog, err := UnmarshalChangelog(sampleChangelogCalVer)
	assert.NoError(t, err)

	entries := changelog.Entries
	assert.Len(t, entries, 3)

	assertDateMatches(t, &entries[0], 26, 5, 2020)
	assert.IsType(t, &modules.VersionNumber{}, entries[0].Version)
	assertVersionMatchesSemVer(t, &entries[0], "20.5.3")

	assertDateMatches(t, &entries[1], 26, 5, 2020)
	assert.IsType(t, &modules.VersionNumber{}, entries[1].Version)
	assertVersionMatchesSemVer(t, &entries[1], "20.5.2")

	assertDateMatches(t, &entries[2], 18, 5, 2020)
	assert.IsType(t, &modules.VersionNumber{}, entries[2].Version)
	assertVersionMatchesSemVer(t, &entries[2], "20.5.1")

	assert.Equal(t, 3, len(entries))
}

func TestAnyVerDashNumber(t *testing.T) {
	changelog, err := UnmarshalChangelog(sampleChangelogDashOneAnyVer)
	assert.NoError(t, err)

	entries := changelog.Entries
	t.Logf("Entries: %v", entries)
	assert.Len(t, entries, 1)

	entry := &entries[0]
	assertDateMatches(t, entry, 2, 9, 2022)
	assert.IsType(t, &modules.VersionString{}, entry.Version)
	assertVersionMatchesSemVer(t, entry, "1.19.1-1")
}

func assertDateMatches(t *testing.T, e *ChangelogEntry, day, month, year int) {
	assert.NotNil(t, e.Timestamp)
	assert.Equal(t, day, e.Timestamp.Day())
	assert.Equal(t, time.Month(month), e.Timestamp.Month())
	assert.Equal(t, year, e.Timestamp.Year())
}

func assertVersionMatchesSemVer(t *testing.T, e *ChangelogEntry, versionAsString string) {
	assert.NotNil(t, e.Timestamp)
	assert.Equal(t, versionAsString, e.Version.String())
}

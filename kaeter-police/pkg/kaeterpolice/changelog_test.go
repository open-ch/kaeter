package kaeterpolice

import (
	"testing"
	"time"

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

func TestUnmarshalVersionString(t *testing.T) {
	versionNumber, err := UnmarshalVersionString("## 1.2.0 - 26.5.20")
	assert.NoError(t, err)
	assert.Equal(t, int16(1), versionNumber.Major)
	assert.Equal(t, int16(2), versionNumber.Minor)
	assert.Equal(t, int16(0), versionNumber.Micro)

	versionNumber, err = UnmarshalVersionString("## 20.05.98 - 26.5.20")
	assert.NoError(t, err)
	assert.Equal(t, int16(20), versionNumber.Major)
	assert.Equal(t, int16(5), versionNumber.Minor)
	assert.Equal(t, int16(98), versionNumber.Micro)
}

func TestUnmarshalTimestampString(t *testing.T) {
	// 2.1.06
	timestamp, err := UnmarshalTimestampString("## 1.2.0 - 9.5.20")
	assert.NoError(t, err)
	assert.Equal(t, 9, timestamp.Day())
	assert.Equal(t, time.Month(5), timestamp.Month())
	assert.Equal(t, 2020, timestamp.Year())

	// 2.1.2006
	timestamp, err = UnmarshalTimestampString("## 1.2.0 - 9.5.2020")
	assert.NoError(t, err)
	assert.Equal(t, 9, timestamp.Day())
	assert.Equal(t, time.Month(5), timestamp.Month())
	assert.Equal(t, 2020, timestamp.Year())

	// 2.01.06
	timestamp, err = UnmarshalTimestampString("## 1.2.0 - 9.05.20")
	assert.NoError(t, err)
	assert.Equal(t, 9, timestamp.Day())
	assert.Equal(t, time.Month(5), timestamp.Month())
	assert.Equal(t, 2020, timestamp.Year())

	// 2.01.2006
	timestamp, err = UnmarshalTimestampString("## 1.2.0 - 9.05.2020")
	assert.NoError(t, err)
	assert.Equal(t, 9, timestamp.Day())
	assert.Equal(t, time.Month(5), timestamp.Month())
	assert.Equal(t, 2020, timestamp.Year())

	// 02.1.06
	timestamp, err = UnmarshalTimestampString("## 1.2.0 - 09.5.20")
	assert.NoError(t, err)
	assert.Equal(t, 9, timestamp.Day())
	assert.Equal(t, time.Month(5), timestamp.Month())
	assert.Equal(t, 2020, timestamp.Year())

	// 02.1.2006
	timestamp, err = UnmarshalTimestampString("## 1.2.0 - 09.5.2020")
	assert.NoError(t, err)
	assert.Equal(t, 9, timestamp.Day())
	assert.Equal(t, time.Month(5), timestamp.Month())
	assert.Equal(t, 2020, timestamp.Year())

	// 02.01.06
	timestamp, err = UnmarshalTimestampString("## 1.2.0 - 09.05.20")
	assert.NoError(t, err)
	assert.Equal(t, 9, timestamp.Day())
	assert.Equal(t, time.Month(5), timestamp.Month())
	assert.Equal(t, 2020, timestamp.Year())

	// 02.01.2006
	timestamp, err = UnmarshalTimestampString("## 1.2.0 - 09.05.2020")
	assert.NoError(t, err)
	assert.Equal(t, 9, timestamp.Day())
	assert.Equal(t, time.Month(5), timestamp.Month())
	assert.Equal(t, 2020, timestamp.Year())
}

func TestUnmarshalChangelogSemVer(t *testing.T) {
	changelog, err := UnmarshalChangelog(sampleChangelog)
	assert.NoError(t, err)

	entries := changelog.Entries

	assert.Equal(t, int16(1), entries[0].Version.Major)
	assert.Equal(t, int16(2), entries[0].Version.Minor)
	assert.Equal(t, int16(0), entries[0].Version.Micro)
	assert.Equal(t, 26, entries[0].Timestamp.Day())
	assert.Equal(t, time.Month(5), entries[0].Timestamp.Month())
	assert.Equal(t, 2020, entries[0].Timestamp.Year())

	assert.Equal(t, int16(1), entries[1].Version.Major)
	assert.Equal(t, int16(1), entries[1].Version.Minor)
	assert.Equal(t, int16(0), entries[1].Version.Micro)
	assert.Equal(t, 26, entries[1].Timestamp.Day())
	assert.Equal(t, time.Month(5), entries[1].Timestamp.Month())
	assert.Equal(t, 2020, entries[1].Timestamp.Year())

	assert.Equal(t, int16(1), entries[2].Version.Major)
	assert.Equal(t, int16(0), entries[2].Version.Minor)
	assert.Equal(t, int16(0), entries[2].Version.Micro)
	assert.Equal(t, 18, entries[2].Timestamp.Day())
	assert.Equal(t, time.Month(5), entries[2].Timestamp.Month())
	assert.Equal(t, 2020, entries[2].Timestamp.Year())

	assert.Equal(t, 3, len(entries))
}

func TestReadFromFileSemVer(t *testing.T) {
	changelog, err := ReadFromFile("test-data/dummy-changelog-SemVer")
	assert.NoError(t, err)

	entries := changelog.Entries

	assert.Equal(t, int16(1), entries[0].Version.Major)
	assert.Equal(t, int16(2), entries[0].Version.Minor)
	assert.Equal(t, int16(0), entries[0].Version.Micro)
	assert.Equal(t, 26, entries[0].Timestamp.Day())
	assert.Equal(t, time.Month(5), entries[0].Timestamp.Month())
	assert.Equal(t, 2020, entries[0].Timestamp.Year())

	assert.Equal(t, int16(1), entries[1].Version.Major)
	assert.Equal(t, int16(1), entries[1].Version.Minor)
	assert.Equal(t, int16(0), entries[1].Version.Micro)
	assert.Equal(t, 26, entries[1].Timestamp.Day())
	assert.Equal(t, time.Month(5), entries[1].Timestamp.Month())
	assert.Equal(t, 2020, entries[1].Timestamp.Year())

	assert.Equal(t, int16(1), entries[2].Version.Major)
	assert.Equal(t, int16(0), entries[2].Version.Minor)
	assert.Equal(t, int16(0), entries[2].Version.Micro)
	assert.Equal(t, 18, entries[2].Timestamp.Day())
	assert.Equal(t, time.Month(5), entries[2].Timestamp.Month())
	assert.Equal(t, 2020, entries[2].Timestamp.Year())

	assert.Equal(t, 3, len(entries))
}

func TestReadFromFileCalVer(t *testing.T) {
	changelog, err := ReadFromFile("test-data/dummy-changelog-CalVer")
	assert.NoError(t, err)

	entries := changelog.Entries

	assert.Equal(t, int16(20), entries[0].Version.Major)
	assert.Equal(t, int16(5), entries[0].Version.Minor)
	assert.Equal(t, int16(3), entries[0].Version.Micro)
	assert.Equal(t, 26, entries[0].Timestamp.Day())
	assert.Equal(t, time.Month(5), entries[0].Timestamp.Month())
	assert.Equal(t, 2020, entries[0].Timestamp.Year())

	assert.Equal(t, int16(20), entries[1].Version.Major)
	assert.Equal(t, int16(5), entries[1].Version.Minor)
	assert.Equal(t, int16(2), entries[1].Version.Micro)
	assert.Equal(t, 26, entries[1].Timestamp.Day())
	assert.Equal(t, time.Month(5), entries[1].Timestamp.Month())
	assert.Equal(t, 2020, entries[1].Timestamp.Year())

	assert.Equal(t, int16(20), entries[2].Version.Major)
	assert.Equal(t, int16(5), entries[2].Version.Minor)
	assert.Equal(t, int16(1), entries[2].Version.Micro)
	assert.Equal(t, 18, entries[2].Timestamp.Day())
	assert.Equal(t, time.Month(5), entries[2].Timestamp.Month())
	assert.Equal(t, 2020, entries[2].Timestamp.Year())

	assert.Equal(t, 3, len(entries))
}

func TestUnmarshalChangelogCalVer(t *testing.T) {
	changelog, err := UnmarshalChangelog(sampleChangelogCalVer)
	assert.NoError(t, err)

	entries := changelog.Entries

	assert.Equal(t, int16(20), entries[0].Version.Major)
	assert.Equal(t, int16(5), entries[0].Version.Minor)
	assert.Equal(t, int16(3), entries[0].Version.Micro)
	assert.Equal(t, 26, entries[0].Timestamp.Day())
	assert.Equal(t, time.Month(5), entries[0].Timestamp.Month())
	assert.Equal(t, 2020, entries[0].Timestamp.Year())

	assert.Equal(t, int16(20), entries[1].Version.Major)
	assert.Equal(t, int16(5), entries[1].Version.Minor)
	assert.Equal(t, int16(2), entries[1].Version.Micro)
	assert.Equal(t, 26, entries[1].Timestamp.Day())
	assert.Equal(t, time.Month(5), entries[1].Timestamp.Month())
	assert.Equal(t, 2020, entries[1].Timestamp.Year())

	assert.Equal(t, int16(20), entries[2].Version.Major)
	assert.Equal(t, int16(5), entries[2].Version.Minor)
	assert.Equal(t, int16(1), entries[2].Version.Micro)
	assert.Equal(t, 18, entries[2].Timestamp.Day())
	assert.Equal(t, time.Month(5), entries[2].Timestamp.Month())
	assert.Equal(t, 2020, entries[2].Timestamp.Year())

	assert.Equal(t, 3, len(entries))
}

package modules

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	initNumber = VersionNumber{0, 0, 0}
	testNumber = VersionNumber{2, 3, 4}
	epoch      = time.Date(1970, 1, 10, 0, 0, 0, 0, time.UTC)
	recent     = time.Date(2020, 11, 20, 0, 0, 0, 0, time.UTC)
	moreRecent = time.Date(2020, 12, 30, 0, 0, 0, 0, time.UTC)
)

func TestNextSemanticVersion(t *testing.T) {
	assert.Equal(t, VersionNumber{0, 0, 1}, initNumber.nextSemanticVersion(BumpPatch))
	assert.Equal(t, VersionNumber{0, 1, 0}, initNumber.nextSemanticVersion(BumpMinor))
	assert.Equal(t, VersionNumber{1, 0, 0}, initNumber.nextSemanticVersion(BumpMajor))

	assert.Equal(t, VersionNumber{2, 3, 5}, testNumber.nextSemanticVersion(BumpPatch))
	assert.Equal(t, VersionNumber{2, 4, 0}, testNumber.nextSemanticVersion(BumpMinor))
	assert.Equal(t, VersionNumber{3, 0, 0}, testNumber.nextSemanticVersion(BumpMajor))
}

func TestNextCalendarVersion(t *testing.T) {
	assert.Equal(t, VersionNumber{70, 1, 0}, initNumber.nextCalendarVersion(&epoch))
	assert.Equal(t, VersionNumber{20, 11, 0}, initNumber.nextCalendarVersion(&recent))

	epochVers := VersionNumber{70, 1, 0}

	assert.Equal(t, VersionNumber{70, 1, 1},
		epochVers.nextCalendarVersion(&epoch),
		"Should increment the micro if still in the same month.")

	afterEpochVers := VersionNumber{70, 1, 1}
	assert.Equal(t, VersionNumber{20, 11, 0}, afterEpochVers.nextCalendarVersion(&recent))

	recentVers := VersionNumber{20, 11, 0}
	assert.Equal(t, VersionNumber{20, 12, 0}, recentVers.nextCalendarVersion(&moreRecent))
}

func TestFromVersionString(t *testing.T) {
	parsed, err := UnmarshalVersionString("2.3.4", SemVer)
	assert.NoError(t, err)
	assert.Equal(t, &testNumber, parsed)

	_, err = UnmarshalVersionString("2.3.4.5", SemVer)
	assert.Error(t, err)

	_, err = UnmarshalVersionString("2.3", SemVer)
	assert.Error(t, err)
}

func TestToVersionString(t *testing.T) {
	assert.Equal(t, "2.3.4", testNumber.GetVersionString())
}

func TestUnmarshallVersionMetadata(t *testing.T) {
	unmarsh, err := UnmarshalVersionMetadata("2.3.4", "2006-01-02T15:04:05Z|deadbeef", SemVer)
	assert.NoError(t, err)
	assert.Equal(t, &VersionMetadata{
		Number:    &VersionNumber{2, 3, 4},
		Timestamp: time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
		CommitID:  "deadbeef",
	}, unmarsh)

	unmarsh, err = UnmarshalVersionMetadata("2.3", "2006-01-02T15:04:05Z|deadbeef", SemVer)
	assert.Error(t, err)
	assert.Nil(t, unmarsh)

	unmarsh, err = UnmarshalVersionMetadata("2.3.4", "2006-01-02T15:04:05|deadbeef", SemVer)
	assert.Error(t, err)
	assert.Nil(t, unmarsh)
}

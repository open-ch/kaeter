package modules

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
)

// This file contains things relating to the metadata of a single version of a module.

// VersionNumber compose a semver.Version
type VersionNumber struct {
	semver.Version
}

// VersionIdentifier represents a 'version number' when SemVer/CalVer are in use or an arbitrary string otherwise.
type VersionIdentifier interface {
	String() string
}

// VersionMetadata contains some basic information assorted to a version number
type VersionMetadata struct {
	Number    VersionIdentifier
	Timestamp time.Time
	CommitID  string
}

// SemVerBump defines the options for semver bumps,
// default to patch, and allow major, minor.
type SemVerBump int

const (
	// BumpPatch bumps the patch version (z) number (default)
	BumpPatch SemVerBump = iota // 0
	// BumpMinor bumps the minor version (y) number
	BumpMinor // 1
	// BumpMajor bumps the major version (x) number
	BumpMajor // 2
)

const versionStringRegex = "^[a-zA-Z0-9.+_~@-]+$"

// VersionString a version represented by an arbitrary string
type VersionString struct {
	Version string
}

func (vs VersionString) String() string {
	return vs.Version
}

// UnmarshalVersionMetadata builds a VersionMetadata struct from the two strings containing the raw version and release data
func UnmarshalVersionMetadata(versionStr string, releaseData string, versioningScheme string) (*VersionMetadata, error) {
	vNum, err := UnmarshalVersionString(versionStr, versioningScheme)
	if err != nil {
		return nil, err
	}

	timestamp, commit, err := unmarshalReleaseData(releaseData)
	if err != nil {
		return nil, err
	}
	vMeta := VersionMetadata{vNum, *timestamp, commit}

	return &vMeta, nil
}

// UnmarshalVersionString builds a VersionIdentifier struct from a string (x.y.z)
func UnmarshalVersionString(versionStr string, versioningScheme string) (VersionIdentifier, error) {
	if strings.ToLower(versioningScheme) == AnyStringVer {
		match, _ := regexp.MatchString(versionStringRegex, "versionStr")
		if match {
			return &VersionString{versionStr}, nil
		}
		return nil, fmt.Errorf("user specified version does not match regex %s: %s ", versionStringRegex, versionStr)
	}
	v, err := semver.NewVersion(versionStr)
	if err != nil {
		return nil, err
	}
	return &VersionNumber{*v}, nil
}

// NewVersion creates a Major.Minor.Patch VersionNumber
func NewVersion(major, minor, micro uint64) *VersionNumber {
	return &VersionNumber{*semver.New(major, minor, micro, "", "")}
}

func (vn *VersionNumber) String() string {
	return vn.Original()
}

// unmarshalReleaseData extracts the timestamp and the commit id from a string of the form 2006-01-02T15:04:05Z07:00|<commitId>
func unmarshalReleaseData(releaseData string) (*time.Time, string, error) {
	splitData := strings.Split(releaseData, "|")
	if len(splitData) < 2 {
		return nil, "", fmt.Errorf("cannot parse release data: %s", releaseData)
	}

	theTime, err := time.Parse(time.RFC3339, splitData[0])
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse release data from string %s: %s", releaseData, err)
	}
	return &theTime, splitData[1], err
}

func (v *VersionMetadata) marshalReleaseData() string {
	return v.Timestamp.Format(time.RFC3339) + "|" + v.CommitID
}

// nextCalendarVersion computes the next calendar version according to the YY.MM.MICRO convention, where
// the micro number corresponds to the build number, and NOT the day of the month.
func (vn *VersionNumber) nextCalendarVersion(refTime *time.Time) *VersionNumber {
	major := uint64(refTime.Year() % 100)
	minor := uint64(refTime.Month())
	if vn.Major() == major && vn.Minor() == minor {
		// Increment the micro
		return &VersionNumber{vn.IncPatch()}
	}
	return &VersionNumber{*semver.New(major, minor, 0, "", "")}
}

// nextSemanticVersion computes the next version according to semantic versioning and whether
// the major or minor flags are set.
func (vn *VersionNumber) nextSemanticVersion(bumpType SemVerBump) *VersionNumber {
	switch bumpType {
	case BumpMajor:
		return &VersionNumber{vn.IncMajor()}
	case BumpMinor:
		return &VersionNumber{vn.IncMinor()}
	default:
		return &VersionNumber{vn.IncPatch()}
	}
}

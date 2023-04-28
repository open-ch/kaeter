package kaeter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// This file contains things relating to the metadata of a single version of a module.

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

// VersionNumber contains the components of a version. Note that in the case of CalVer,
// Major and Minor become the year and the month, respectively
type VersionNumber struct {
	Major int16
	Minor int16
	Micro int16
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

func (vn VersionNumber) String() string {
	return fmt.Sprintf("%d.%d.%d", vn.Major, vn.Minor, vn.Micro)
}

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

// UnmarshalVersionString builds a VersionNumber struct from a string (x.y.z)
func UnmarshalVersionString(versionStr string, versioningScheme string) (VersionIdentifier, error) {
	if strings.ToLower(versioningScheme) == AnyStringVer {
		match, _ := regexp.MatchString(versionStringRegex, "versionStr")
		if match {
			return &VersionString{versionStr}, nil
		}
		return nil, fmt.Errorf("user specified version does not match regex %s: %s ", versionStringRegex, versionStr)
	}
	return unmarshalNumberTripletVersionString(versionStr)
}

func unmarshalNumberTripletVersionString(versionStr string) (*VersionNumber, error) {
	split := strings.Split(versionStr, ".")
	if len(split) != 3 {
		return nil, fmt.Errorf("version strings must be of the form MAJOR.MINOR.MICRO, nothing else. Was: %s", versionStr)
	}
	// I really wish I was writing Scala here...
	// TODO proper error handling: some utility package somewhere to map slices of strings to slices if ints?
	major, _ := strconv.Atoi(split[0])
	minor, _ := strconv.Atoi(split[1])
	micro, _ := strconv.Atoi(split[2])
	return &VersionNumber{int16(major), int16(minor), int16(micro)}, nil
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

// GetVersionString returns a string of the format 'Major.Minor.Micro'
// TODO can be removed to the profit of String on the interface?
func (vn *VersionNumber) GetVersionString() string {
	return strconv.Itoa(int(vn.Major)) + "." + strconv.Itoa(int(vn.Minor)) + "." + strconv.Itoa(int(vn.Micro))
}

func (v *VersionMetadata) marshalReleaseData() string {
	return v.Timestamp.Format(time.RFC3339) + "|" + v.CommitID
}

// nextCalendarVersion computes the next calendar version according to the YY.MM.MICRO convention, where
// the micro number corresponds to the build number, and NOT the day of the month.
func (vn *VersionNumber) nextCalendarVersion(refTime *time.Time) VersionNumber {
	currentYearMonth := VersionNumber{int16(refTime.Year() % 100), int16(refTime.Month()), 0}
	if vn.Major == currentYearMonth.Major && vn.Minor == currentYearMonth.Minor {
		// Increment the micro
		return VersionNumber{vn.Major, vn.Minor, vn.Micro + 1}
	}
	return currentYearMonth
}

// nextSemanticVersion computes the next version according to semantic versioning and whether
// the major or minor flags are set.
func (vn *VersionNumber) nextSemanticVersion(bumpType SemVerBump) VersionNumber {
	switch bumpType {
	case BumpMajor:
		return VersionNumber{vn.Major + 1, 0, 0}
	case BumpMinor:
		return VersionNumber{vn.Major, vn.Minor + 1, 0}
	default:
		return VersionNumber{vn.Major, vn.Minor, vn.Micro + 1}
	}
}

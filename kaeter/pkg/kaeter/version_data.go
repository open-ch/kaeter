package kaeter

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// This file contains things relating to the metadata of a single version of a module.

// VersionMetadata contains some basic information assorted to a version number
type VersionMetadata struct {
	Number    VersionNumber
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

func (vn *VersionNumber) String() string {
    return fmt.Sprintf("%d.%d.%d", vn.Major, vn.Minor, vn.Micro)
}

// UnmarshalVersionMetadata builds a VersionMetadata struct from the two strings containing the raw version and release data
func UnmarshalVersionMetadata(versionStr string, releaseData string) (*VersionMetadata, error) {
	vNum, err := UnmarshalVersionString(versionStr)
	if err != nil {
		return nil, err
	}

	timestamp, commit, err := unmarshalReleaseData(releaseData)
	if err != nil {
		return nil, err
	}
	return &VersionMetadata{*vNum, *timestamp, commit}, nil
}

// UnmarshalVersionString builds a VersionNumber struct from a string (x.y.z)
func UnmarshalVersionString(versionStr string) (*VersionNumber, error) {
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

	time, err := time.Parse(time.RFC3339, splitData[0])
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse release data from string %s: %s", releaseData, err)
	}
	return &time, splitData[1], err
}

// GetVersionString returns a string of the format 'Major.Minor.Micro'
func (v *VersionNumber) GetVersionString() string {
	return strconv.Itoa(int(v.Major)) + "." + strconv.Itoa(int(v.Minor)) + "." + strconv.Itoa(int(v.Micro))
}

func (v *VersionMetadata) marshalReleaseData() string {
	return v.Timestamp.Format(time.RFC3339) + "|" + v.CommitID
}

// nextCalendarVersion computes the next calendar version according to the YY.MM.MICRO convention, where
// the micro number corresponds to the build number, and NOT the day of the month.
func (v *VersionNumber) nextCalendarVersion(refTime *time.Time) VersionNumber {
	currentYearMonth := VersionNumber{int16(refTime.Year() % 100), int16(refTime.Month()), 0}
	if v.Major == currentYearMonth.Major && v.Minor == currentYearMonth.Minor {
		// Increment the micro
		return VersionNumber{v.Major, v.Minor, v.Micro + 1}
	}
	return currentYearMonth
}

// nextSemanticVersion computes the next version according to semantic versioning and whether
// the major or minor flags are set.
// Note that this method will unapologetically panic if both flags are true.
func (v *VersionNumber) nextSemanticVersion(bumpMajor bool, bumpMinor bool) VersionNumber {
	if bumpMajor && bumpMinor {
		// Doing a 'panic' here, because this should _REALLY_ have been checked earlier.
		panic(fmt.Errorf("cannot bump both major and minor"))
	}
	if bumpMajor {
		return VersionNumber{v.Major + 1, 0, 0}
	}
	if bumpMinor {
		return VersionNumber{v.Major, v.Minor + 1, 0}
	}
	return VersionNumber{v.Major, v.Minor, v.Micro + 1}
}

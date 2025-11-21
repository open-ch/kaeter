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
	Tags      []string // Optional custom tags for the version
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

const expectedReleaseDataChunks = 2
const versionStringRegex = "^[a-zA-Z0-9.+_~@-]+$"
const yearYYFormatModulo = 100

// VersionString a version represented by an arbitrary string
type VersionString struct {
	Version string
}

func (vs VersionString) String() string {
	return vs.Version
}

// UnmarshalVersionMetadata builds a VersionMetadata struct from the two strings containing the raw version and release data
func UnmarshalVersionMetadata(versionStr, releaseData, versioningScheme string) (*VersionMetadata, error) {
	vNum, err := UnmarshalVersionString(versionStr, versioningScheme)
	if err != nil {
		return nil, err
	}

	timestamp, commit, tags, err := unmarshalReleaseData(releaseData)
	if err != nil {
		return nil, err
	}
	vMeta := VersionMetadata{
		Number:    vNum,
		Timestamp: *timestamp,
		CommitID:  commit,
		Tags:      tags,
	}

	return &vMeta, nil
}

// UnmarshalVersionString builds a VersionIdentifier struct from a string (x.y.z)
// revive:disable:function-result-limit
func UnmarshalVersionString(versionStr, versioningScheme string) (VersionIdentifier, error) {
	if strings.EqualFold(versioningScheme, AnyStringVer) {
		match, err := regexp.MatchString(versionStringRegex, "versionStr")
		if err != nil {
			return nil, err
		}
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

// unmarshalReleaseData extracts the timestamp, commit id, and optional tags from a string
// Format: 2006-01-02T15:04:05Z07:00|<commitId> or 2006-01-02T15:04:05Z07:00|<commitId>|<tag1,tag2,...>
// Tags are optional and backward compatible
func unmarshalReleaseData(releaseData string) (*time.Time, string, []string, error) {
	splitData := strings.Split(releaseData, "|")
	if len(splitData) < expectedReleaseDataChunks {
		return nil, "", nil, fmt.Errorf("cannot parse release data: %s", releaseData)
	}

	theTime, err := time.Parse(time.RFC3339, splitData[0])
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to parse release data from string %s: %w", releaseData, err)
	}

	commitID := splitData[1]
	var tags []string

	// Parse optional tags (backward compatible)
	if len(splitData) > 2 && splitData[2] != "" {
		// Split comma-separated tags and trim whitespace
		rawTags := strings.Split(splitData[2], ",")
		tags = make([]string, 0, len(rawTags))
		for _, tag := range rawTags {
			trimmed := strings.TrimSpace(tag)
			if trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
	}

	return &theTime, commitID, tags, nil
}

func (v *VersionMetadata) marshalReleaseData() string {
	result := v.Timestamp.Format(time.RFC3339) + "|" + v.CommitID

	// Append tags if present (backward compatible)
	if len(v.Tags) > 0 {
		result += "|" + strings.Join(v.Tags, ",")
	}

	return result
}

// nextCalendarVersion computes the next calendar version according to the YY.MM.MICRO convention, where
// the micro number corresponds to the build number, and NOT the day of the month.
func (vn *VersionNumber) nextCalendarVersion(refTime *time.Time) *VersionNumber {
	major := uint64(refTime.Year() % yearYYFormatModulo) //nolint:gosec // No overflow: year will be positive and small
	minor := uint64(refTime.Month())                     //nolint:gosec // No overflow: month is between 1 and 12
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

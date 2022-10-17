package kaeterpolice

import (
	"fmt"
	"io/ioutil"
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"
	"regexp"
	"time"
)

const numericVersionRegex = `^\d+\.\d+\.\d+$`
const changelogEntryRegex = `## ([^\s]+) - ([0-9][0-9]?\.[0-9][0-9]?\.[0-9][0-9])(?:\s.*)?`

// Changelog is a struct that represents a changelog file
type Changelog struct {
	Entries []ChangelogEntry
}

// ChangelogEntry is a struct that represents an entry of the changelog file (i.e. the changes that were implemented in a release)
type ChangelogEntry struct {
	Version   kaeter.VersionIdentifier
	Content   *ChangelogEntryContent
	Timestamp *time.Time // TODO: agree on the time format (dd.mm.yy for now)
}

// ChangelogEntryContent is a struct that represents the content of a changelog entry
type ChangelogEntryContent struct {
	// unless a fixed structure is defined for all the entries, it probably doesn't make sense to have a more complex structure here
	Content string
}

// UnmarshalVersionString builds a VersionNumber struct from a changelog line
func UnmarshalVersionString(changelogLine string) (kaeter.VersionIdentifier, error) {
	re := regexp.MustCompile(changelogEntryRegex)
	// [line, versionCaptureGroup, dateCapturegroup]
	versionComponents := re.FindStringSubmatch(changelogLine)
	if versionComponents == nil || len(versionComponents) != 3 {
		return nil, fmt.Errorf("unable to parse changelog entry %s", changelogLine)
	}

	versionStr := versionComponents[1]

	isSemVerMatch, _ := regexp.MatchString(numericVersionRegex, versionStr)

	if isSemVerMatch {
		return kaeter.UnmarshalVersionString(versionStr, kaeter.SemVer)
	}
	return kaeter.UnmarshalVersionString(versionStr, kaeter.AnyStringVer)
}

// UnmarshalTimestampString builds a Timestamp struct from a changelog line
func UnmarshalTimestampString(changelogLine string) (*time.Time, error) {
	// multiple date formats are supported using golang layouts: https://golang.org/src/time/format.go
	dateFormats := []string{
		"2.1.06", "2.1.2006", "2.01.06", "2.01.2006",
		"02.1.06", "02.1.2006", "02.01.06", "02.01.2006",
	}

	re := regexp.MustCompile(changelogEntryRegex)
	// [line, versionCaptureGroup, dateCapturegroup]
	versionComponents := re.FindStringSubmatch(changelogLine)
	if versionComponents == nil || len(versionComponents) != 3 {
		return nil, fmt.Errorf("unable to parse changelog entry %s", changelogLine)
	}

	date := versionComponents[2]
	var err error = nil

	for _, dateFormat := range dateFormats {
		timestamp, err := time.Parse(dateFormat, date)
		if err == nil {
			return &timestamp, err
		}
	}

	return nil, err
}

// Parses released changelog entries as a tuple of version and date
// Where the supported line format is:
// - an h2 (##) for each release
// - First a version number SemVer or AnyStringVer
// - dash suround ded by spaces
// - release date (dd.mm.yy)
// - additional information (authors, ...)
func getEntries(str string) ([]ChangelogEntry, error) {
	re := regexp.MustCompile(changelogEntryRegex)
	// Grabs only the maching header lines
	changelogEntryHeaders := re.FindAllString(str, -1)
	// Splits the changelog into blocks which include the release notes
	// The first block will include what comes before the first release (main title and ...)
	changeLogSplitContent := re.Split(str, -1)

	entries := make([]ChangelogEntry, len(changelogEntryHeaders))
	// process each release's content skipping the first chunk which isn't a release (# Changelog...)
	// ...hopefully
	for i, content := range changeLogSplitContent[1:] {
		change := changelogEntryHeaders[i]
		versionNumber, err := UnmarshalVersionString(change)
		if err != nil {
			return nil, fmt.Errorf("Error while parsing the version number: %w", err)
		}
		timestamp, err := UnmarshalTimestampString(change)
		if err != nil {
			return nil, fmt.Errorf("Error while parsing the timestamp: %w", err)
		}

		entry := ChangelogEntry{
			Version:   versionNumber,
			Content:   &ChangelogEntryContent{content},
			Timestamp: timestamp,
		}
		entries[i] = entry
	}

	return entries, nil
}

// UnmarshalChangelog builds a Changelog struct from a string containing a raw changelog file
func UnmarshalChangelog(changelog string) (*Changelog, error) {
	entries, err := getEntries(changelog)
	if err != nil {
		return nil, fmt.Errorf("Error while parsing the changelog file: %s", err.Error())
	}

	return &Changelog{
		Entries: entries,
	}, nil
}

// ReadFromFile reads a Changelog object from the file living at the passed path.
func ReadFromFile(path string) (*Changelog, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return UnmarshalChangelog(string(bytes))
}

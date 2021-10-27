package kaeterpolice

import (
	"fmt"
	"io/ioutil"
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"
	"regexp"
	"strings"
	"time"
)

const numericVersionRegex = `^\d+\.\d+\.\d+$`

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
	versionStr := strings.Trim(strings.Split(changelogLine, "-")[0], "# ")

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
	var err error = nil
	date := strings.TrimSpace(strings.Split(changelogLine, "-")[1])

	for _, dateFormat := range dateFormats {
		timestamp, err := time.Parse(dateFormat, date)
		if err == nil {
			return &timestamp, err
		}
	}

	return nil, err
}

func getEntries(str string) ([]ChangelogEntry, error) {
	// For more complex use cases it may be worth to use a markdown parser
	pattern := "## ([0-9]+\\.[0-9]+\\.[0-9]+) - ([0-9][0-9]?\\.[0-9][0-9]?\\.[0-9][0-9])"
	re := regexp.MustCompile(pattern)
	contents := re.Split(str, -1)
	changes := re.FindAllString(str, -1)

	entries := make([]ChangelogEntry, len(changes))
	for i, content := range contents[1:] {
		change := changes[i]
		versionNumber, err := UnmarshalVersionString(change)
		if err != nil {
			return nil, fmt.Errorf("Error while parsing the version number: %s", err.Error())
		}

		timestamp, err := UnmarshalTimestampString(change)
		if err != nil {
			return nil, fmt.Errorf("Error while parsing the timestamp: %s", err.Error())
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

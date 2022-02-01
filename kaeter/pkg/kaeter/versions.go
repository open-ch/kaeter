package kaeter

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

// template for an empty versions.yaml file
const versionsTemplate = `# Auto-generated file: please edit with care.

# Identifies this module within the fat repo.
id: {{.ID}}
# The underlying tool to which building and releasing is handed off
type: Makefile
# Should this module be versioned with semantic or calendar versioning?
versioning: {{.VersioningScheme}}
# Version identifiers have the following format:
# <version string>: <RFC3339 formatted timestamp>|<commit ID>
versions:
    0.0.0: 1970-01-01T00:00:00Z|INIT
`

const changeLogLink = `
## CHANGELOG

See [CHANGELOG](%s)
`

// Supported Versioning schemes
const (
	SemVer       = "semver"       // Semantic Versioning
	CalVer       = "calver"       // Calendar Versioning, using the YY.MM.MICRO convention
	AnyStringVer = "anystringver" // Anything the user wants that matches [a-zA-Z0-9.+_~@-]+
)

type rawVersions struct {
	ID                  string    `yaml:"id"`
	ModuleType          string    `yaml:"type"`
	VersioningType      string    `yaml:"versioning"`
	RawReleasedVersions yaml.Node `yaml:"versions"`
	Metadata            *Metadata `yaml:"metadata"`
}

// Versions is a fully unmarshalled representation of a versions file
type Versions struct {
	ID               string             `yaml:"id"`
	ModuleType       string             `yaml:"type"`
	VersioningType   string             `yaml:"versioning"`
	ReleasedVersions []*VersionMetadata `yaml:"versions"`
	Metadata         *Metadata          `yaml:"metadata,omitempty"`
	// documentNode contains the complete document representation.
	// It is kep around to safeguard the comments.
	documentNode *yaml.Node
}

// Metadata olds the parsed Annotations from versions.yaml if present.
type Metadata struct {
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// rawVersionHashPair is a simple tuple used while parsing the versions file
type rawKeyValuePair struct {
	Key   string
	Value string
}

// UnmarshalVersions reads a high level Versions object from the passed bytes.
func UnmarshalVersions(bytes []byte) (*Versions, error) {
	var rawNode yaml.Node
	err := yaml.Unmarshal(bytes, &rawNode)
	if err != nil {
		return nil, err
	}

	var rawVers rawVersions
	err = rawNode.Decode(&rawVers)
	if err != nil {
		return nil, err
	}

	return rawVers.toHighLevelVersions(&rawNode)
}

// toHighLevelVersions turns the raw unmarshalled versions object to higher level and user-friendly object.
func (v *rawVersions) toHighLevelVersions(original *yaml.Node) (*Versions, error) {
	releasedVersions, err := v.releasedVersionsMap()
	if err != nil {
		return nil, err
	}
	var parsedReleases []*VersionMetadata
	for _, versionDetails := range releasedVersions {
		unmarshalled, err := UnmarshalVersionMetadata(versionDetails.Key, versionDetails.Value, v.VersioningType)
		if err != nil {
			return nil, err
		}
		parsedReleases = append(parsedReleases, unmarshalled)
	}

	versions := &Versions{
		ID:               v.ID,
		ModuleType:       v.ModuleType,
		VersioningType:   v.VersioningType,
		ReleasedVersions: parsedReleases,
		Metadata:         v.Metadata,
		documentNode:     original,
	}

	return versions, nil
}

func (v *rawVersions) releasedVersionsMap() ([]rawKeyValuePair, error) {

	// Iterating over the 'versions' map and manually parse the content,
	// also retaining the order in which we extracted things.
	// (Just parsing to map[string]string makes us lose the order of the entries in the file.)

	if len(v.RawReleasedVersions.Content)%2 != 0 {
		return nil, fmt.Errorf("raw released versions content length should be even")
	}

	var rawReleasedVersions []rawKeyValuePair

	for i := 0; i < len(v.RawReleasedVersions.Content); i += 2 {
		raw := rawKeyValuePair{
			v.RawReleasedVersions.Content[i].Value,
			v.RawReleasedVersions.Content[i+1].Value,
		}
		rawReleasedVersions = append(rawReleasedVersions, raw)
	}

	return rawReleasedVersions, nil
}

// toRawVersions converts the rich Versions instance back to a simpler object
// ready to be marshaled back to YAML, while taking care to set the underlying data to
// the current content of the v.ReleasedVersions slice.
// the original raw yaml Node, properly mutated and with the original comments, is returned as well.
func (v *Versions) toRawVersions() (*rawVersions, *yaml.Node) {
	// Make a copy of the node to not mutate the one in v
	origNodeCopy := *v.documentNode
	// We need to mutate the last map of the YAML document.
	// We can't just serialize the isntance's map: we would lose the comments on top of it,
	// and the yaml_v3 lib does not offer much help for this, so we need to mutate things by hand.
	mapIdx := len(origNodeCopy.Content[0].Content) - 1
	mapNode := origNodeCopy.Content[0].Content[mapIdx]

	// Here we get a COPY (thus the dereference) from a Node representing a key in a YAML dict,
	// as well a another one representing a value in such a dict.
	aKeyNode := *(mapNode.Content[0])
	aValueNode := *(mapNode.Content[1])

	var newMapContent []*yaml.Node
	for _, versionData := range v.ReleasedVersions {
		// Copy the structs
		keyNode := aKeyNode
		valueNode := aValueNode
		// Set the value to the correct thing
		keyNode.Value = versionData.Number.String()
		valueNode.Value = versionData.marshalReleaseData()
		// Append to the new map content
		newMapContent = append(newMapContent, &keyNode, &valueNode)
	}

	// Update the content of the map node with the up to date map entries
	mapNode.Content = newMapContent

	return &rawVersions{
		ID:                  v.ID,
		ModuleType:          v.ModuleType,
		VersioningType:      v.VersioningType,
		RawReleasedVersions: *mapNode,
	}, &origNodeCopy

}

// nextVersionMetadata computes the VersionMetadata for the next version, based on this object's versioning scheme
// and the passed parameters.
func (v *Versions) nextVersionMetadata(
	refTime *time.Time,
	bumpMajor bool,
	bumpMinor bool,
	userProvidedVersion string,
	commitID string) (*VersionMetadata, error) {
	switch strings.ToLower(v.VersioningType) {
	case AnyStringVer:
		if userProvidedVersion == "" {
			return nil, fmt.Errorf("need to provide a version when versioning scheme is AnyStringVer. Do so with --version")
		}
	case CalVer:
		if userProvidedVersion != "" {
			return nil, fmt.Errorf("cannot manually specify a version with CalVer")
		}
	}
	if bumpMajor && bumpMinor {
		return nil, fmt.Errorf("cannot bump both minor and major at the same time")
	}
	if (bumpMajor || bumpMinor) && userProvidedVersion != "" {
		return nil, fmt.Errorf("--version and --minor/--major are mutually exclusive: automated version bumping is not possible with a user provided version")
	}
	if len(commitID) == 0 {
		return nil, fmt.Errorf("passed commitID is empty")
	}
	if len(v.ReleasedVersions) == 0 {
		return nil, fmt.Errorf("versions instance was not properly initialised: previous release list is empty")
	}
	// .tail(), where are you...
	last := v.ReleasedVersions[len(v.ReleasedVersions)-1]

	var nextNumber VersionIdentifier
	switch versionID := last.Number.(type) {
	case *VersionNumber:
		switch strings.ToLower(v.VersioningType) {
		case SemVer:
			if userProvidedVersion != "" {
				parsedVersionNumber, err := unmarshalNumberTripletVersionString(userProvidedVersion)
				if err != nil {
					return nil, err
				}
				nextNumber = parsedVersionNumber
			} else {
				nextNumber = versionID.nextSemanticVersion(bumpMajor, bumpMinor)
			}
		case CalVer:
			nextNumber = versionID.nextCalendarVersion(refTime)
		default:
			return nil, fmt.Errorf("unknown versioning scheme (acceptable balues are SemVer and CalVer): %s", v.VersioningType)
		}

	case *VersionString:
		match, _ := regexp.MatchString(versionStringRegex, userProvidedVersion)
		if match {
			nextNumber = VersionString{userProvidedVersion}
		} else {
			return nil, fmt.Errorf("user specified version does not match reges %s: %s ", versionStringRegex, userProvidedVersion)
		}
	}

	return &VersionMetadata{
		Number:    nextNumber,
		Timestamp: *refTime,
		CommitID:  commitID,
	}, nil
}

// AddRelease adds a new release to this Versions instance. Note that this does not yet update the YAML
// file from which this object may have been created from.
func (v *Versions) AddRelease(refTime *time.Time, bumpMajor bool, bumpMinor bool, userProvidedVersion string, commitID string) (*VersionMetadata, error) {
	nextMetadata, err := v.nextVersionMetadata(refTime, bumpMajor, bumpMinor, userProvidedVersion, commitID)
	if err != nil {
		return nil, err
	}
	v.ReleasedVersions = append(v.ReleasedVersions, nextMetadata)
	return nextMetadata, nil
}

// Marshal serializes this instance to YAML and returns the corresponding bytes
func (v *Versions) Marshal() ([]byte, error) {
	_, node := v.toRawVersions()
	return yaml.Marshal(node)
}

// SaveToFile saves this instances to a YAML file
func (v *Versions) SaveToFile(path string) error {
	bytes, err := v.Marshal()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, bytes, 0644)
}

// ReadFromFile reads a Versions object from the YAML file living at the passed path.
func ReadFromFile(path string) (*Versions, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return UnmarshalVersions(bytes)
}

type newModule struct {
	ID               string
	VersioningScheme string
}

// Initialise initialises a versions.yaml file at the specified path and a module identified with 'moduleId'.
// path should point to the module's directory.
func Initialise(path string, moduleID string, versioningScheme string, initReadme bool, initChangelog bool) (*Versions, error) {
	sanitizedVersioningScheme, err := validateVersioningScheme(versioningScheme)
	if err != nil {
		return nil, err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("requires a path to an existing directory. Was: %s and resolved to %s", path, absPath)
	}

	versions, err := initVersionsFile(absPath, moduleID, sanitizedVersioningScheme)
	if err != nil {
		return nil, err
	}

	var readmePath string
	if initReadme {
		readmePath, err = initReadmeIfAbsent(absPath)
		if err != nil {
			return nil, err
		}
	}

	if initChangelog {
		err = initChangelogIfAbsent(absPath)
		if err != nil {
			return nil, err
		}
	}

	if initReadme && initChangelog {
		appendChangelogLinkToFile(readmePath, "CHANGELOG.md")
	}

	return versions, nil
}

func initVersionsFile(moduleAbsPath string, moduleID string, sanitizedVersioningScheme string) (*Versions, error) {
	versionsPathYaml := filepath.Join(moduleAbsPath, "versions.yaml")
	if _, err := os.Stat(versionsPathYaml); !os.IsNotExist(err) {
		return nil, fmt.Errorf("cannot init a module with a pre-existing versions.yaml file: %s", versionsPathYaml)
	}
	versionsPathYml := filepath.Join(moduleAbsPath, "versions.yml")
	if _, err := os.Stat(versionsPathYml); !os.IsNotExist(err) {
		return nil, fmt.Errorf("cannot init a module with a pre-existing versions.yml file: %s", versionsPathYml)
	}

	tmpl, err := template.New("versions template").Parse(versionsTemplate)
	if err != nil {
		return nil, err
	}
	file, err := os.Create(versionsPathYaml)
	if err != nil {
		return nil, err
	}
	tmpl.Execute(file, newModule{moduleID, sanitizedVersioningScheme})
	file.Close()
	return ReadFromFile(versionsPathYaml)
}

// initReadme will create an empty README.md file in the moduleAbsPath directory if none exists. Otherwise
func initReadmeIfAbsent(moduleAbsPath string) (string, error) {
	// TODO consider checking for lower case and extension-less variants.
	readmePath := filepath.Join(moduleAbsPath, "README.md")
	_, err := os.Stat(readmePath)
	if !os.IsNotExist(err) {
		// File exists, stop here
		return readmePath, nil
	}

	// Create an empty file
	newReadme, e := os.Create(readmePath)
	if e != nil {
		return "", e
	}

	_, err = newReadme.WriteString("BLESS-THE-MEANING-UPON-ME\n")
	if err != nil {
		return "", err
	}
	newReadme.Close()
	return readmePath, nil
}

func initChangelogIfAbsent(moduleAbsPath string) error {
	// TODO consider checking for lower case and extension-less variants.
	changelogPath := filepath.Join(moduleAbsPath, "CHANGELOG.md")
	_, err := os.Stat(changelogPath)
	if !os.IsNotExist(err) {
		// File exists, stop here
		return nil
	}

	// Create an empty file
	newChangelog, e := os.Create(changelogPath)
	if e != nil {
		return e
	}

	newChangelog.WriteString("# CHANGELOG\n")
	
	newChangelog.Close()

	return nil
}

func appendChangelogLinkToFile(targetPath string, relativeChangelogLocation string) error {
	_, err := os.Stat(targetPath)
	if os.IsNotExist(err) {
		// File does not exist, stop here
		return err
	}
	targetFile, err := os.OpenFile(targetPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	_, err = targetFile.WriteString(fmt.Sprintf(changeLogLink, relativeChangelogLocation))
	if err != nil {
		return err
	}

	targetFile.Close()
	return nil
}

func validateVersioningScheme(versioningScheme string) (string, error) {
	switch strings.ToLower(versioningScheme) {
	case SemVer:
		return "SemVer", nil
	case CalVer:
		return "CalVer", nil
	case AnyStringVer:
		return "AnyStringVer", nil
	}
	return "", fmt.Errorf("unknown versioning scheme: %s", versioningScheme)
}

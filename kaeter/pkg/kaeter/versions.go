package kaeter

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/open-ch/go-libs/fsutils"
	"gopkg.in/yaml.v3"
)

// VersionsFileNameRegex allows identifying kaeter version files
const VersionsFileNameRegex = `versions\.ya?ml`

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
	// It is kept around to safeguard the comments.
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
//
//revive:disable-next-line:cyclomatic High complexity score for older code
//revive:disable-next-line:flag-parameter significant refactoring needed to clean this up
func (v *Versions) nextVersionMetadata(
	refTime *time.Time,
	bump SemVerBump,
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
	if len(commitID) == 0 {
		return nil, fmt.Errorf("given commitID is empty")
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
				nextNumber = versionID.nextSemanticVersion(bump)
			}
		case CalVer:
			nextNumber = versionID.nextCalendarVersion(refTime)
		default:
			return nil, fmt.Errorf("unknown versioning scheme (acceptable balues are SemVer and CalVer): %s", v.VersioningType)
		}

	case *VersionString:
		match, _ := regexp.MatchString(versionStringRegex, userProvidedVersion)
		if !match {
			return nil, fmt.Errorf("user specified version does not match reges %s: %s ", versionStringRegex, userProvidedVersion)
		}
		nextNumber = VersionString{userProvidedVersion}
	}

	return &VersionMetadata{
		Number:    nextNumber,
		Timestamp: *refTime,
		CommitID:  commitID,
	}, nil
}

// AddRelease adds a new release to this Versions instance. Note that this does not yet update the YAML
// file from which this object may have been created from.
// Note if userProvidedVersion is set it will prime over any semantic versionin bump option.
func (v *Versions) AddRelease(refTime *time.Time, bumpType SemVerBump, userProvidedVersion string, commitID string) (*VersionMetadata, error) {
	nextMetadata, err := v.nextVersionMetadata(refTime, bumpType, userProvidedVersion, commitID)
	if err != nil {
		return nil, err
	}

	for _, releasedVersion := range v.ReleasedVersions {
		if releasedVersion.Number.String() == nextMetadata.Number.String() {
			return nil, fmt.Errorf("error version %s already exists in the list of released versions", nextMetadata.Number.String())
		}
		if releasedVersion.CommitID == commitID {
			return nil, fmt.Errorf("error commit ref %s already exists in the list of released versions", commitID)
		}
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
func (v *Versions) SaveToFile(versionsPath string) error {
	bytes, err := v.Marshal()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(versionsPath, bytes, 0644)
}

// ReadFromFile reads a Versions object from the YAML file living at the passed path.
func ReadFromFile(versionsPath string) (*Versions, error) {
	bytes, err := ioutil.ReadFile(versionsPath)
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
//
//revive:disable-next-line:flag-parameter significant refactoring needed to clean this up
func Initialise(modulePath string, moduleID string, versioningScheme string, initReadme bool, initChangelog bool) (*Versions, error) {
	sanitizedVersioningScheme, err := validateVersioningScheme(versioningScheme)
	if err != nil {
		return nil, err
	}
	absPath, err := filepath.Abs(modulePath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("requires a path to an existing directory. Was: %s and resolved to %s", modulePath, absPath)
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
		err = appendChangelogLinkToFile(readmePath, "CHANGELOG.md")
		if err != nil {
			return nil, err
		}
	}

	return versions, nil
}

// FindVersionsYamlFilesInPath recursively looks for versions.yaml
// files starting from the given path.
// Returns on the first error encountered.
func FindVersionsYamlFilesInPath(startingPath string) ([]string, error) {
	allVersionsYAMLFound, err := fsutils.SearchByFileNameRegex(startingPath, VersionsFileNameRegex)
	if err != nil {
		return nil, err
	}
	return allVersionsYAMLFound, nil
}

// GetVersionsFilePath checks if the passed path is a directory, then:
//   - checks if there is a versions.yml or .yaml file, and appends the existing one to the abspath if so
//   - appends 'versions.yaml' to it if there is none.
func GetVersionsFilePath(modulePath string) (string, error) {
	absModulePath, err := filepath.Abs(modulePath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absModulePath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		versionsFilesFound, err := FindVersionsYamlFilesInPath(absModulePath)
		if err != nil {
			return "", err
		}
		if len(versionsFilesFound) == 1 {
			return versionsFilesFound[0], nil
		}

		// Multiple matches? Return the file that is at the specified path, otherwise fail
		if len(versionsFilesFound) > 1 {
			for _, match := range versionsFilesFound {
				if path.Dir(match) == absModulePath {
					return match, nil
				}
			}
			return "", fmt.Errorf("Error multiple versions file in: %s", modulePath)
		}

		return filepath.Join(absModulePath, "versions.yaml"), nil
	}
	return absModulePath, nil
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
	err = tmpl.Execute(file, newModule{moduleID, sanitizedVersioningScheme})
	if err != nil {
		return nil, err
	}
	err = file.Close()
	if err != nil {
		return nil, err
	}
	return ReadFromFile(versionsPathYaml)
}

// initReadme will create an empty README.md file in the moduleAbsPath directory if none exists. Otherwise
func initReadmeIfAbsent(moduleAbsPath string) (string, error) {
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
	err = newReadme.Close()
	if err != nil {
		return "", err
	}
	return readmePath, nil
}

func initChangelogIfAbsent(moduleAbsPath string) error {
	changelogPath := filepath.Join(moduleAbsPath, "CHANGELOG.md")
	_, err := os.Stat(changelogPath)
	if !os.IsNotExist(err) {
		// File exists, stop here
		return nil
	}

	newChangelog, err := os.Create(changelogPath)
	if err != nil {
		return err
	}

	_, err = newChangelog.WriteString("# CHANGELOG\n")
	if err != nil {
		return err
	}

	err = newChangelog.Close()
	if err != nil {
		return err
	}

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

	err = targetFile.Close()
	if err != nil {
		return err
	}
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

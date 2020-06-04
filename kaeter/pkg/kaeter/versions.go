package kaeter

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

// template for an empty versions.yml file
const versionsTemplate = `# Auto-generated file: please edit with care.

# Identifies this module within the fat repo.
id: {{.ID}}
# The underlying tool to which building and releasing is handed off
type: Makefile
# Should this module be versioned with semantic or calendar versioning?
versioning: SemVer
# Version identifiers have the following format:
# <version string>: <RFC3339 formatted timestamp>|<commit ID>
versions:
    0.0.0: 1970-01-01T00:00:00Z|INIT
`

type rawVersions struct {
	ID                  string    `yaml:"id"`
	ModuleType          string    `yaml:"type""`
	VersioningType      string    `yaml:"versioning"`
	RawReleasedVersions yaml.Node `yaml:"versions"`
}

// Versions is a fully unmarshalled representation of a versions file
type Versions struct {
	ID               string             `yaml:"id"`
	ModuleType       string             `yaml:"type""`
	VersioningType   string             `yaml:"versioning"`
	ReleasedVersions []*VersionMetadata `yaml:"versions"`
	// documentNode contains the complete document representation.
	// It is kep around to safeguard the comments.
	documentNode *yaml.Node
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
	versionsMap, err := v.releasedVersionsMap()
	if err != nil {
		return nil, err
	}
	var parsedReleases []*VersionMetadata
	for versNumber, versionDetails := range versionsMap {
		unmarshalled, err := UnmarshalVersionMetadata(versNumber, versionDetails)
		if err != nil {
			return nil, err
		}
		parsedReleases = append(parsedReleases, unmarshalled)
	}
	// Sort the damn thing according to version number:
	// order in the underlying yaml map is not preserved
	// Not
	sort.Slice(parsedReleases, func(i, j int) bool {
		return compareVersionNumbers(parsedReleases[i].Number, parsedReleases[j].Number)
	})

	return &Versions{
		ID:               v.ID,
		ModuleType:       v.ModuleType,
		VersioningType:   v.VersioningType,
		ReleasedVersions: parsedReleases,
		documentNode:     original,
	}, nil
}

// Sort the damn thing according to version number:
// order in the underlying yaml map is not preserved
// Note that we must be careful to order things correctly (ie, naive string sort does not match, ie, breaks on 1.0, 2.0, 10.0)
// returns true if i is smaller than j
func compareVersionNumbers(i, j VersionNumber) bool {
	return i.Major == j.Major && // If both majors are the same, we need to check minors:
		(i.Minor == j.Minor && i.Micro < j.Micro || // If both minors are the same, we compare micros
			i.Minor < j.Minor) || // Otherwise we compare minors
		i.Major < j.Major
}

func (v *rawVersions) releasedVersionsMap() (map[string]string, error) {
	releasedVersionsMap := make(map[string]string)
	err := v.RawReleasedVersions.Decode(&releasedVersionsMap)
	return releasedVersionsMap, err
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
		keyNode.Value = versionData.Number.GetVersionString()
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
	commitID string) (*VersionMetadata, error) {
	if bumpMajor && bumpMinor {
		return nil, fmt.Errorf("cannot bump both minor and major at the same time")
	}
	if len(commitID) == 0 {
		return nil, fmt.Errorf("passed commitID is empty")
	}
	if len(v.ReleasedVersions) == 0 {
		return nil, fmt.Errorf("versions instance was not properly initialised: previous release list is empty")
	}
	// .tail(), where are you...
	last := v.ReleasedVersions[len(v.ReleasedVersions)-1]
	var nextNumber VersionNumber
	switch strings.ToLower(v.VersioningType) {
	case "semver":
		nextNumber = last.Number.nextSemanticVersion(bumpMajor, bumpMinor)
	case "calver":
		nextNumber = last.Number.nextCalendarVersion(refTime)
	default:
		return nil, fmt.Errorf("unknown versioning scheme (acceptable balues are SemVer and CalVer): %s", v.VersioningType)
	}

	return &VersionMetadata{
		Number:    nextNumber,
		Timestamp: *refTime,
		CommitID:  commitID,
	}, nil
}

// AddRelease adds a new release to this Versions instance. Note that this does not yet update the YAML
// file from which this object may have been created from.
func (v *Versions) AddRelease(refTime *time.Time, bumpMajor bool, bumpMinor bool, commitID string) (*VersionMetadata, error) {
	nextMetadata, err := v.nextVersionMetadata(refTime, bumpMajor, bumpMinor, commitID)
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
	ID string
}

// Initialise initialises a versions.yml file at the specified path and a module identified with 'moduleId'.
// path should point to the module's directory.
func Initialise(path string, moduleID string) (*Versions, error) {
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
	versionsPath := filepath.Join(absPath, "versions.yml")
	if _, err := os.Stat(versionsPath); !os.IsNotExist(err) {
		return nil, fmt.Errorf("cannot init a module with a pre-existing versions.yml file: %s", versionsPath)
	}

	tmpl, err := template.New("versions template").Parse(versionsTemplate)
	if err != nil {
		return nil, err
	}
	file, err := os.Create(versionsPath)
	if err != nil {
		return nil, err
	}
	tmpl.Execute(file, newModule{moduleID})
	file.Close()

	return ReadFromFile(versionsPath)
}

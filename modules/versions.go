package modules

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Supported Versioning schemes
const (
	SemVer       = "SemVer"       // Semantic Versioning
	CalVer       = "CalVer"       // Calendar Versioning, using the YY.MM.MICRO convention
	AnyStringVer = "AnyStringVer" // Anything the user wants that matches [a-zA-Z0-9.+_~@-]+
)

type rawVersions struct {
	ID                  string    `yaml:"id"`
	ModuleType          string    `yaml:"type"`
	VersioningType      string    `yaml:"versioning"`
	RawReleasedVersions yaml.Node `yaml:"versions"`
	Metadata            *Metadata `yaml:"metadata"`
	Dependencies        []string  `yaml:"dependencies"`
}

// Versions is a fully unmarshalled representation of a versions file
type Versions struct {
	ID               string             `yaml:"id"`
	ModuleType       string             `yaml:"type"`
	VersioningType   string             `yaml:"versioning"`
	ReleasedVersions []*VersionMetadata `yaml:"versions"`
	Metadata         *Metadata          `yaml:"metadata,omitempty"`
	Dependencies     []string           `yaml:"dependencies,omitempty"`
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

// unmarshalVersions reads a high level Versions object from the passed bytes.
func unmarshalVersions(bytes []byte) (*Versions, error) {
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
	parsedReleases := make([]*VersionMetadata, 0, len(releasedVersions))
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
		Dependencies:     v.Dependencies,
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

	newMapContent := make([]*yaml.Node, 0, len(v.ReleasedVersions))
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
func (v *Versions) nextVersionMetadata(refTime *time.Time, bump SemVerBump, userProvidedVersion, commitID string) (*VersionMetadata, error) {
	err := v.versionBumpSupported(userProvidedVersion, commitID)
	if err != nil {
		return nil, err
	}

	last := v.ReleasedVersions[len(v.ReleasedVersions)-1]
	var nextNumber VersionIdentifier

	switch versionID := last.Number.(type) {
	case *VersionNumber:
		switch v.VersioningType {
		case SemVer:
			if userProvidedVersion != "" {
				parsedVersionNumber, err := UnmarshalVersionString(userProvidedVersion, SemVer)
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
			return nil, fmt.Errorf("unknown versioning scheme (acceptable values are SemVer, CalVer & AnyStringVer): %s", v.VersioningType)
		}
	case *VersionString:
		match, _ := regexp.MatchString(versionStringRegex, userProvidedVersion) //nolint:errcheck
		if !match {
			return nil, fmt.Errorf("user specified version does not match regex %s: %s", versionStringRegex, userProvidedVersion)
		}
		nextNumber = VersionString{userProvidedVersion}
	}

	return &VersionMetadata{
		Number:    nextNumber,
		Timestamp: *refTime,
		CommitID:  commitID,
	}, nil
}

func (v *Versions) versionBumpSupported(userProvidedVersion, commitID string) error {
	switch strings.ToLower(v.VersioningType) {
	case AnyStringVer:
		if userProvidedVersion == "" {
			return fmt.Errorf("need to provide a version when versioning scheme is AnyStringVer. Do so with --version")
		}
	case CalVer:
		if userProvidedVersion != "" {
			return fmt.Errorf("cannot manually specify a version with CalVer")
		}
	}
	if commitID == "" {
		return fmt.Errorf("given commitID is empty")
	}
	if len(v.ReleasedVersions) == 0 {
		return fmt.Errorf("versions instance was not properly initialized: previous release list is empty")
	}
	return nil
}

// AddRelease adds a new release to this Versions instance. Note that this does not yet update the YAML
// file from which this object may have been created from.
// Note if userProvidedVersion is set it will prime over any semantic versionin bump option.
func (v *Versions) AddRelease(refTime *time.Time, bumpType SemVerBump, userProvidedVersion, commitID string) (*VersionMetadata, error) {
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
	return os.WriteFile(versionsPath, bytes, 0600)
}

// ReadFromFile reads a Versions object from the YAML file living at the passed path.
func ReadFromFile(versionsPath string) (*Versions, error) {
	bytes, err := os.ReadFile(versionsPath)
	if err != nil {
		return nil, err
	}
	return unmarshalVersions(bytes)
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
	if !info.IsDir() {
		return absModulePath, nil
	}

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
		return "", fmt.Errorf("error multiple versions file in: %s", modulePath)
	}

	return filepath.Join(absModulePath, "versions.yaml"), nil
}

package kaeter

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

const commitMessageTemplate = `[release] {{.FirstModuleID}} version {{.FirstModuleVersion}}{{if .OtherModulesCount}} (+{{.OtherModulesCount}} other modules){{end}}

Release message generated by #kaeter.
Please do not edit the part below until the end of the raw YAML Segment.

Release Plan:
` + "```lang=yaml" + `

{{.YamlReleasePlan}}
` + "```\n"

// Matching backticks in Go regexes is a ton of fun...
// the first (?s) enables multi-line matching for the dot (.) character.
var releasePlanRegex = regexp.MustCompile(
	`(?s).*Release Plan:(?:\n|\r\n?){1,2}` + "```" + `(?:lang=yaml)?(?:\n|\r\n?){1,2}(.*)` + "```")

type rawReleasePlan struct {
	Releases []string `yaml:"releases"`
}

// ReleasePlan references one or more modules to be released.
type ReleasePlan struct {
	Releases []ReleaseTarget
}

// ReleaseTarget represents a single module to be released. It is identified by its module id and
// contains the version to be released
type ReleaseTarget struct {
	ModuleID string
	Version  string
}

// SingleReleasePlan is a convenience function to create a release plan for a single module (the most common use case)
func SingleReleasePlan(moduleID string, moduleVersion string) *ReleasePlan {
	return &ReleasePlan{Releases: []ReleaseTarget{{moduleID, moduleVersion}}}
}

// Marshal returns a simple string representation of this object in the form <module_id>:<version>.
// Note that it is safe to have a colon (:) in the module id itself, as long as #ReleasePlanFromYaml() is used
// to read this back.
func (rt *ReleaseTarget) Marshal() string {
	return rt.ModuleID + ":" + rt.Version
}

// ReleasePlanFromCommitMessage expects to receive a complete commit message containing a YAML release plan
// formatted in markdown within some triple back ticks (```) like so:
//
// [release] $moduleId version $version
//
// Release message generated by #kaeter.
// Please do not edit the part below until the end of the raw YAML Segment.
//
// Release Plan:
// ```lang=yaml
//
// <RELEASE_PLAN_HERE>
// ```
// It will read the release plan and return an unmarshaled object from it.
func ReleasePlanFromCommitMessage(commitMsg string) (*ReleasePlan, error) {
	groups := releasePlanRegex.FindStringSubmatch(commitMsg)
	if len(groups) != 2 {
		return nil, fmt.Errorf("could not extract release plan from commit message")
	}
	return ReleasePlanFromYaml(groups[1])
}

// ReleasePlanFromYaml reads a release plan from a clean YAML string, as it was created by #ToYamlString()
func ReleasePlanFromYaml(yamlStr string) (*ReleasePlan, error) {
	var rawReleases rawReleasePlan
	err := yaml.Unmarshal([]byte(yamlStr), &rawReleases)
	if err != nil {
		return nil, err
	}
	if len(rawReleases.Releases) == 0 {
		return nil, fmt.Errorf("did not find any releases in the passed yaml string")
	}

	var releases []ReleaseTarget
	for _, rawData := range rawReleases.Releases {
		splitStr := strings.Split(rawData, ":")
		if len(splitStr) < 2 {
			return nil, fmt.Errorf("invalid release targe: %s", rawData)
		}
		versionIdx := len(splitStr) - 1
		releases = append(releases,
			ReleaseTarget{
				ModuleID: strings.Join(splitStr[0:versionIdx], ":"),
				Version:  splitStr[versionIdx],
			},
		)
	}

	return &ReleasePlan{releases}, nil
}

// ToYamlString writes the specifics of this release plan to YAML. The returned YAML
// will only contain the release targets.
func (rp *ReleasePlan) ToYamlString() (string, error) {
	var targetStrings []string
	for _, target := range rp.Releases {
		targetStrings = append(targetStrings, target.Marshal())
	}

	bytes, err := yaml.Marshal(rawReleasePlan{targetStrings})
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// formattingStruct is used to replace the template variables in commitMessageTemplate
type formattingStruct struct {
	FirstModuleID      string
	FirstModuleVersion string
	YamlReleasePlan    string
	OtherModulesCount  int
}

// ToCommitMessage serializes this release plan to a complete commit message
// that can be passed as-is to git.
func (rp *ReleasePlan) ToCommitMessage() (string, error) {
	if len(rp.Releases) == 0 {
		return "", fmt.Errorf("cannot write empty release plan to commit message")
	}
	yamlStr, err := rp.ToYamlString()
	if err != nil {
		return "", nil
	}
	tmpl, err := template.New("Commit Message").Parse(commitMessageTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, formattingStruct{
		rp.Releases[0].ModuleID,
		rp.Releases[0].Version,
		yamlStr,
		len(rp.Releases) - 1,
	})
	if err != nil {
		return "", err
	}
	return string(buf.Bytes()), nil
}

// HasReleasePlan returns true if the passed string (expected to be a commit message) seems to contain a release plan.
// This will check that:
//   - there is a [release] tag somewhere in the passed message
//   - we can extract a release plan from the rest of the body.
func HasReleasePlan(commitMsg string) bool {
	if !strings.Contains(commitMsg, "[release]") {
		return false
	}
	_, err := ReleasePlanFromCommitMessage(commitMsg)
	return err == nil
}

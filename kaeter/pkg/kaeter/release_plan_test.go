package kaeter

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testReleasePlanYaml = `releases:
  - someId:1.2.3
  - groupId:moduleId:4.5.6
  - stringVerId:moduleId:StringVerLulz
`
const testComplexVersionReleasePlanYaml = `releases:
  - complexVersioning:moduleId:v1.2.3+beta1
`

func getTestCommitMsg(t *testing.T, filename string) string {
	bytes, err := os.ReadFile("test-data/" + filename)
	assert.NoError(t, err)
	return string(bytes)
}

func getTestSingleModuleCommitMsg(t *testing.T) string {
	bytes, err := os.ReadFile("test-data/single-module-test-commit-message.txt")
	assert.NoError(t, err)
	return string(bytes)
}

func getTestMultiTagCommitMsg(t *testing.T) string {
	bytes, err := os.ReadFile("test-data/multitag-test-commit-message.txt")
	assert.NoError(t, err)
	return string(bytes)
}

func getTestSquashedCommitMsg(t *testing.T) string {
	bytes, err := os.ReadFile("test-data/squashed-test-commit-message.txt")
	assert.NoError(t, err)
	return string(bytes)
}

func TestReleasePlanFromCommitMessage(t *testing.T) {
	plan, err := ReleasePlanFromCommitMessage(getTestCommitMsg(t, "test-commit-message.txt"))
	assert.NoError(t, err)
	assert.Equal(
		t,
		&ReleasePlan{
			[]ReleaseTarget{
				{"groupId:module2", "2.4.0"},
				{"nonMavenId", "3.4.0"},
				{"stringVerId:moduleId", "StringVerLulz"},
			},
		},
		plan)
}

func TestReleasePlanFromMultiTagCommitMessage(t *testing.T) {
	commitMsg := getTestMultiTagCommitMsg(t)
	assert.True(t, HasReleasePlan(commitMsg))

	plan, err := ReleasePlanFromCommitMessage(getTestMultiTagCommitMsg(t))
	assert.NoError(t, err)
	assert.Equal(
		t,
		&ReleasePlan{
			[]ReleaseTarget{
				{"groupId:module2", "2.4.0"},
				{"nonMavenId", "3.4.0"},
				{"stringVerId:moduleId", "StringVerLulz"},
			},
		},
		plan)
}

func TestReleasePlanFromSquashedCommitMessage(t *testing.T) {
	commitMsg := getTestSquashedCommitMsg(t)
	assert.True(t, HasReleasePlan(commitMsg))

	plan, err := ReleasePlanFromCommitMessage(getTestMultiTagCommitMsg(t))
	assert.NoError(t, err)
	assert.Equal(
		t,
		&ReleasePlan{
			[]ReleaseTarget{
				{"groupId:module2", "2.4.0"},
				{"nonMavenId", "3.4.0"},
				{"stringVerId:moduleId", "StringVerLulz"},
			},
		},
		plan)
}

func TestToCommitMessage(t *testing.T) {
	var tests = []struct {
		name          string
		rp            *ReleasePlan
		commitMessage string
	}{
		{
			name: "single module plan",
			rp: &ReleasePlan{
				[]ReleaseTarget{
					{"groupId:module2", "2.4.0"},
				},
			},
			commitMessage: getTestSingleModuleCommitMsg(t),
		},
		{
			name: "multiple module plan",
			rp: &ReleasePlan{
				[]ReleaseTarget{
					{"groupId:module2", "2.4.0"},
					{"nonMavenId", "3.4.0"},
					{"stringVerId:moduleId", "StringVerLulz"},
				},
			},
			commitMessage: getTestCommitMsg(t, "test-commit-message.txt"),
		},
		{
			name: "complex version module plan",
			rp: &ReleasePlan{
				[]ReleaseTarget{
					{"complexVersioning:moduleId", "v1.2.3+beta1"},
				},
			},
			commitMessage: getTestCommitMsg(t, "complex-version-test-commit-message.txt"),
		},
	}

	for _, tc := range tests {
		commitMsg, err := tc.rp.ToCommitMessage()

		assert.NoError(t, err, tc.name)
		assert.Equal(t, tc.commitMessage, commitMsg, tc.name)
	}
}

func TestReleasePlanFromYaml(t *testing.T) {
	plan, err := ReleasePlanFromYaml(testReleasePlanYaml)
	assert.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Equal(
		t,
		&ReleasePlan{
			[]ReleaseTarget{
				{"someId", "1.2.3"},
				{"groupId:moduleId", "4.5.6"},
				{"stringVerId:moduleId", "StringVerLulz"},
			},
		},
		plan)
}

func TestToYamlString(t *testing.T) {
	var tests = []struct {
		name string
		rp   *ReleasePlan
		yaml string
	}{
		{
			name: "basic release plan",
			rp: &ReleasePlan{
				[]ReleaseTarget{
					{"someId", "1.2.3"},
					{"groupId:moduleId", "4.5.6"},
					{"stringVerId:moduleId", "StringVerLulz"},
				},
			},
			yaml: testReleasePlanYaml,
		},
		{
			name: "complex version numbers",
			rp: &ReleasePlan{
				[]ReleaseTarget{
					{"complexVersioning:moduleId", "v1.2.3+beta1"},
				},
			},
			yaml: testComplexVersionReleasePlanYaml,
		},
	}

	for _, tc := range tests {
		releasePlanYaml, err := tc.rp.ToYamlString()

		assert.NoError(t, err, tc.name)
		assert.Equal(t, tc.yaml, releasePlanYaml, tc.name)
	}
}

func TestHasReleasePlan(t *testing.T) {
	commitMessage := getTestCommitMsg(t, "test-commit-message.txt")
	assert.True(t, HasReleasePlan(commitMessage), "The test commit message should be recognized")
	assert.False(t, HasReleasePlan(strings.TrimPrefix(commitMessage, "[release]")),
		"Without the leading [release] string, this method should return false.")

	notAReleasePlan := `[release] this is not really a release plan, but starts in the same way.

Bla Bla Bla, this is a nice commit message.

Thank you and good-bye.
`
	assert.False(t, HasReleasePlan(notAReleasePlan))
}

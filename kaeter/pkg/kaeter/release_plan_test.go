package kaeter

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const releasePlan = `releases:
  - someId:1.2.3
  - groupId:moduleId:4.5.6
`

func getTestCommitMsg(t *testing.T) string {
	bytes, err := ioutil.ReadFile("test-data/test-commit-message.txt")
	assert.NoError(t, err)
	return string(bytes)
}

func getTestMultiTagCommitMsg(t *testing.T) string {
	bytes, err := ioutil.ReadFile("test-data/multitag-test-commit-message.txt")
	assert.NoError(t, err)
	return string(bytes)
}

func getTestSquashedCommitMsg(t *testing.T) string {
	bytes, err := ioutil.ReadFile("test-data/squashed-test-commit-message.txt")
	assert.NoError(t, err)
	return string(bytes)
}

func TestReleasePlanFromCommitMessage(t *testing.T) {
	plan, err := ReleasePlanFromCommitMessage(getTestCommitMsg(t))
	assert.NoError(t, err)
	assert.Equal(
		t,
		&ReleasePlan{
			[]ReleaseTarget{
				{"groupId:module2", "2.4.0"},
				{"nonMavenId", "3.4.0"}},
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
				{"nonMavenId", "3.4.0"}},
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
				{"nonMavenId", "3.4.0"}},
		},
		plan)
}

func TestReleasePlan_ToCommitMessage(t *testing.T) {
	expected := getTestCommitMsg(t)
	rp := ReleasePlan{
		[]ReleaseTarget{
			{"groupId:module2", "2.4.0"},
			{"nonMavenId", "3.4.0"}},
	}
	commitMsg, err := rp.ToCommitMessage()
	assert.NoError(t, err)
	assert.Equal(t, expected, commitMsg)
}

func TestReleasePlanFromYaml(t *testing.T) {
	plan, err := ReleasePlanFromYaml(releasePlan)
	assert.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Equal(
		t,
		&ReleasePlan{
			[]ReleaseTarget{
				{"someId", "1.2.3"},
				{"groupId:moduleId", "4.5.6"}},
		},
		plan)
}

func TestReleasePlan_ToYamlString(t *testing.T) {
	rp := ReleasePlan{
		[]ReleaseTarget{
			{"someId", "1.2.3"},
			{"groupId:moduleId", "4.5.6"}},
	}
	yamlStr, err := rp.ToYamlString()
	assert.NoError(t, err)
	assert.Equal(t, releasePlan, yamlStr)
}

func TestHasReleasePlan(t *testing.T) {
	assert.True(t, HasReleasePlan(getTestCommitMsg(t)), "The test commit message should be recognized")
	assert.False(t, HasReleasePlan(strings.TrimPrefix(getTestCommitMsg(t), "[release]")),
		"Without the leading [release] string, this method should return false.")

	notAReleasePlan := `[release] this is not really a release plan, but starts in the same way.

Bla Bla Bla, this is a nice commit message.

Thank you and good-bye.
`
	assert.False(t, HasReleasePlan(notAReleasePlan))
}

package kaeter

import (
	"os"
	"path"
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
	bytes, err := os.ReadFile(path.Join("test-data/" + filename))
	assert.NoError(t, err)
	return string(bytes)
}

func TestHasReleasePlan(t *testing.T) {
	var tests = []struct {
		name          string
		commitMessage string
		expectPlan    bool
	}{
		{
			name:          "The test commit message should be recognized",
			commitMessage: getTestCommitMsg(t, "test-commit-message.txt"),
			expectPlan:    true,
		},
		{
			name:          "Release plan with second code block after it",
			commitMessage: getTestCommitMsg(t, "release-commit-message-multiple-code-blocks.txt"),
			expectPlan:    true,
		},
		{
			name:          "Without the leading [release] string, this method should return false.",
			commitMessage: strings.TrimPrefix(getTestCommitMsg(t, "test-commit-message.txt"), "[release]"),
		},
		{
			name: "Correct header but without plan",
			commitMessage: `[release] this is not really a release plan, but starts in the same way.

Bla Bla Bla, this is a nice commit message.

Thank you and good-bye.
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hasPlan := HasReleasePlan(tc.commitMessage)

			assert.Equal(t, tc.expectPlan, hasPlan)
		})
	}
}

func TestReleasePlanFromCommitMessage(t *testing.T) {
	var tests = []struct {
		name          string
		commitMessage string
		rp            *ReleasePlan
	}{
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
			name: "multi tag release plan",
			rp: &ReleasePlan{
				[]ReleaseTarget{
					{"groupId:module2", "2.4.0"},
					{"nonMavenId", "3.4.0"},
					{"stringVerId:moduleId", "StringVerLulz"},
				},
			},
			commitMessage: getTestCommitMsg(t, "multitag-test-commit-message.txt"),
		},
		{
			name: "plan with squashed messages in commit",
			rp: &ReleasePlan{
				[]ReleaseTarget{
					{"ch.open.xdr:pipeline/waldo-image", "21.7.0"},
				},
			},
			commitMessage: getTestCommitMsg(t, "squashed-test-commit-message.txt"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, HasReleasePlan(tc.commitMessage))

			plan, err := ReleasePlanFromCommitMessage(tc.commitMessage)

			assert.NoError(t, err)
			assert.Equal(t, tc.rp, plan)
		})
	}
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
			commitMessage: getTestCommitMsg(t, "single-module-test-commit-message.txt"),
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
		t.Run(tc.name, func(t *testing.T) {
			commitMsg, err := tc.rp.ToCommitMessage()

			assert.NoError(t, err)
			assert.Equal(t, tc.commitMessage, commitMsg)
		})
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

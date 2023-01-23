package change

import (
	"os"
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"

	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var commitMessageWithRelease = "[release] unittest\nRelease Plan:\n```" +
	`lang=yaml
releases:
- ch.open.kaeter:unit-test:0.1.0
` + "```" // no simple way to use a backtick in a raw string...
var commitMessageWithTags = `[unit][testing][tags] TRIVIAL: Commit about unit testing

It includes details
And multiple lines`

func TestCommitCheck(t *testing.T) {
	testCases := []struct {
		lastCommitMessage   string
		expectedTags        []string
		expectedReleasePlan *kaeter.ReleasePlan
		name                string
	}{
		{
			lastCommitMessage: commitMessageWithRelease,
			expectedTags:      []string{"release"},
			expectedReleasePlan: &kaeter.ReleasePlan{
				Releases: []kaeter.ReleaseTarget{
					kaeter.ReleaseTarget{ModuleID: "ch.open.kaeter:unit-test", Version: "0.1.0"},
				},
			},
			name: "Changeset with a Release",
		},
		{
			lastCommitMessage:   commitMessageWithTags,
			expectedTags:        []string{"unit", "testing", "tags"},
			expectedReleasePlan: &kaeter.ReleasePlan{Releases: []kaeter.ReleaseTarget{}},
			name:                "Changeset with tags only",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoPath := createMockRepo(t)
			defer os.RemoveAll(repoPath)
			t.Logf("Temp folder: %s\n(disable `defer os.RemoveAll(testFolder)` to keep for debugging)\n", repoPath)
			firstCommit := commitFileAndGetHash(t, repoPath, "README.md", "# Test Repo", "initial commit")
			secondCommit := commitFileAndGetHash(t, repoPath, "main.go", "", tc.lastCommitMessage)

			detector := &Detector{
				Logger:         logrus.New(),
				RootPath:       repoPath,
				PreviousCommit: firstCommit,
				CurrentCommit:  secondCommit,
			}
			info := &Information{}

			commitMsg := detector.CommitCheck(info)

			assert.Equal(t, commitMsg.Tags, tc.expectedTags, tc.name)
			assert.Equal(t, commitMsg.ReleasePlan, tc.expectedReleasePlan, tc.name)
		})
	}
}

func TestExtractTags(t *testing.T) {
	falseFlags := []string{
		"[tag[ this is not a valid tag",
		"(tag) this is not a valid tag",
		"{tag} this is not a valid tag",
		"[tag[(tag){tag} this is not a valid tag",
	}

	for _, test := range falseFlags {
		assert.NotRegexp(t, tagRegex, test)
	}

	validFlags := []struct {
		commitMsg     string
		expectedFlags []string
	}{
		{
			commitMsg:     "[tag] this is a valid tag",
			expectedFlags: []string{"tag"},
		},
		{
			commitMsg: "[kaeter][buildkite] BE-000: fetch export the commit message tags in the changeset file for use in pipeline step generation\n\n" +
				"Summary: Get the commit message tags and expose them in the changeset.json file to be used when generating pipeline steps\n\n" +
				"Test Plan: Run detect changes locally and in the pipeline\n\n" +
				"Reviewers: pfi, #gophers!, #blazin!, #beng!\n\n" +
				"Differential Revision: SCRUBBED-URL",
			expectedFlags: []string{"kaeter", "buildkite"},
		},
		// Only the leftmost tags should match so adding tags somewhere else
		// in the commit message should be ignored
		{
			commitMsg: "[kaeter][buildkite] BE-000: fetch export the commit message tags in the changeset file for use in pipeline step generation\n\n" +
				"Summary: Get the commit message tags and expose them in the changeset.json file to be used when generating pipeline steps\n\n" +
				"Test Plan: Run detect changes locally and in the pipeline\n\n" +
				"Reviewers: pfi, #gophers!, #blazin!, #beng!\n\n" +
				"Differential Revision: SCRUBBED-URL" +
				"[tag2][tag3]",
			expectedFlags: []string{"kaeter", "buildkite"},
		},
	}

	for _, test := range validFlags {
		regexCapture := extractTags(test.commitMsg)
		assert.Equal(t, test.expectedFlags, regexCapture)
	}
}

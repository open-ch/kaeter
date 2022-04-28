package change

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTagRegex(t *testing.T) {
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
			expectedFlags: []string{"[tag]", "tag", "", ""},
		},
		{
			commitMsg: "[kaeter][buildkite] BE-000: fetch export the commit message tags in the changeset file for use in pipeline step generation\n\n" +
				"Summary: Get the commit message tags and expose them in the changeset.json file to be used when generating pipeline steps\n\n" +
				"Test Plan: Run detect changes locally and in the pipeline\n\n" +
				"Reviewers: pfi, #gophers!, #blazin!, #beng!\n\n" +
				"Differential Revision: SCRUBBED-URL",
			expectedFlags: []string{"[kaeter][buildkite]", "kaeter", "buildkite", ""},
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
			expectedFlags: []string{"[kaeter][buildkite]", "kaeter", "buildkite", ""},
		},
	}

	for _, test := range validFlags {
		regexCapture := tagRegex.FindStringSubmatch(test.commitMsg)
		assert.Equal(t, test.expectedFlags, regexCapture)
	}
}

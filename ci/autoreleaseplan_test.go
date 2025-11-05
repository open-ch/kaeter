package ci

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/change"
	"github.com/open-ch/kaeter/mocks"
	"github.com/open-ch/kaeter/modules"
)

const (
	examplePRBodySimple             = "Some regular\r\n\r\n- PR\r\n- description"
	exampleReleasePlanSingleRelease = "Autorelease-Plan: ch.open.kaeter:example-module:0.1.0\n"
	exampleReleasePlanCRLF          = "Autorelease-Plan: ch.open.kaeter:example-module:0.1.0\r\n"
)

func TestGetUpdatedPRBody(t *testing.T) {
	var tests = []struct {
		name                string
		changeset           string
		expectedBodyContent string
		expectedError       bool
	}{
		{
			name:          "Rejects changeset with prepare based releases",
			changeset:     "changeset-regrelease.json",
			expectedError: true,
		},
		{
			name:                "Zero autoreleases no change needed to body",
			changeset:           "changeset-0_autorelease.json",
			expectedBodyContent: "Bumps example module\r\n\r\n- now using autorelease\r\n- updated with more examples\r\n",
		},
		{
			name:      "Single autorelease in changeset not in body",
			changeset: "changeset-1_autorelease.json",
			expectedBodyContent: "Bumps example module\r\n\r\n- now using autorelease\r\n- updated with more examples\r\n" + `

Autorelease-Plan: ch.open.kaeter:example-module:0.1.0
`,
		},
		{
			name:      "Autorelease in body with CRLFs",
			changeset: "changeset-1_autorelease_update.json",
			expectedBodyContent: "Bumps example module\r\n\r\n- now using autorelease\r\n- updated with more examples" + `

Autorelease-Plan: ch.open.kaeter:example-module:0.1.0
`,
		},
		{
			name:      "Autorelease in body middle (because a line was manually added below)",
			changeset: "changeset-1_autorelease_update_top.json",
			expectedBodyContent: "- bullet list at top\n- added line below" + `

Autorelease-Plan: ch.open.kaeter:example-module:0.1.0
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder := mocks.CreateTmpFolder(t)
			bodyOutputPath := path.Join(testFolder, "prbody.md")
			arc := &AutoReleaseConfig{
				ChangesetPath:       path.Join("testdata", tc.changeset),
				PullRequestBodyPath: bodyOutputPath,
			}

			err := arc.GetUpdatedPRBody()

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				rawBody, err := os.ReadFile(bodyOutputPath)
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedBodyContent, string(rawBody))
			}
		})
	}
}

func TestGetAutoReleasePlan(t *testing.T) {
	var tests = []struct {
		name         string
		changeset    *change.Information
		expectedPlan string
	}{
		{
			name:         "Empty plan without changes",
			changeset:    &change.Information{},
			expectedPlan: "",
		},
		{
			name: "Empty plan without autoreleases",
			changeset: &change.Information{
				Kaeter: change.KaeterChange{
					Modules: map[string]modules.KaeterModule{
						"ch.open.kaeter:example-module": {
							ModuleID:   "ch.open.kaeter:example-module",
							ModulePath: "kaeter-module-under-test",
							ModuleType: "Makefile",
						},
					},
				},
			},
			expectedPlan: "",
		},
		{
			name: "Plan with 1 autorelease",
			changeset: &change.Information{
				Kaeter: change.KaeterChange{
					Modules: map[string]modules.KaeterModule{
						"ch.open.kaeter:example-module": {
							ModuleID:    "ch.open.kaeter:example-module",
							ModulePath:  "kaeter-module-under-test",
							ModuleType:  "Makefile",
							AutoRelease: "0.1.0",
						},
					},
				},
			},
			expectedPlan: `
Autorelease-Plan: ch.open.kaeter:example-module:0.1.0
`,
		},
		{
			name: "Plan with 2 autoreleases",
			changeset: &change.Information{
				Kaeter: change.KaeterChange{
					Modules: map[string]modules.KaeterModule{
						"ch.open.kaeter:example-module": {
							ModuleID:    "ch.open.kaeter:example-module",
							ModulePath:  "kaeter-module-under-test",
							ModuleType:  "Makefile",
							AutoRelease: "0.1.0",
						},
						"ch.open.kaeter:secondary-module": {
							ModuleID:    "ch.open.kaeter:secondary-module",
							ModulePath:  "kaeter-module-under-test",
							ModuleType:  "Makefile",
							AutoRelease: "4.2.0",
						},
					},
				},
			},
			expectedPlan: `
Autorelease-Plan: ch.open.kaeter:example-module:0.1.0
Autorelease-Plan: ch.open.kaeter:secondary-module:4.2.0
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			releasePlan, err := getAutoReleasePlan(tc.changeset)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedPlan, releasePlan)
		})
	}
}

func TestStripAutoReleasePlan(t *testing.T) {
	var tests = []struct {
		name       string
		inputBody  string
		outputBody string
	}{
		{
			name:       "Body without a plan is unchanged",
			inputBody:  examplePRBodySimple,
			outputBody: examplePRBodySimple,
		},
		{
			name:       "Body with simple release plan",
			inputBody:  exampleReleasePlanSingleRelease + "\n" + examplePRBodySimple,
			outputBody: examplePRBodySimple,
		},
		{
			name:       "Body double plan and too many spaces between",
			inputBody:  examplePRBodySimple + "\n" + exampleReleasePlanCRLF + exampleReleasePlanCRLF,
			outputBody: examplePRBodySimple,
		},
		{
			name:       "Body simple plan and too many spaces between",
			inputBody:  exampleReleasePlanSingleRelease + "\n\n\n\n\n\n" + examplePRBodySimple,
			outputBody: examplePRBodySimple,
		},
		{
			name:       "Body with something before the plan",
			inputBody:  "Something unexpected\n" + exampleReleasePlanSingleRelease + examplePRBodySimple,
			outputBody: "Something unexpected\n" + examplePRBodySimple,
		},
		{
			name:       "Body with CRLF release plan (gh formating)",
			inputBody:  exampleReleasePlanCRLF + "\r\n" + examplePRBodySimple,
			outputBody: examplePRBodySimple,
		},
		{
			name:       "Body with CRLF release plan (gh formating) and text before",
			inputBody:  "Something before\r\n\r\n" + exampleReleasePlanCRLF + "\r\n" + examplePRBodySimple,
			outputBody: "Something before\r\n\r\n\r\n" + examplePRBodySimple,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bodyWithoutPlan := stripAutoReleasePlan(tc.inputBody)

			assert.Equal(t, tc.outputBody, bodyWithoutPlan)
		})
	}
}

func TestInsertPlan(t *testing.T) {
	var tests = []struct {
		name         string
		body         string
		plan         string
		expectedBody string
	}{
		{
			name:         "Body without a plan",
			body:         "PR Body",
			plan:         "- ch.open.unit.test:unit-test-module:0.0.0\n",
			expectedBody: "PR Body\n- ch.open.unit.test:unit-test-module:0.0.0\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			updatedBody := insertPlan(tc.body, tc.plan)

			assert.Equal(t, tc.expectedBody, updatedBody)
		})
	}
}

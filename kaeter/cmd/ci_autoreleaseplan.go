package cmd

import (
	"github.com/spf13/cobra"

	"github.com/open-ch/kaeter/kaeter/ci"
)

func getCIAutoReleasePlanCommand() *cobra.Command {
	var changeset string
	var output string

	cmd := &cobra.Command{
		Use:   "autoreleaseplan",
		Short: "Generates an updated pull request body with an autorelease plan",
		Long: `Reads a changeset.json and generate a new PR body
- Parses the changeset.json from kaeter ci detect-changes
- Generate an autorelease plan based on available releases
- Strip previous plan from PR body
- Output a new body with updated autorelease plan

The autorelease plan will contain a list of the modules for which
an autorelease was detected, this is then used on merge to release
the listed modules.
`,
		Run: func(cmd *cobra.Command, args []string) {
			arc := &ci.AutoReleaseConfig{
				ChangesetPath:       changeset,
				PullRequestBodyPath: output,
				Logger:              logger,
			}

			err := arc.GetUpdatedPRBody()
			if err != nil {
				logger.Fatalf("autoreleaseplan failed: %s\n", err)
			}
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&changeset, "changeset", "./changeset.json", "The path to the file with change information")
	flags.StringVar(&output, "output", "./prbody.md", "The path to update pull request body output file")

	return cmd
}

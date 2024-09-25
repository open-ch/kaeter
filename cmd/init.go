package cmd

import (
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/open-ch/kaeter/log"
	"github.com/open-ch/kaeter/modules"
)

func getInitCommand() *cobra.Command {
	var moduleID string
	var flavor string
	var versioningScheme string
	var noReadme bool
	var noChangelog bool
	var noMakefile bool

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a module's versions.yaml file.",
		Long: `Initialize a new kaeter module using the given path and id.
A kaeter module has 4 key components:
- versions.yaml
- README.md
- A Makefile (Makefile.kaeter or Makefile) with the default targets
- A changelog (different formats supported)

init must create the versions.yaml file, and will fail in case of an existing file.

Basic README.md and a CHANGELOG.md will be created if none are found, unless using flags to skip
their creation. When both are created the CHANGELOG.md will be linked from the README.md to avoid
deadlinks.`, // Note that Long is dynamically updated in customHelp() below
		RunE: func(_ *cobra.Command, _ []string) error {
			modulePaths := viper.GetStringSlice("path")
			if len(modulePaths) != 1 {
				return errors.New("init command only supports exactly one path value")
			}

			moduleConfig := &modules.InitializationConfig{
				InitChangelog:    !noChangelog,
				InitMakefile:     !noMakefile,
				InitReadme:       !noReadme,
				ModuleID:         moduleID,
				ModulePath:       modulePaths[0],
				VersioningScheme: versioningScheme,
				Flavor:           flavor,
			}

			log.Info("Initializing new kaeter module", "moduleID", moduleConfig.ModuleID, "modulePath", moduleConfig.ModulePath)
			_, err := modules.Initialize(moduleConfig)
			return err
		},
	}

	flags := initCmd.Flags()

	flags.StringVar(&moduleID, "id", "",
		"The identification string for this module. Something looking like maven coordinates is preferred.")
	err := initCmd.MarkFlagRequired("id")
	if err != nil {
		log.Warn("Error with required id flag", "err", err)
	}
	flags.StringVar(&versioningScheme, "scheme", "SemVer",
		"Versioning scheme to use: one of SemVer, CalVer or AnyStringVer. Defaults to SemVer.")
	flags.BoolVar(&noReadme, "no-readme", false, "Skip README.md creation even if none exists.")
	flags.BoolVar(&noChangelog, "no-changelog", false, "Skip CHANGELOG.md creation even if none exists. ")
	flags.BoolVar(&noMakefile, "no-makefile", false, "Skip Makefile.kaeter creation even if none exists. ")
	flags.StringVar(&flavor, "template", "default", "Allows selecting a preconfigured template flavor.")

	// The default template relies on a private helper function (trimTrailingWhitespaces) so we do without for simplicity:
	initCmd.SetHelpTemplate("{{with (or .Long .Short)}}{{.}}\n\n{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}")
	initCmd.SetHelpFunc(customHelp)

	return initCmd
}

func customHelp(c *cobra.Command, _ []string) {
	// HelpFunc follows a special path and does not run after PreRun/PersistentPreRun functions
	// so we need to manually load the config if we want to access it:
	err := initializeConfig(c)
	if err != nil {
		log.Error("Unable to load config", "err", err)
		return
	}

	rawTemplate := c.HelpTemplate()
	viperKeys := viper.AllKeys()
	firstFlavorFound := true
	flavorArgsDocbuilder := strings.Builder{}

	for _, key := range viperKeys {
		// The template keys are nested and separated with a dot (i.e. templates.default.versions)
		keyElements := strings.Split(key, ".")

		nonTemplatesKey := len(keyElements) != 3 && keyElements[0] != "templates"
		if nonTemplatesKey {
			continue
		}

		flavor := keyElements[1]
		templateType := keyElements[2]

		nonDefaultVersionsTemplate := flavor != "default" && templateType == "versions"
		if nonDefaultVersionsTemplate {
			if firstFlavorFound {
				_, err = flavorArgsDocbuilder.WriteString("\n\nConfigured options for `--template`:\n")
				if err != nil {
					log.Error("Unable to dynamically exend help text", "err", err)
					return
				}
				firstFlavorFound = false
			}
			_, err = flavorArgsDocbuilder.WriteString(fmt.Sprintf("- %s\n", flavor))
			if err != nil {
				log.Error("Unable to dynamically exend help text", "err", err)
				return
			}
		}
	}

	c.Long = fmt.Sprintf("%s%s", c.Long, flavorArgsDocbuilder.String())

	t := template.New("inithelp")
	template.Must(t.Parse(rawTemplate))
	err = t.Execute(c.OutOrStdout(), c)
	if err != nil {
		log.Error(err.Error())
	}
}

package cmd

import (
	"github.com/jim-barber-he/go/util"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the version of the tool",
	Run: func(_ *cobra.Command, _ []string) {
		util.DisplayVersion("ssm")
	},
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

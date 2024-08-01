/*
Package cmd implements the commands for the `ssm` tool.
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/jim-barber-he/go/util"
	"github.com/spf13/cobra"
)

// Commandline options.
type rootOptions struct {
	profile string
	region  string
}

var rootLong = heredoc.Doc(`
	A tool for manipulating parameters in the AWS SSM Parameter Store.

	The tool is somewhat tailored to the environment at my workplace.

	Each of the 'delete', 'get', 'list', and 'put' commands accepts an environment name as the first argument.
	This is one of 'dev', 'test*', or 'prod*'.
	The command maps these to the 'hetest', 'hetest', or 'heaws' AWS profile respectively.

	The environments also influence where the SSM parameters are looked for if not fully qualified by starting with
	a slash (/).
	Non-qualified parameters will be prefixed with '/helm/minikube/', '/helm/test*/', or '/helm/prod*/'.
	The 'minikube' in the path is a legacy path for the development environments at my work place.
	The '/helm/' prefix for all of them is a strange naming convention where the name of the product using these
	parameters was used for the initial path.
`)

// rootCmd represents the base command when called without any subcommands.
var (
	rootCmd = &cobra.Command{
		Use:   "ssm",
		Short: "Manipulate SSM parameter store entries",
		Long:  rootLong,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			cmd.SilenceUsage = true
		},
	}

	rootOpts rootOptions
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	// Get the terminal width for use with my templates below.
	cols, _, err := util.TerminalSize()
	if err != nil {
		cols = 80
	}
	// Reduce it by once since words that bump to the hard right of the terminal look uncomfortable.
	cols--

	// Determine the default region for when one isn't passed in.
	var defaultRegion string
	switch {
	case os.Getenv("AWS_REGION") != "":
		defaultRegion = os.Getenv("AWS_REGION")
	case os.Getenv("AWS_DEFAULT_REGION") != "":
		defaultRegion = os.Getenv("AWS_DEFAULT_REGION")
	default:
		defaultRegion = "ap-southeast-2"
	}

	// Add my own template function to Cobra to handle printing the long description in a way that it wraps with
	// the terminal width properly.
	cobra.AddTemplateFunc("wrapTextToWidth", util.WrapTextToWidth)

	// Retrieve the help template and replace trimTrailingWhitespaces with my own wrapTextToWidth template function.
	helpTemplate := strings.ReplaceAll(
		rootCmd.HelpTemplate(), "trimTrailingWhitespaces", fmt.Sprintf("wrapTextToWidth %d", cols),
	)
	rootCmd.SetHelpTemplate(helpTemplate)

	// Change the usage template to use the new wrapFlags template function.
	usageTemplate := strings.ReplaceAll(
		rootCmd.UsageTemplate(), "FlagUsages", fmt.Sprintf("FlagUsagesWrapped %d", cols),
	)
	rootCmd.SetUsageTemplate(usageTemplate)

	rootCmd.PersistentFlags().StringVar(&rootOpts.profile, "profile", "", "AWS profile to use")
	rootCmd.PersistentFlags().StringVar(&rootOpts.region, "region", defaultRegion, "AWS region to use")
}

// getSSMPath takes an environment name and a path to a location in the SSM parameter store
// and then returns a potentially modified SSM parameter store path.
// The results of these are based on rules used at my workplace.
func getSSMPath(environment, path string) string {
	// dev parameters at my workplace are under the /helm/minikube/ SSM parameter store path.
	if environment == "dev" {
		environment = "minikube"
	}
	// Absolute SSM paths are returned exactly as passed in.
	// Otherwise SSM paths are formatted to suit my workplace,
	// where they are converted to be lowercase, and have a path prefix added based on the environment.
	if path == "" {
		path = "/helm/" + environment
	} else if !strings.HasPrefix(path, "/") {
		path = fmt.Sprintf("/helm/%s/%s", environment, strings.ToLower(path))
	}

	return path
}

// getAWSProfile takes an environment name and returns an AWS Profile based on what is used at my workplace.
// Note that if --profile was passed, then that will take precedence.
func getAWSProfile(environment string) (string, error) {
	// The --profile command line option wins.
	if rootOpts.profile != "" {
		return rootOpts.profile, nil
	}

	var profile string
	switch {
	case environment == "dev":
		profile = "hetest"
	case strings.HasPrefix(environment, "prod"):
		profile = "heaws"
	case strings.HasPrefix(environment, "test"):
		profile = "hetest"
	default:
		// For anything else set the AWS profile to the name of the environment.
		profile = environment
	}

	return profile, nil
}

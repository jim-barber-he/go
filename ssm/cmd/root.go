package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// var cfgFile string

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "ssm",
	Short: "Manipulate SSM parameter store entries",
	Long: `A tool for manipulating parameters in the AWS SSM Parameter Store.

The tool is somewhat tailored to the environment at my workplace.

Each of the 'delete', 'get', 'list', and 'put' commands accepts an environment name as the first argument.
This is one of 'dev', 'test', or 'prod'.
The command maps these to the 'hedev', 'hetest', or 'heaws' AWS profile respectively.

The environments also influence where the SSM parameters are looked for if not fully qualified by starting with a slash (/).
Non-qualified parameters will be prefixed with '/helm/minikube/', '/helm/test/', or '/helm/prod'.
The 'minikube' in the path is a legacy path for the development environments at my work place.
The '/helm/' prefix for all of them is a strange naming convention where the name of a product was used for the path.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here, will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.go.yaml)")

	// Cobra also supports local flags, which will only run when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
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
		path = fmt.Sprintf("/helm/%s", environment)
	} else if !strings.HasPrefix(path, "/") {
		path = fmt.Sprintf("/helm/%s/%s", environment, strings.ToLower(path))
	}

	return path
}

// getAWSProfile takes an environment name and returns an AWS Profile based on what is used at my workplace.
func getAWSProfile(environment string) (string, error) {
	var profile string
	switch {
	case environment == "dev":
		profile = "hedev"
	case environment == "prod":
		profile = "heaws"
	case environment == "test":
		profile = "hetest"
	default:
		return "", fmt.Errorf("Unknown environment %s", environment)
	}

	return profile, nil
}

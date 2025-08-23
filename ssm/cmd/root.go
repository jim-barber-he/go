/*
Package cmd implements the commands for the `ssm` tool.
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/jim-barber-he/go/aws"
	"github.com/jim-barber-he/go/util"
	"github.com/spf13/cobra"
)

// Constants for environment configuration
const (
	// Environment names
	EnvDev        = "dev"
	EnvProd       = "prod"
	EnvTest       = "test"
	EnvMinikube   = "minikube"
	
	// AWS profiles
	ProfileHeTest = "hetest"
	ProfileHeAWS  = "heaws"
	
	// Default configuration values
	DefaultRegion        = "ap-southeast-2"
	DefaultTerminalWidth = 80
	
	// SSM path configuration
	SSMPathPrefix = "/helm/"
	
	// Environment variables
	EnvVarAWSRegion        = "AWS_REGION"
	EnvVarAWSDefaultRegion = "AWS_DEFAULT_REGION"
)

// EnvironmentConfig holds environment-specific configuration settings.
type EnvironmentConfig struct {
	Name        string
	AWSProfile  string
	SSMBasePath string
}

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
	cols := getTerminalWidth()

	// Determine the default region for when one isn't passed in.
	defaultRegion := getDefaultRegion()

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

// getEnvironmentConfig returns the configuration for a specific environment.
func getEnvironmentConfig(environment string) EnvironmentConfig {
	switch {
	case environment == EnvDev:
		return EnvironmentConfig{
			Name:        EnvDev,
			AWSProfile:  ProfileHeTest,
			SSMBasePath: SSMPathPrefix + EnvMinikube,
		}
	case strings.HasPrefix(environment, EnvProd):
		return EnvironmentConfig{
			Name:        environment,
			AWSProfile:  ProfileHeAWS,
			SSMBasePath: SSMPathPrefix + environment,
		}
	case strings.HasPrefix(environment, EnvTest):
		return EnvironmentConfig{
			Name:        environment,
			AWSProfile:  ProfileHeTest,
			SSMBasePath: SSMPathPrefix + environment,
		}
	default:
		return EnvironmentConfig{
			Name:        environment,
			AWSProfile:  environment,
			SSMBasePath: SSMPathPrefix + environment,
		}
	}
}

// getAWSProfile takes an environment name and returns an AWS Profile based on what is used at my workplace.
// Note that if --profile was passed, then that will take precedence.
func getAWSProfile(environment string) string {
	// The --profile command line option takes precedence.
	if rootOpts.profile != "" {
		return rootOpts.profile
	}

	// Get environment configuration
	config := getEnvironmentConfig(environment)
	return config.AWSProfile
}

// getDefaultRegion determines the default AWS region based on environment variables.
func getDefaultRegion() string {
	switch {
	case os.Getenv(EnvVarAWSRegion) != "":
		return os.Getenv(EnvVarAWSRegion)
	case os.Getenv(EnvVarAWSDefaultRegion) != "":
		return os.Getenv(EnvVarAWSDefaultRegion)
	default:
		return DefaultRegion
	}
}

// getSSMClient returns an SSM client based on the provided environment name.
func getSSMClient(ctx context.Context, environment string) *ssm.Client {
	profile := getAWSProfile(environment)
	cfg := aws.Login(ctx, &aws.LoginSessionDetails{Profile: profile, Region: rootOpts.region}, "ssm")

	return aws.SSMClient(cfg)
}

// getSSMPath takes an environment name and a path to a location in the SSM parameter store
// and then returns a potentially modified SSM parameter store path.
// The results of these are based on rules used at my workplace.
func getSSMPath(environment, path string) string {
	// Return fully qualified paths unmodified.
	if strings.HasPrefix(path, "/") {
		return path
	}

	// Get environment configuration
	config := getEnvironmentConfig(environment)
	
	// Build the path based on the environment configuration
	if path == "" {
		return config.SSMBasePath
	}
	
	return config.SSMBasePath + "/" + strings.ToLower(path)
}

// getTerminalWidth retrieves the terminal width, defaulting to the default width if an error occurs.
// The width is reduced by one since words that bump to the hard right of the terminal look uncomfortable.
func getTerminalWidth() int {
	cols, _, err := util.TerminalSize()
	if err != nil {
		cols = DefaultTerminalWidth
	}
	// Reduce it by one since words that bump to the hard right of the terminal look uncomfortable.
	return cols - 1
}

// validateEnvironment checks that the environment name has valid syntax.
// It uses the same rules as an AWS profile name.
func validateEnvironment(environment string) error {
	// Check that the environment name contains only lowercase letters, numbers, and hyphens.
	if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(environment) {
		return newInvalidEnvError(environment)
	}

	// Check that the environment name doesn't contain more than one consecutive hyphen.
	if strings.Contains(environment, "--") {
		return newInvalidEnvError(environment)
	}

	// Check that the environment name doesn't start or end with a hyphen.
	if strings.HasPrefix(environment, "-") || strings.HasSuffix(environment, "-") {
		return newInvalidEnvError(environment)
	}

	// Check that the environment name isn't a 12 digit number.
	if regexp.MustCompile(`^\d{12}$`).MatchString(environment) {
		return newInvalidEnvError(environment)
	}

	return nil
}

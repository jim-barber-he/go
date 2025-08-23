package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/jim-barber-he/go/aws"
	"github.com/jim-barber-he/go/util"
	"github.com/spf13/cobra"
)

// Commandline options.
type getOptions struct {
	full bool
	json bool
}

var getLong = heredoc.Doc(`
	Retrieve a parameter from the AWS SSM parameter store for a given environment.

	By default it will retrieve just the parameter's value.
	Passing the --full flag will show all sorts of details about the parameter including its value.

	You can also add a :$VERSION_NUMBER suffix to the parameter name to retrieve a specific version of a parameter.
`)

var (
	// getCmd represents the get command.
	getCmd = &cobra.Command{
		Use:   "get [flags] ENVIRONMENT PARAMETER[:VERSION_NUMBER]",
		Short: "Retrieve a parameter from the AWS SSM parameter store",
		Long:  getLong,
		Args:  cobra.ExactArgs(2),
		PreRunE: func(_ *cobra.Command, args []string) error {
			return validateEnvironment(args[0])
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return doGet(cmd.Context(), args)
		},
		SilenceErrors: true,
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return getCompletionHelp(args)
		},
	}

	getOpts getOptions
)

func init() {
	rootCmd.AddCommand(getCmd)

	getCmd.Flags().BoolVarP(&getOpts.full, "full", "f", false, "Show all details for the parameter")
	getCmd.Flags().BoolVar(&getOpts.json, "json", false, "Output the parameter in JSON format")
}

// getCompletionHelp provides shell completion help for the delete command.
func getCompletionHelp(args []string) ([]string, cobra.ShellCompDirective) {
	var completionHelp []string

	switch {
	case len(args) == 0:
		completionHelp = cobra.AppendActiveHelp(completionHelp, "dev, test*, or prod*")
	case len(args) == 1:
		completionHelp = cobra.AppendActiveHelp(
			completionHelp, "The path of the SSM parameter, optionally followed by a colon and version number",
		)
	default:
		completionHelp = cobra.AppendActiveHelp(completionHelp, "No more arguments")
	}

	return completionHelp, cobra.ShellCompDirectiveNoFileComp
}

// doGet fetches a parameter from the SSM parameter store.
// args[0] is the name of the AWS Profile to use when accessing the SSM parameter store.
// args[1] is the path of the SSM parameter to get.
func doGet(ctx context.Context, args []string) error {
	ssmClient := getSSMClient(ctx, args[0])

	param := getSSMPath(args[0], args[1])

	par, err := aws.SSMGet(ctx, ssmClient, param, getOpts.full)
	if err != nil {
		var notFound *types.ParameterNotFound
		if errors.As(err, &notFound) {
			fmt.Printf("Parameter %s is not found.", args[1])

			return nil
		}

		return fmt.Errorf("%w: %w", errGetSSMParameter, err)
	}

	if getOpts.full {
		par.Print(false, getOpts.json)
	} else {
		if getOpts.json {
			jsonData, err := util.MarshalWithFields(par, "value")
			if err != nil {
				return fmt.Errorf("failed to marshal parameter value to JSON: %w", err)
			}

			fmt.Println(string(jsonData))
		} else {
			fmt.Println(par.Value)
		}
	}

	return nil
}

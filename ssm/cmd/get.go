package cmd

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/jim-barber-he/go/aws"
	"github.com/spf13/cobra"
)

// Commandline options.
type getOptions struct {
	full bool
}

var getLong = heredoc.Doc(`
	Retrieve a parameter from the AWS SSM parameter store for a given environment.

	By default it will retrieve just the parameter's value.
	Passing the --full flag will show all sorts of details about the parameter including its value.
`)

var (
	// getCmd represents the get command.
	getCmd = &cobra.Command{
		Use:   "get [flags] ENVIRONMENT PARAMETER",
		Short: "Retrieve a parameter from the AWS SSM parameter store",
		Long:  getLong,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return doGet(cmd.Context(), args)
		},
		SilenceErrors: true,
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			var completionHelp []string
			switch {
			case len(args) == 0:
				completionHelp = cobra.AppendActiveHelp(completionHelp, "dev, test*, or prod*")
			case len(args) == 1:
				completionHelp = cobra.AppendActiveHelp(completionHelp, "The path of the SSM parameter")
			default:
				completionHelp = cobra.AppendActiveHelp(completionHelp, "No more arguments")
			}
			return completionHelp, cobra.ShellCompDirectiveNoFileComp
		},
	}

	getOpts getOptions
)

func init() {
	rootCmd.AddCommand(getCmd)

	getCmd.Flags().BoolVarP(&getOpts.full, "full", "f", false, "Show all details for the parameter")
}

// doGet fetches a parameter from the SSM parameter store.
// args[0] is the name of to AWS Profile to use when accessing the SSM parameter store.
// args[1] is the path of the SSM parameter to get.
func doGet(ctx context.Context, args []string) error {
	profile := getAWSProfile(args[0])
	cfg := aws.Login(ctx, &aws.LoginSessionDetails{Profile: profile, Region: rootOpts.region})
	ssmClient := aws.SSMClient(cfg)

	param := getSSMPath(args[0], args[1])
	p, err := aws.SSMGet(ctx, ssmClient, param)
	// I don't know how to handle errors properly... i.e. I don't know how to test if it was a ParameterNotFound error.
	if err != nil {
		return fmt.Errorf("%s%s", err, param)
	}

	if getOpts.full {
		p.Print()
	} else {
		fmt.Println(p.Value)
	}

	return nil
}

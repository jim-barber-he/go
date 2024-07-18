package cmd

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/jim-barber-he/go/aws"
	"github.com/spf13/cobra"
)

// Commandline options.
type listOptions struct {
	full      bool
	recursive bool
}

var listLong = heredoc.Doc(`
	List variables from the SSM parameter store below the supplied path.

	By default it will only list the parameters directly below the supplied path.

	If the --recursive flag is used then it will also show all parameters in the paths below the specified path.

	If the --full flag is specified, then more details about each parameter will be shown.

	If no PATH is passed at all, then for the 'dev', 'test*', and 'prod*' environments it will look in
	'/helm/minikube/', '/helm/test*/', or '/helm/prod*/' respectively.
`)

var (
	// listCmd represents the list command.
	listCmd = &cobra.Command{
		Use:   "list [flags] ENVIRONMENT [PATH]",
		Short: "List parameters from the SSM parameter store below a supplied path",
		Long:  listLong,
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return doList(cmd.Context(), args)
		},
		SilenceErrors: true,
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			var completionHelp []string
			switch {
			case len(args) == 0:
				completionHelp = cobra.AppendActiveHelp(completionHelp, "dev, test*, or prod*")
			case len(args) == 1:
				completionHelp = cobra.AppendActiveHelp(completionHelp, "The path in the SSM parameter store to list")
			default:
				completionHelp = cobra.AppendActiveHelp(completionHelp, "No more arguments")
			}
			return completionHelp, cobra.ShellCompDirectiveNoFileComp
		},
	}

	listOpts listOptions
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listOpts.full, "full", "f", false, "Show additional details for each parameter")
	listCmd.Flags().BoolVarP(
		&listOpts.recursive, "recursive", "r", false, "Recursively list parameters below the parameter store path",
	)
}

// doList will list the SSM Parameter Store parameters below the specified path.
// args[0] is the name of to AWS Profile to use when accessing the SSM parameter store.
// args[1] is the path of the SSM parameter to delete.
func doList(ctx context.Context, args []string) error {
	profile, err := getAWSProfile(args[0])
	if err != nil {
		return err
	}
	cfg := aws.Login(ctx, profile)
	ssmClient := aws.SSMClient(cfg)

	var path string
	if len(args) > 1 {
		path = getSSMPath(args[0], args[1])
	} else {
		path = getSSMPath(args[0], "")
	}

	params, err := aws.SSMList(ctx, ssmClient, path, listOpts.recursive, listOpts.full)
	if err != nil {
		return err
	}

	// Sort function to sort the parameters by Name when iterating through them.
	slices.SortFunc(params, func(a, b aws.SSMParameter) int {
		return cmp.Compare(a.Name, b.Name)
	})

	numParams := len(params) - 1
	for i, param := range params {
		if listOpts.full {
			param.Print()
		} else {
			fmt.Printf("Name: %s\n", param.Name)
			fmt.Printf("Value: %s\n", param.Value)
			fmt.Printf("Type: %s\n", param.Type)
		}
		if i < numParams {
			fmt.Println()
		}
	}

	return nil
}

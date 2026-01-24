package cmd

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/jim-barber-he/go/aws"
	"github.com/spf13/cobra"
)

// Commandline options.
type deleteOptions struct {
	verbose bool
}

var deleteLong = heredoc.Doc(`
	Delete a parameter from the SSM parameter store.

	There is no confirmation, and once deleted you cannot recover.
`)

var (
	// deleteCmd represents the delete command.
	deleteCmd = &cobra.Command{
		Use:   "delete [flags] ENVIRONMENT PARAMETER",
		Short: "Delete a parameter from the SSM parameter store",
		Long:  deleteLong,
		Args:  cobra.ExactArgs(2),
		PreRunE: func(_ *cobra.Command, args []string) error {
			return validateEnvironment(args[0])
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return doDelete(cmd.Context(), args)
		},
		SilenceErrors: true,
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return deleteCompletionHelp(args)
		},
	}

	deleteOpts deleteOptions
)

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().BoolVarP(&deleteOpts.verbose, "verbose", "v", false, "Enable verbose output")
}

// deleteCompletionHelp provides shell completion help for the delete command.
func deleteCompletionHelp(args []string) ([]string, cobra.ShellCompDirective) {
	var completionHelp []cobra.Completion

	switch {
	case len(args) == 0:
		completionHelp = cobra.AppendActiveHelp(completionHelp, "dev, test*, or prod*")
	case len(args) == 1:
		completionHelp = cobra.AppendActiveHelp(completionHelp, "The path of the SSM parameter")
	default:
		completionHelp = cobra.AppendActiveHelp(completionHelp, "No more arguments")
	}

	return completionHelp, cobra.ShellCompDirectiveNoFileComp
}

// doDelete deletes a parameter from the SSM parameter store.
// args[0] is the name of the AWS Profile to use when accessing the SSM parameter store.
// args[1] is the path of the SSM parameter to delete.
func doDelete(ctx context.Context, args []string) error {
	ssmClient := getSSMClient(ctx, args[0])

	param := getSSMPath(args[0], args[1])

	if deleteOpts.verbose {
		fmt.Printf("Deleting parameter: %s\n", param)
	}

	err := aws.SSMDelete(ctx, ssmClient, param)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

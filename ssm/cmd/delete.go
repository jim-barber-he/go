package cmd

import (
	"context"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/jim-barber-he/go/aws"
	"github.com/spf13/cobra"
)

var deleteLong = heredoc.Doc(`
	Delete a parameter from the SSM parameter store.

	There is no confirmation, and once deleted you cannot recover.
`)

// deleteCmd represents the delete command.
var deleteCmd = &cobra.Command{
	Use:   "delete [flags] ENV PARAM",
	Short: "Delete a parameter from the SSM parameter store",
	Long:  deleteLong,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return doDelete(cmd.Context(), args)
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

func init() {
	rootCmd.AddCommand(deleteCmd)
}

// doDelete deletes a parameter from the SSM parameter store.
// args[0] is the name of to AWS Profile to use when accessing the SSM parameter store.
// args[1] is the path of the SSM parameter to delete.
func doDelete(ctx context.Context, args []string) error {
	profile, err := getAWSProfile(args[0])
	if err != nil {
		return err
	}
	cfg := aws.Login(ctx, &aws.LoginSessionDetails{Profile: profile, Region: rootOpts.region})
	ssmClient := aws.SSMClient(cfg)

	param := getSSMPath(args[0], args[1])
	return aws.SSMDelete(ctx, ssmClient, param)
}

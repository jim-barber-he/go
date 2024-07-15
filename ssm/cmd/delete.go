package cmd

import (
	"context"
	"log"

	"github.com/jim-barber-he/go/aws"
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command.
var deleteCmd = &cobra.Command{
	Use:   "delete [flags] ENV PARAM",
	Short: "Delete a parameter from the SSM parameter store",
	Long: `Delete a parameter from the SSM parameter store.

There is no confirmation, and once deleted you cannot recover.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		doDelete(args)
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var completionHelp []string
		switch {
		case len(args) == 0:
			completionHelp = cobra.AppendActiveHelp(completionHelp, "dev, test, or prod")
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
func doDelete(args []string) {
	log.SetFlags(0)

	profile, err := getAWSProfile(args[0])
	if err != nil {
		log.Fatalln(err)
	}

	ctx := context.Background()
	cfg := aws.Login(ctx, profile)

	ssmClient := aws.SSMClient(cfg)

	param := getSSMPath(args[0], args[1])
	err = aws.SSMDelete(ctx, ssmClient, param)
	if err != nil {
		log.Fatalln(err)
	}
}

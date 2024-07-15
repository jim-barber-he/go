package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/jim-barber-he/go/aws"
	"github.com/spf13/cobra"
)

// Commandline options.
type getOptions struct {
	full bool
}

var (
	// getCmd represents the get command.
	getCmd = &cobra.Command{
		Use:   "get [flags] ENVIRONMENT PARAMETER",
		Short: "Retrieve a parameter from the AWS SSM parameter store",
		Long: `Retrieve a parameter from the AWS SSM parameter store for a given environment.

By default it will retrieve just the parameter's value.
Passing the --full flag will show all sorts of details about the parameter including its value.`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			doGet(args)
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

	getOpts getOptions
)

func init() {
	rootCmd.AddCommand(getCmd)

	getCmd.Flags().BoolVarP(&getOpts.full, "full", "f", false, "Show all details for the parameter")
}

// doGet fetches a parameter from the SSM parameter store.
// args[0] is the name of to AWS Profile to use when accessing the SSM parameter store.
// args[1] is the path of the SSM parameter to delete.
func doGet(args []string) {
	log.SetFlags(0)

	profile, err := getAWSProfile(args[0])
	if err != nil {
		log.Fatalln(err)
	}

	ctx := context.Background()
	cfg := aws.Login(ctx, profile)

	ssmClient := aws.SSMClient(cfg)

	param := getSSMPath(args[0], args[1])
	p, err := aws.SSMGet(ctx, ssmClient, param)
	// I don't know how to handle errors properly... i.e. I don't know how to test if it was a ParameterNotFound error.
	if err != nil {
		log.Fatalf("%s%s\n", err, param)
	}

	if getOpts.full {
		p.Print()
	} else {
		fmt.Println(p.Value)
	}
}

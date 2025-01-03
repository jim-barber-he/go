package cmd

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/jim-barber-he/go/aws"
	"github.com/spf13/cobra"
)

// Commandline options.
type listOptions struct {
	brief       bool
	full        bool
	recursive   bool
	safeDecrypt bool
}

var listLong = heredoc.Doc(`
	List variables from the SSM parameter store below the supplied path.

	By default it will only list the parameters directly below the supplied path.

	If the --recursive flag is used then it will also show all parameters in the paths below the specified path.

	If the --full flag is specified, then more details about each parameter will be shown.

	If no PATH is passed at all, then for the 'dev', 'test*', and 'prod*' environments it will look in
	'/helm/minikube/', '/helm/test*/', or '/helm/prod*/' respectively.

	The --safe-decrypt flag is slower, but can handle if you have SecureStrings in your SSM parameter store that
	can't be decrypted due to their KMS key being inaccessible or deleted.
`)

var (
	// listCmd represents the list command.
	listCmd = &cobra.Command{
		Use:   "list [flags] ENVIRONMENT [PATH]",
		Short: "List parameters from the SSM parameter store below a supplied path",
		Long:  listLong,
		Args:  cobra.RangeArgs(1, 2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			result := validateListOptions(cmd)
			if result != nil {
				return result
			}
			return validateEnvironment(args[0])
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return doList(cmd.Context(), args)
		},
		SilenceErrors: true,
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return listCompletionHelp(args)
		},
	}

	listOpts listOptions
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listOpts.brief, "brief", "b", false, "Show parameter = value output")
	listCmd.Flags().BoolVarP(&listOpts.full, "full", "f", false, "Show additional details for each parameter")
	listCmd.Flags().BoolVarP(
		&listOpts.recursive, "recursive", "r", false, "Recursively list parameters below the parameter store path",
	)
	listCmd.Flags().BoolVarP(&listOpts.safeDecrypt, "safe-decrypt", "s", false, "Slower decrypt that can handle errors")
}

// listCompletionHelp provides shell completion help for the delete command.
func listCompletionHelp(args []string) ([]string, cobra.ShellCompDirective) {
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
}

// validateListOptions validates the list command options.
func validateListOptions(cmd *cobra.Command) error {
	if listOpts.brief && listOpts.full {
		return newBriefAndFullError(cmd.UsageString())
	}
	return nil
}

// doList will list the SSM Parameter Store parameters below the specified path.
// args[0] is the name of to AWS Profile to use when accessing the SSM parameter store.
// args[1] is the path of the SSM parameter to list.
func doList(ctx context.Context, args []string) error {
	profile := getAWSProfile(args[0])
	cfg := aws.Login(ctx, &aws.LoginSessionDetails{Profile: profile, Region: rootOpts.region})
	ssmClient := aws.SSMClient(cfg)

	var path string
	if len(args) > 1 {
		path = getSSMPath(args[0], args[1])
	} else {
		path = getSSMPath(args[0], "")
	}

	params, err := listParameters(ctx, ssmClient, path)
	if err != nil {
		return fmt.Errorf("%w: %w", errListSSMParameters, err)
	}

	displayListParameters(params)

	return nil
}

// displayListParameters displays the list of SSM parameters formatted according to the command line flags.
func displayListParameters(params []aws.SSMParameter) {
	// Sort function to sort the parameters by Name when iterating through them.
	slices.SortFunc(params, func(a, b aws.SSMParameter) int {
		return cmp.Compare(a.Name, b.Name)
	})

	numParams := len(params) - 1
	for i, param := range params {
		switch {
		case listOpts.brief:
			fmt.Printf("%s = %s\n", param.Name, param.Value)
		case listOpts.full:
			param.Print()
		default:
			fmt.Printf("Name: %s\n", param.Name)
			fmt.Printf("Value: %s\n", param.Value)
			fmt.Printf("Type: %s\n", param.Type)
			if param.Error != "" {
				fmt.Printf("Error: %s\n", param.Error)
			}
		}
		if i < numParams && !listOpts.brief {
			fmt.Println()
		}
	}
}

// listParameters fetches the SSM parameters handling how decryption is performed based on the safeDecrypt flag.
func listParameters(ctx context.Context, ssmClient *ssm.Client, path string) ([]aws.SSMParameter, error) {
	if listOpts.safeDecrypt {
		return aws.SSMListSafeDecrypt(ctx, ssmClient, path, listOpts.recursive, listOpts.full)
	}
	return aws.SSMList(ctx, ssmClient, path, listOpts.recursive, listOpts.full)
}

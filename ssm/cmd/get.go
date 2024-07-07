package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jim-barber-he/go/aws"
	"github.com/spf13/cobra"
)

var (
	onlyValue bool

	// getCmd represents the get command.
	getCmd = &cobra.Command{
		Use:   "get ENVIRONMENT PARAMETER",
		Short: "Retrieve a parameter from the AWS SSM parameter store",
		Long: `Retrieve a parameter from the AWS SSM parameter store for a given environment.

By default will retrieve a number of fields regarding the parameter, but can be configured to just return the value.`,
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
)

func init() {
	rootCmd.AddCommand(getCmd)

	// I don't know how to refer to vars set like so...
	// getCmd.Flags().BoolP("value", "v", false, "Only return the parameter's value")

	getCmd.Flags().BoolVarP(&onlyValue, "value", "v", false, "Only return the parameter's value")
}

func doGet(args []string) {
	environment := args[0]
	param := args[1]

	log.SetFlags(0)

	// The profile handling is specific to my place of work.
	var awsProfile string
	switch {
	case environment == "dev":
		awsProfile = "hedev"
		// dev parameters at my workplace are under the /helm/minikube/ SSM parameter store path.
		environment = "minikube"
	case environment == "prod":
		awsProfile = "heaws"
	case environment == "test":
		awsProfile = "hetest"
	}

	ctx := context.Background()
	cfg := aws.Login(ctx, awsProfile)

	// Absolute SSM paths are retrieved exactly as specified.
	// Otherwise SSM paths are assumed to be for my workplace, which need to be lowercase,
	// and have a path prefix added based on the environment.
	if !strings.HasPrefix(param, "/") {
		param = fmt.Sprintf("/helm/%s/%s", environment, strings.ToLower(param))
	}

	p, err := aws.SSMGet(ctx, cfg, param)
	// I don't know how to handle errors properly... i.e. I don't know how to test if it was a ParameterNotFound error.
	if err != nil {
		log.Fatalf("%s%s\n", err, param)
	}

	if onlyValue {
		fmt.Println(p.Value)
	} else {
		p.Print()
	}
}

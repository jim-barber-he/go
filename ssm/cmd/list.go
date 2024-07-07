package cmd

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/jim-barber-he/go/aws"
	"github.com/spf13/cobra"
)

var (
	compact   bool
	recursive bool

	// listCmd represents the list command.
	listCmd = &cobra.Command{
		Use:   "list ENVIRONMENT [PATH]",
		Short: "List variables from the SSM parameter store below the supplied path",
		Long: `List variables from the SSM parameter store below the supplied path.

By default it will only list the parameters directly below the supplied path,
but it can also show all parameters in the paths below.`,
		Run: func(cmd *cobra.Command, args []string) {
			doList(args)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			var completionHelp []string
			switch {
			case len(args) == 0:
				completionHelp = cobra.AppendActiveHelp(completionHelp, "dev, test, or prod")
			case len(args) == 1:
				completionHelp = cobra.AppendActiveHelp(completionHelp, "The path in the SSM parameter store to list")
			default:
				completionHelp = cobra.AppendActiveHelp(completionHelp, "No more arguments")
			}
			return completionHelp, cobra.ShellCompDirectiveNoFileComp
		},
	}
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&compact, "compact", "c", false, "Only show the name, value, and type for each parameter")
	listCmd.Flags().BoolVarP(
		&recursive, "recursive", "r", false, "Recursively list parameters below the SSM parameter store path",
	)
}

func doList(args []string) {
	environment := args[0]
	var path string
	if len(args) > 1 {
		path = args[1]
	} else {
		// The default path is specific to my place of work.
		path = fmt.Sprintf("/helm/%s", environment)
	}

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
	if !strings.HasPrefix(path, "/") {
		path = fmt.Sprintf("/helm/%s/%s", environment, strings.ToLower(path))
	}

	params, err := aws.SSMList(ctx, cfg, path, recursive)
	if err != nil {
		log.Fatalln(err)
	}

	// Sort function to sort the parameters by Name when iterating through them.
	slices.SortFunc(params, func(a, b aws.SSMParameter) int {
		return cmp.Compare(a.Name, b.Name)
	})

	for _, param := range params {
		if compact {
			fmt.Printf("Name: %s\n", param.Name)
			fmt.Printf("Value: %s\n", param.Value)
			fmt.Printf("Type: %s\n", param.Type)
		} else {
			param.Print()
		}
		fmt.Println()
	}
}

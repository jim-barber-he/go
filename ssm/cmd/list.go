package cmd

import (
	"cmp"
	"context"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/jim-barber-he/go/aws"
	"github.com/jim-barber-he/go/util"
	"github.com/spf13/cobra"
)

// Commandline options.
type listOptions struct {
	env         bool
	full        bool
	json        bool
	noValue     bool
	recursive   bool
	safeDecrypt bool
	verbose     bool
}

var listLong = heredoc.Doc(`
	List variables from the SSM parameter store below the supplied path.

	By default it will only list the parameters directly below the supplied path.

	The default output will be one line per parameter of the form:
	'ssm_parameter_name = value'.

	If the --recursive flag is used then it will also show all parameters in the paths below the specified path.

	If the --env flag is specified the output will be formatted as environment variables,
	with the parameter names converted to uppercase and values quoted.
	This option cannot be used with the --full, --json, or --verbose flags.

	If the --full flag is specified, then more details about each parameter will be shown.
	This option cannot be used with the --verbose flag.

	If the --json flag is specified, then the output will be formatted as JSON.

	If the --no-value flag is specified, then the parameter values will not be shown.

	The --safe-decrypt flag is slower, but can handle if you have SecureStrings in your SSM parameter store that
	can't be decrypted due to their KMS key being inaccessible or deleted.

	If the --verbose flag is specified, then each parameter will be listed over multiple lines against the
	'Name', 'Value', and 'Type' fields.

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
		PreRunE: func(_ *cobra.Command, args []string) error {
			err := validateListOptions()
			if err != nil {
				return err
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

	listCmd.Flags().BoolVar(&listOpts.env, "env", false, "Display the output formatted as environment variables")
	listCmd.Flags().BoolVarP(&listOpts.full, "full", "f", false, "Show additional details for each parameter")
	listCmd.Flags().BoolVar(&listOpts.json, "json", false, "Display the output as JSON")
	listCmd.Flags().BoolVarP(&listOpts.noValue, "no-value", "n", false, "Do not show the parameter value")
	listCmd.Flags().BoolVarP(
		&listOpts.recursive, "recursive", "r", false, "Recursively list parameters below the parameter store path",
	)
	listCmd.Flags().BoolVarP(&listOpts.safeDecrypt, "safe-decrypt", "s", false, "Slower decrypt that can handle errors")
	listCmd.Flags().BoolVarP(
		&listOpts.verbose, "verbose", "v", false, "Show Name, Value, and Type fields for each parameter",
	)
}

// listCompletionHelp provides shell completion help for the delete command.
func listCompletionHelp(args []string) ([]string, cobra.ShellCompDirective) {
	var completionHelp []cobra.Completion

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
func validateListOptions() error {
	if listOpts.env && (listOpts.full || listOpts.json || listOpts.verbose) {
		return newEnvUsageError()
	}

	if listOpts.full && listOpts.verbose {
		return newFullAndVerboseError()
	}

	return nil
}

// doList will list the SSM Parameter Store parameters below the specified path.
// args[0] is the name of the AWS Profile to use when accessing the SSM parameter store.
// args[1] is the path of the SSM parameter to list.
func doList(ctx context.Context, args []string) error {
	ssmClient := getSSMClient(ctx, args[0])

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
	if len(params) == 0 {
		return
	}

	// Sort function to sort the parameters by Name when iterating through them.
	slices.SortFunc(params, func(a, b aws.SSMParameter) int {
		return cmp.Compare(a.Name, b.Name)
	})

	numParams := len(params) - 1
	for idx, param := range params {
		switch {
		case listOpts.full:
			param.Print(listOpts.noValue, listOpts.json)
		case listOpts.json:
			displayJSON(param)
		case listOpts.verbose:
			displayVerbose(param)
		default:
			displayDefault(param)
		}

		if idx < numParams && !listOpts.json && (listOpts.full || listOpts.verbose) {
			fmt.Println()
		}
	}
}

// displayDefault is a helper function to display a parameter in a one line format.
func displayDefault(param aws.SSMParameter) {
	varLine := fmt.Sprintf("%s = %s", param.Name, param.Value)
	varName := param.Name

	if listOpts.env {
		varName = strings.ToUpper(path.Base(param.Name))
		varLine = fmt.Sprintf("%s=%q", varName, param.Value)
	}

	if listOpts.noValue {
		fmt.Println(varName)
	} else {
		fmt.Printf("%s\n", varLine)
	}
}

// displayJSON is a helper function to display a parameter in JSON format.
func displayJSON(param aws.SSMParameter) {
	fields := []string{"name"}
	if listOpts.verbose {
		fields = append(fields, "type")
	}

	if !listOpts.noValue {
		fields = append(fields, "value")
	}

	jsonData, err := util.MarshalWithFields(param, fields...)
	if err != nil {
		fmt.Printf("Error: failed to marshal parameter to JSON: %v\n", err)
	}

	fmt.Println(string(jsonData))
}

// displayVerbose is a helper function to display a parameter showing its value and type on separate lines.
func displayVerbose(param aws.SSMParameter) {
	fmt.Printf("Name: %s\n", param.Name)

	if !listOpts.noValue {
		fmt.Printf("Value: %s\n", param.Value)
	}

	fmt.Printf("Type: %s\n", param.Type)

	if param.Error != "" {
		fmt.Printf("Error: %s\n", param.Error)
	}
}

// listParameters fetches the SSM parameters handling how decryption is performed based on the safeDecrypt flag.
func listParameters(ctx context.Context, ssmClient *ssm.Client, path string) ([]aws.SSMParameter, error) {
	if listOpts.safeDecrypt {
		params, err := aws.SSMListSafeDecrypt(ctx, ssmClient, path, listOpts.recursive, listOpts.full)
		if err != nil {
			return nil, fmt.Errorf("%w", err)
		}

		return params, nil
	}

	params, err := aws.SSMList(ctx, ssmClient, path, listOpts.recursive, listOpts.full)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	return params, nil
}

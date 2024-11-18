package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/jim-barber-he/go/aws"
	"github.com/spf13/cobra"
)

// Commandline options.
type putOptions struct {
	file    string
	keyID   string
	secure  bool
	verbose bool
}

var putLong = heredoc.Doc(`
	Store a parameter and its value in the AWS SSM parameter store.

	The value to be stored can be passed directly on the command line or read from a file via the --file flag.

	The value will be encrypted if --secure is passed.
	By default it will use the alias/parameter_store_key KMS key to encrypt the value, but you can supply a key via
	--key-id.

	If the --verbose flag is shown, the value stored will be shown.
`)

var (
	// putCmd represents the put command.
	putCmd = &cobra.Command{
		Use:   "put [flags] ENVIRONMENT PARAMETER VALUE\n  ssm put [flags] ENVIRONMENT PARAMETER --file FILE",
		Short: "Store a parameter and its value in the AWS SSM parameter store",
		Long:  putLong,
		Args:  cobra.RangeArgs(2, 3),
		PreRunE: func(_ *cobra.Command, args []string) error {
			return validateEnvironment(args[0])
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return doPut(cmd.Context(), args)
		},
		SilenceErrors: true,
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return putCompletionHelp(args)
		},
	}

	putOpts putOptions
)

func init() {
	rootCmd.AddCommand(putCmd)

	putCmd.Flags().StringVarP(&putOpts.file, "file", "f", "", "Get the value from the file contents")
	putCmd.Flags().StringVar(
		&putOpts.keyID, "key-id", "alias/parameter_store_key", "The ID of the KMS key to encrypt SecureStrings",
	)
	putCmd.Flags().BoolVar(&putOpts.secure, "secure", false, "Store the value as a SecureString")
	putCmd.Flags().BoolVarP(&putOpts.verbose, "verbose", "v", false, "Show the value set for the parameter")
}

// putCompletionHelp provides shell completion help for the put command.
func putCompletionHelp(args []string) ([]string, cobra.ShellCompDirective) {
	var completionHelp []string
	switch {
	case len(args) == 0:
		completionHelp = cobra.AppendActiveHelp(completionHelp, "dev, test*, or prod*")
	case len(args) == 1:
		completionHelp = cobra.AppendActiveHelp(completionHelp, "The path of the SSM parameter")
	case len(args) == 2:
		if putOpts.file != "" {
			completionHelp = cobra.AppendActiveHelp(completionHelp, "No more arguments")
		} else {
			completionHelp = cobra.AppendActiveHelp(completionHelp, "The parameter value")
		}
	default:
		completionHelp = cobra.AppendActiveHelp(completionHelp, "No more arguments")
	}
	return completionHelp, cobra.ShellCompDirectiveNoFileComp
}

// doPut stores a parameter and its value into the SSM parameter store.
// args[0] is the name of to AWS Profile to use when accessing the SSM parameter store.
// args[1] is the path of the SSM parameter to put.
// args[2] is the value to put, but is only valid to use if --file is not used.
func doPut(ctx context.Context, args []string) error {
	profile := getAWSProfile(args[0])
	cfg := aws.Login(ctx, &aws.LoginSessionDetails{Profile: profile, Region: rootOpts.region})
	ssmClient := aws.SSMClient(cfg)

	param := getSSMPath(args[0], args[1])

	value, err := getPutValue(args)
	if err != nil {
		return err
	}

	ssmParam := createPutSSMParameter(param, value)

	// Return if the parameter is already set to the same value and type.
	if unchanged, err := isPutValueUnchanged(ctx, ssmClient, param, ssmParam); err == nil && unchanged {
		fmt.Println("Value unchanged.")
		return nil
	}

	version, err := aws.SSMPut(ctx, ssmClient, &ssmParam)
	if err != nil {
		return fmt.Errorf("%w: %w", errPutSSMParameter, err)
	}
	if putOpts.verbose {
		fmt.Printf("Setting %s = %s\n", param, value)
	}
	fmt.Printf("Parameter %s updated to version %d\n", param, version)

	return nil
}

// createPutSSMParameter creates an SSMParameter struct based on the provided values.
func createPutSSMParameter(name, value string) aws.SSMParameter {
	ssmParam := aws.SSMParameter{
		Name:  name,
		Value: value,
	}
	if putOpts.secure {
		ssmParam.KeyID = putOpts.keyID
		ssmParam.Type = "SecureString"
	} else {
		ssmParam.Type = "String"
	}
	return ssmParam
}

// getPutValue retrieves the value to put into the SSM parameter store.
func getPutValue(args []string) (string, error) {
	if putOpts.file != "" {
		if len(args) > 2 {
			return "", errValueWithFile
		}
		bytes, err := os.ReadFile(putOpts.file)
		if err != nil {
			return "", fmt.Errorf("%w: %w", errReadFile, err)
		}
		return string(bytes), nil
	}
	if len(args) == 2 {
		return "", errValueRequired
	}
	return args[2], nil
}

// isPutValueUnchanged checks if the parameter is already set to the same value and type.
func isPutValueUnchanged(
	ctx context.Context, ssmClient *ssm.Client, param string, ssmParam aws.SSMParameter,
) (bool, error) {
	p, err := aws.SSMGet(ctx, ssmClient, param)
	if err != nil {
		return false, fmt.Errorf("%w: %w", errGetSSMParameter, err)
	}
	return p.Value == ssmParam.Value && p.Type == ssmParam.Type, nil
}

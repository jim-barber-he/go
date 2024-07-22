package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
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
		Use:   "put [flags] ENV PARAM VALUE\n  ssm put [flags] ENV PARAM --file FILE",
		Short: "Store a parameter and its value in the AWS SSM parameter store",
		Long:  putLong,
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return doPut(cmd.Context(), args)
		},
		SilenceErrors: true,
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
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

// doPut stores a parameter and its value into the SSM parameter store.
// args[0] is the name of to AWS Profile to use when accessing the SSM parameter store.
// args[1] is the path of the SSM parameter to delete.
func doPut(ctx context.Context, args []string) error {
	profile, err := getAWSProfile(args[0])
	if err != nil {
		return err
	}
	cfg := aws.Login(ctx, &aws.LoginSessionDetails{Profile: profile, Region: rootOpts.region})
	ssmClient := aws.SSMClient(cfg)

	param := getSSMPath(args[0], args[1])

	var value string
	if putOpts.file != "" {
		if len(args) > 2 {
			return fmt.Errorf("VALUE should not be provided when --file is used")
		}
		bytes, err := os.ReadFile(putOpts.file)
		if err != nil {
			return err
		}
		value = string(bytes)
	} else {
		if len(args) == 2 {
			return fmt.Errorf("VALUE is required when --file is not used")
		}
		value = args[2]
	}

	ssmParam := aws.SSMParameter{
		Name:  param,
		Value: value,
	}
	if putOpts.secure {
		ssmParam.KeyID = putOpts.keyID
		ssmParam.Type = "SecureString"
	} else {
		ssmParam.Type = "String"
	}

	// Return if the parameter is already set to the same value and type.
	p, err := aws.SSMGet(ctx, ssmClient, param)
	if err == nil && p.Value == value && p.Type == ssmParam.Type {
		fmt.Println("Value unchanged.")
		return nil
	}

	version, err := aws.SSMPut(ctx, ssmClient, &ssmParam)
	if err != nil {
		return err
	}
	if putOpts.verbose {
		fmt.Printf("Setting %s = %s\n", param, value)
	}
	fmt.Printf("Parameter %s updated to version %d\n", param, version)

	return nil
}

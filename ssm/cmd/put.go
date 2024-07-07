package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// putCmd represents the put command.
var putCmd = &cobra.Command{
	Use:   "put",
	Short: "Store a parameter and its value in the AWS SSM parameter store.",
	Long: `Store a parameter and its value in the AWS SSM parameter store.

Also handles storing secure strings encrypted via a KMS key.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("put is yet to be implemented")
	},
}

func init() {
	rootCmd.AddCommand(putCmd)
}

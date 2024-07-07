package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command.
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a parameter from the SSM parameter store.",
	Long:  `Delete a parameter from the SSM parameter store.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("delete is yet to be implmented.")
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

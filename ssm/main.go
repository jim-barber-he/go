/*
A tool for managing parameters in the AWS SSM Parameter store.
*/
package main

import (
	"context"
	"log"

	"github.com/jim-barber-he/go/ssm/cmd"
)

func main() {
	// Set log flags to 0 to disable timestamp and other prefixes.
	log.SetFlags(0)

	ctx := context.Background()
	if err := cmd.Execute(ctx); err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}

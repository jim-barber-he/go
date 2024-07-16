package main

import (
	"context"
	"log"

	"github.com/jim-barber-he/go/ssm/cmd"
)

func main() {
	log.SetFlags(0)

	ctx := context.Background()
	if err := cmd.Execute(ctx); err != nil {
		log.Fatalln(err)
	}
}

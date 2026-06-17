package main

import (
	"os"

	"github.com/authorizerdev/authorizer/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

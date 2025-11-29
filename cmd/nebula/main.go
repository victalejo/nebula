package main

import (
	"os"

	"github.com/victalejo/nebula/cmd/nebula/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

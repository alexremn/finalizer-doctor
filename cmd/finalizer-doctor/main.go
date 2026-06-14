package main

import (
	"os"

	"github.com/alexremn/finalizer-doctor/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}

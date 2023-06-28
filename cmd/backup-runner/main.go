package main

import (
	"os"
)

func main() {
	if err := cmdRoot.Execute(); err != nil {
		os.Exit(1)
	}
}

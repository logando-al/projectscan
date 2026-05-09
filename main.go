package main

import (
	"fmt"
	"os"

	"projectscan/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:], mustGetwd(), os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "projectscan: %v\n", err)
		os.Exit(1)
	}
}

func mustGetwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

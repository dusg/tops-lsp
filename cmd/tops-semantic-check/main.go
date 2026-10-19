package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"tops-lsp/internal/semantic"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("tops-semantic-check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	manifestPath := flags.String("manifest", "testdata/semantic/manifest.json", "semantic fixture manifest path")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	report, err := semantic.RunFixtureManifest(*manifestPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if report.Failed > 0 {
		return 1
	}
	return 0
}

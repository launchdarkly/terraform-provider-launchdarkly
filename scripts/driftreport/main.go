package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

const defaultSpecURL = "https://app.launchdarkly.com/api/v2/openapi.json"

// Exit codes: 0 = no drift, 1 = runtime error, 2 = drift detected.
func main() {
	specSource := flag.String("spec", defaultSpecURL, "OpenAPI spec URL or local file path")
	mappingPath := flag.String("mapping", "scripts/driftreport/mapping.yaml", "path to the curated family mapping file")
	outPath := flag.String("out", "-", "output path for the report ('-' for stdout)")
	format := flag.String("format", "md", "report format: md or json")
	flag.Parse()

	if err := run(*specSource, *mappingPath, *outPath, *format); err != nil {
		if err == errDrift {
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "driftreport: %v\n", err)
		os.Exit(1)
	}
}

var errDrift = fmt.Errorf("drift detected")

func run(specSource, mappingPath, outPath, format string) error {
	mapping, err := loadMapping(mappingPath)
	if err != nil {
		return err
	}

	rawSpec, err := fetchSpec(specSource)
	if err != nil {
		return err
	}
	families, err := specFamilies(rawSpec)
	if err != nil {
		return err
	}

	resources, dataSources := registeredTypes()
	report := buildReport(families, mapping, resources, dataSources, specSource)

	var w io.Writer = os.Stdout
	if outPath != "-" {
		f, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer f.Close()
		w = f
	}

	switch format {
	case "md":
		renderMarkdown(w, report)
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return fmt.Errorf("encoding report: %w", err)
		}
	default:
		return fmt.Errorf("unknown format %q (want md or json)", format)
	}

	if report.HasDrift() {
		return errDrift
	}
	return nil
}

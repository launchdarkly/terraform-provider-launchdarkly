package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func main() {
	legacyConfig := flag.String("config", "", "Deprecated alias for -overlay")
	overlay := flag.String("overlay", "", "Path to the overlay config JSON file")
	catalog := flag.String("catalog", "", "Path to the discovered catalog JSON file")
	catalogOut := flag.String("discover-catalog-out", "", "Path to write the discovered OpenAPI resource catalog JSON")
	discoverOnly := flag.Bool("discover-only", false, "Only discover and optionally write catalog output")
	templateDir := flag.String("template-dir", "", "Path to the templates directory")
	outDir := flag.String("out-dir", ".", "Path to launchdarkly output directory")
	testsOutDir := flag.String("tests-out-dir", "./tests", "Path to launchdarkly/tests output directory")
	flag.Parse()

	overlayPath := *overlay
	if overlayPath == "" {
		overlayPath = *legacyConfig
	}
	if overlayPath == "" {
		fmt.Fprintln(os.Stderr, "-overlay is required (or use legacy -config)")
		os.Exit(1)
	}

	if !*discoverOnly && *templateDir == "" {
		fmt.Fprintln(os.Stderr, "-template-dir is required unless -discover-only is set")
		os.Exit(1)
	}

	opts := runOptions{
		OverlayPath:  overlayPath,
		CatalogPath:  *catalog,
		CatalogOut:   *catalogOut,
		DiscoverOnly: *discoverOnly,
		TemplateDir:  *templateDir,
		OutDir:       *outDir,
		TestsOutDir:  *testsOutDir,
	}

	g := newGenerator()
	if err := g.runWithOptions(context.Background(), opts); err != nil {
		fmt.Fprintf(os.Stderr, "openapi-provider-gen failed: %v\n", err)
		os.Exit(1)
	}
}

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/launchdarkly/terraform-provider-launchdarkly/launchdarkly"
)

// Run "go generate" to generate the docs

// Install tools as needed
//go:generate go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
//go:generate go install github.com/ashanbrown/gofmts/cmd/gofmts

// Format examples
//go:generate terraform fmt -recursive ./examples/
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --rendered-provider-name LaunchDarkly --provider-name launchdarkly

// The version string gets updated at build time using -ldflags
var version string = "development"

func main() {
	debugFlag := flag.Bool("debug", false, "Start provider in debug mode.")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address:         "registry.terraform.io/launchdarkly/launchdarkly",
		Debug:           *debugFlag,
		ProtocolVersion: 5,
	}

	if err := providerserver.Serve(context.Background(), launchdarkly.NewPluginProvider(version), opts); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"terraform-provider-launchdarkly/launchdarkly"

	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: launchdarkly.Provider})
}

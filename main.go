package main

import (
	"github.com/terraform-providers/terraform-provider-launchdarkly/launchdarkly"

	"github.com/hashicorp/terraform-plugin-sdk/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: launchdarkly.Provider})
}

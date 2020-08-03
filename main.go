package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/launchdarkly/terraform-provider-launchdarkly/launchdarkly"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: launchdarkly.Provider})
}

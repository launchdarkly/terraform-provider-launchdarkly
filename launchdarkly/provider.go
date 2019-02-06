package launchdarkly

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

const (
	apiKey = "api_key"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			apiKey: {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("LAUNCHDARKLY_API_KEY", nil),
				Description: "The LaunchDarkly API key",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"launchdarkly_project":     resourceProject(),
			"launchdarkly_environment": resourceEnvironment(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		APIKey: d.Get(apiKey).(string),
	}
	return config.New()
}

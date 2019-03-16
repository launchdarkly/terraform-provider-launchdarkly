package launchdarkly

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

const (
	apiKey                   = "api_key"
	launchDarklyAPIKeyEnvVar = "LAUNCHDARKLY_API_KEY"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			apiKey: {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc(launchDarklyAPIKeyEnvVar, nil),
				Description: "The ld API key",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"launchdarkly_project":      resourceProject(),
			"launchdarkly_environment":  resourceEnvironment(),
			"launchdarkly_feature_flag": resourceFeatureFlag(),
			"launchdarkly_webhook":      resourceWebhook(),
			"launchdarkly_custom_role":  resourceCustomRole(),
			"launchdarkly_segment":      resourceSegment(),
			"launchdarkly_team_member":  resourceTeamMember(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	return newClient(d.Get(apiKey).(string))
}

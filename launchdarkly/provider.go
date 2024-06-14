package launchdarkly

import (
	"context"
	"net/url"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//go:generate codegen -o integration_configs_generated.go

const (
	DEFAULT_LAUNCHDARKLY_HOST = "https://app.launchdarkly.com"
	DEFAULT_HTTP_TIMEOUT_S    = 20
)

// Environment Variables
const (
	LAUNCHDARKLY_ACCESS_TOKEN = "LAUNCHDARKLY_ACCESS_TOKEN"
	LAUNCHDARKLY_API_HOST     = "LAUNCHDARKLY_API_HOST"
	LAUNCHDARKLY_OAUTH_TOKEN  = "LAUNCHDARKLY_OAUTH_TOKEN"
)

// Provider keys
const (
	ACCESS_TOKEN = "access_token"
	OAUTH_TOKEN  = "oauth_token"
	API_HOST     = "api_host"
	HTTP_TIMEOUT = "http_timeout"
)

func providerSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		ACCESS_TOKEN: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The [personal access token](https://docs.launchdarkly.com/home/account-security/api-access-tokens#personal-tokens) or [service token](https://docs.launchdarkly.com/home/account-security/api-access-tokens#service-tokens) used to authenticate with LaunchDarkly. You can also set this with the `LAUNCHDARKLY_ACCESS_TOKEN` environment variable. You must provide either `access_token` or `oauth_token`.",
		},
		OAUTH_TOKEN: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "An OAuth V2 token you use to authenticate with LaunchDarkly. You can also set this with the `LAUNCHDARKLY_OAUTH_TOKEN` environment variable. You must provide either `access_token` or `oauth_token`.",
		},
		API_HOST: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The LaunchDarkly host address. If this argument is not specified, the default host address is `https://app.launchdarkly.com`",
		},
		HTTP_TIMEOUT: {
			Type:        schema.TypeInt,
			Optional:    true,
			Description: "The HTTP timeout (in seconds) when making API calls to LaunchDarkly.",
		},
	}
}

// Provider returns a *schema.Provider.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: providerSchema(),
		ResourcesMap: map[string]*schema.Resource{
			"launchdarkly_team":                      resourceTeam(),
			"launchdarkly_project":                   resourceProject(),
			"launchdarkly_environment":               resourceEnvironment(),
			"launchdarkly_feature_flag":              resourceFeatureFlag(),
			"launchdarkly_webhook":                   resourceWebhook(),
			"launchdarkly_custom_role":               resourceCustomRole(),
			"launchdarkly_segment":                   resourceSegment(),
			"launchdarkly_team_member":               resourceTeamMember(),
			"launchdarkly_feature_flag_environment":  resourceFeatureFlagEnvironment(),
			"launchdarkly_destination":               resourceDestination(),
			"launchdarkly_access_token":              resourceAccessToken(),
			"launchdarkly_flag_trigger":              resourceFlagTrigger(),
			"launchdarkly_audit_log_subscription":    resourceAuditLogSubscription(),
			"launchdarkly_relay_proxy_configuration": resourceRelayProxyConfig(),
			"launchdarkly_metric":                    resourceMetric(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"launchdarkly_team":                      dataSourceTeam(),
			"launchdarkly_team_member":               dataSourceTeamMember(),
			"launchdarkly_team_members":              dataSourceTeamMembers(),
			"launchdarkly_project":                   dataSourceProject(),
			"launchdarkly_environment":               dataSourceEnvironment(),
			"launchdarkly_feature_flag":              dataSourceFeatureFlag(),
			"launchdarkly_feature_flag_environment":  dataSourceFeatureFlagEnvironment(),
			"launchdarkly_webhook":                   dataSourceWebhook(),
			"launchdarkly_segment":                   dataSourceSegment(),
			"launchdarkly_flag_trigger":              dataSourceFlagTrigger(),
			"launchdarkly_audit_log_subscription":    dataSourceAuditLogSubscription(),
			"launchdarkly_relay_proxy_configuration": dataSourceRelayProxyConfig(),
			"launchdarkly_metric":                    dataSourceMetric(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	// check environment variables first
	accessToken := os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN)
	oauthToken := os.Getenv(LAUNCHDARKLY_OAUTH_TOKEN)
	host := os.Getenv(LAUNCHDARKLY_API_HOST)

	if host == "" {
		host = DEFAULT_LAUNCHDARKLY_HOST
	}
	configHost := d.Get(API_HOST).(string)
	if configHost != "" {
		host = configHost
	}

	if strings.HasPrefix(host, "http") {
		u, _ := url.Parse(host)
		host = u.Host
	}

	// Check configuration data, which should take precedence over
	// environment variable data, if found.
	configAccessToken := d.Get(ACCESS_TOKEN).(string)
	if configAccessToken != "" {
		accessToken = configAccessToken
	}
	configOAuthToken := d.Get(OAUTH_TOKEN).(string)
	if configOAuthToken != "" {
		oauthToken = configOAuthToken
	}
	if oauthToken == "" && accessToken == "" {
		return nil, diag.Errorf("either an %q or %q must be specified", ACCESS_TOKEN, OAUTH_TOKEN)
	}

	httpTimeoutSeconds := d.Get(HTTP_TIMEOUT).(int)
	if httpTimeoutSeconds == 0 {
		httpTimeoutSeconds = DEFAULT_HTTP_TIMEOUT_S
	}

	if oauthToken != "" {
		client, err := newClient(oauthToken, host, true, httpTimeoutSeconds)
		if err != nil {
			return client, diag.FromErr(err)
		}
		return client, diags
	}

	client, err := newClient(accessToken, host, false, httpTimeoutSeconds)
	if err != nil {
		return client, diag.FromErr(err)
	}
	return client, diags
}

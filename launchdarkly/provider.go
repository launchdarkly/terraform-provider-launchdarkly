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
			Description: "The HTTP timeout (in seconds) when making API calls to LaunchDarkly. Defaults to 20 seconds.",
		},
	}
}

// Provider returns a *schema.Provider.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: providerSchema(),
		ResourcesMap: map[string]*schema.Resource{
			// launchdarkly_access_token now served by the framework provider; see resource_access_token_framework.go.
			"launchdarkly_ai_config":           resourceAIConfig(),
			"launchdarkly_ai_config_variation": resourceAIConfigVariation(),
			// launchdarkly_ai_tool now served by the framework provider; see resource_ai_tool_framework.go.
			"launchdarkly_audit_log_subscription": resourceAuditLogSubscription(),
			// launchdarkly_custom_role now served by the framework provider; see resource_custom_role_framework.go.
			"launchdarkly_destination":              resourceDestination(),
			"launchdarkly_environment":              resourceEnvironment(),
			"launchdarkly_feature_flag":             resourceFeatureFlag(),
			"launchdarkly_flag_templates":           resourceFlagTemplates(),
			"launchdarkly_feature_flag_environment": resourceFeatureFlagEnvironment(),
			// launchdarkly_flag_trigger now served by the framework provider; see resource_flag_trigger_framework.go.
			"launchdarkly_ip_allowlist_config": resourceIpAllowlistConfig(),
			"launchdarkly_ip_allowlist_entry":  resourceIpAllowlistEntry(),
			"launchdarkly_metric":              resourceMetric(),
			// launchdarkly_model_config now served by the framework provider; see resource_model_config_framework.go.
			"launchdarkly_project": resourceProject(),
			// launchdarkly_relay_proxy_configuration now served by the framework provider; see resource_relay_proxy_configuration_framework.go.
			"launchdarkly_segment": resourceSegment(),
			"launchdarkly_team":    resourceTeam(),
			// launchdarkly_team_member now served by the framework provider; see resource_team_member_framework.go.
			"launchdarkly_view":              resourceView(),
			"launchdarkly_view_filter_links": resourceViewFilterLinks(),
			"launchdarkly_view_links":        resourceViewLinks(),
			// launchdarkly_webhook now served by the framework provider; see resource_webhook_framework.go.
		},
		DataSourcesMap: map[string]*schema.Resource{
			// launchdarkly_ai_config now served by the framework provider; see data_source_ai_config_framework.go.
			// launchdarkly_ai_config_variation now served by the framework provider; see data_source_ai_config_variation_framework.go.
			// launchdarkly_ai_tool now served by the framework provider; see data_source_ai_tool_framework.go.
			// launchdarkly_audit_log_subscription now served by the framework provider; see data_source_audit_log_subscription_framework.go.
			// launchdarkly_environment now served by the framework provider; see data_source_environment_framework.go.
			// launchdarkly_feature_flag now served by the framework provider; see data_source_feature_flag_framework.go.
			// launchdarkly_flag_templates now served by the framework provider; see data_source_flag_templates_framework.go.
			// launchdarkly_feature_flag_environment now served by the framework provider; see data_source_feature_flag_environment_framework.go.
			// launchdarkly_flag_trigger now served by the framework provider; see data_source_flag_trigger_framework.go.
			// launchdarkly_metric now served by the framework provider; see data_source_metric_framework.go.
			// launchdarkly_model_config now served by the framework provider; see data_source_model_config_framework.go.
			// launchdarkly_project now served by the framework provider; see data_source_project_framework.go.
			// launchdarkly_relay_proxy_configuration now served by the framework provider; see data_source_relay_proxy_configuration_framework.go.
			// launchdarkly_segment now served by the framework provider; see data_source_segment_framework.go.
			// launchdarkly_team now served by the framework provider; see data_source_team_framework.go.
			// launchdarkly_team_member now served by the framework provider; see data_source_team_member_framework.go.
			// launchdarkly_team_members now served by the framework provider; see data_source_team_members_framework.go.
			// launchdarkly_view now served by the framework provider; see data_source_view_framework.go.
			// launchdarkly_webhook now served by the framework provider; see data_source_webhook_framework.go.
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
	configHost := optionalStringAttr(d, API_HOST)
	if configHost != "" {
		host = configHost
	}

	if strings.HasPrefix(host, "http") {
		u, _ := url.Parse(host)
		host = u.Host
	}

	// Check configuration data, which should take precedence over
	// environment variable data, if found.
	configAccessToken := optionalStringAttr(d, ACCESS_TOKEN)
	if configAccessToken != "" {
		accessToken = configAccessToken
	}
	configOAuthToken := optionalStringAttr(d, OAUTH_TOKEN)
	if configOAuthToken != "" {
		oauthToken = configOAuthToken
	}
	if oauthToken == "" && accessToken == "" {
		return nil, diag.Errorf("either an %q or %q must be specified", ACCESS_TOKEN, OAUTH_TOKEN)
	}

	httpTimeoutSeconds := optionalIntFromResourceData(d, HTTP_TIMEOUT, 0)
	if httpTimeoutSeconds == 0 {
		httpTimeoutSeconds = DEFAULT_HTTP_TIMEOUT_S
	}

	if oauthToken != "" {
		client, err := newClient(oauthToken, host, true, httpTimeoutSeconds, DEFAULT_MAX_CONCURRENCY)
		if err != nil {
			return client, diag.FromErr(err)
		}
		return client, diags
	}

	client, err := newClient(accessToken, host, false, httpTimeoutSeconds, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return client, diag.FromErr(err)
	}
	return client, diags
}

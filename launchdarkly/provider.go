package launchdarkly

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
	ACCESS_TOKEN             = "access_token"
	OAUTH_TOKEN              = "oauth_token"
	API_HOST                 = "api_host"
	HTTP_TIMEOUT             = "http_timeout"
	MAX_CONCURRENCY          = "max_concurrency"
	ARCHIVE_FLAGS_ON_DESTROY = "archive_flags_on_destroy"
)

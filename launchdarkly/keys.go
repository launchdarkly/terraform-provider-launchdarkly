package launchdarkly

const (
	// keys used in terraform files referencing keys in launchdarkly resource objects.
	// The name of each constant is the same as its value.
	defaultProjectKey    = "default"
	project_key          = "project_key"
	env_key              = "env_key"
	key                  = "key"
	name                 = "name"
	tags                 = "tags"
	environments         = "environments"
	api_key              = "api_key"
	mobile_key           = "mobile_key"
	color                = "color"
	default_ttl          = "default_ttl"
	secure_mode          = "secure_mode"
	default_track_events = "default_track_events"
	description          = "description"
	variations           = "variations"
	temporary            = "temporary"
	include_in_snippet   = "include_in_snippet"
	value                = "value"
	//TODO waiting on https://github.com/launchdarkly/api-client-go/issues/1
	//custom_properties    = "custom_properties"

	variation_type = "variation_type"
	url            = "url"
	secret         = "secret"
	sign           = "sign"
	on             = "on"
	_id            = "_id"
	resources      = "resources"
	actions        = "actions"
	effect         = "effect"
	policy         = "policy"
	excluded       = "excluded"
	included       = "included"
)

package launchdarkly

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

var (
	KINESIS_CONVERSION = map[string]interface{}{
		"region":      "region",
		"role_arn":    "roleArn",
		"stream_name": "streamName",
	}
	MPARTICLE_CONVERSION = map[string]interface{}{
		"api_key":       "apiKey",
		"secret":        "secret",
		"user_identity": "userIdentity",
		"environment":   "environment",
	}
	SEGMENT_CONVERSION = map[string]interface{}{
		"write_key": "writeKey",
	}
	GOOGLE_PUBSUB_CONVERSION = map[string]interface{}{
		"project": "project",
		"topic":   "topic",
	}
	AZURE_EVENT_HUBS_CONVERSION = map[string]interface{}{
		"namespace":   "namespace",
		"name":        "name",
		"policy_name": "policyName",
		"policy_key":  "policyKey",
	}
	CONFIG_CONVERSIONS = map[string]map[string]interface{}{
		"kinesis":          KINESIS_CONVERSION,
		"mparticle":        MPARTICLE_CONVERSION,
		"segment":          SEGMENT_CONVERSION,
		"google-pubsub":    GOOGLE_PUBSUB_CONVERSION,
		"azure-event-hubs": AZURE_EVENT_HUBS_CONVERSION,
	}
)

// destinationConfigFromResourceData transforms the terraform resource destination config into a format that complies with the ld-api-go client.
func destinationConfigFromResourceData(d *schema.ResourceData) (interface{}, error) {
	destinationKind := d.Get(KIND).(string)
	// we are just renaming the keys here because it is more conventional for terraform to use snake case, however the LD API uses camel case
	resourceConfig := d.Get(CONFIG).(map[string]interface{})
	requiredAttributes, ok := CONFIG_CONVERSIONS[destinationKind]
	if !ok {
		return resourceConfig, fmt.Errorf("%q is not one of the supported destination kinds", destinationKind)
	}
	config := make(map[string]interface{}, len(requiredAttributes))
	for k, v := range requiredAttributes {
		val, ok := resourceConfig[k]
		if !ok {
			return resourceConfig, fmt.Errorf("missing required config field %q for destination kind %q", k, destinationKind)
		}
		config[v.(string)] = val
	}
	return config, nil
}

// destinationConfigToResourceData transforms the response from to ld-api-go client into the terraform resource structure specified by the schema above.
func destinationConfigToResourceData(kind string, destinationConfig interface{}) map[string]interface{} {
	coercedCfg := destinationConfig.(map[string]interface{})
	config := make(map[string]interface{}, len(coercedCfg))
	for k, v := range CONFIG_CONVERSIONS[kind] {
		config[k] = coercedCfg[v.(string)]
	}
	return config
}

// preserveObfuscatedConfigAttributes overwrites any obfuscated fields in the rawResourceConfig with fields from the resourceConfig provided by the user
func preserveObfuscatedConfigAttributes(originalResourceConfig map[string]interface{}, rawResourceConfig map[string]interface{}) map[string]interface{} {
	ret := rawResourceConfig

	obfuscatedKeys := []string{"api_key", "secret", "write_key", "policy_key"}
	for _, key := range obfuscatedKeys {
		if _, ok := rawResourceConfig[key]; ok {
			if original, ok := originalResourceConfig[key]; ok {
				ret[key] = original
			}
		}
	}

	return ret
}

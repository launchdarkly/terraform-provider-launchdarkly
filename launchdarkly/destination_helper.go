package launchdarkly

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// destinationConfigFromResourceData transforms the terraform resource destination config into a format that complies with the ld-api-go client.
func destinationConfigFromResourceData(d *schema.ResourceData) (interface{}, error) {
	destinationKind := d.Get(KIND).(string)
	// we are just renaming the keys here because it is more conventional for terraform to use snake case, however the LD API uses camel case
	resourceConfig := d.Get(CONFIG).(map[string]interface{})
	switch destinationKind {
	case "kinesis":
		return map[string]interface{}{
			"region":     resourceConfig["region"],
			"roleArn":    resourceConfig["role_arn"],
			"streamName": resourceConfig["stream_name"],
		}, nil
	case "mparticle":
		return map[string]interface{}{
			"apiKey":       resourceConfig["api_key"],
			"secret":       resourceConfig["secret"],
			"userIdentity": resourceConfig["user_identity"],
			"environment":  resourceConfig["environment"],
		}, nil
	case "segment":
		return map[string]interface{}{
			"writeKey": resourceConfig["write_key"],
		}, nil
	case "google-pubsub":
		return resourceConfig, nil // keys are the same in snake and camel case
	default:
		return resourceConfig, fmt.Errorf("%q is not one of the supported destination kinds", destinationKind)
	}
}

// destinationConfigToResourceData transforms the response from to ld-api-go client into the terraform resource structure specified by the schema above.
func destinationConfigToResourceData(kind string, destinationConfig interface{}) map[string]interface{} {
	coercedCfg := destinationConfig.(map[string]interface{})
	switch kind {
	case "kinesis":
		return map[string]interface{}{
			"region":      coercedCfg["region"],
			"role_arn":    coercedCfg["roleArn"],
			"stream_name": coercedCfg["streamName"],
		}
	case "mparticle":
		return map[string]interface{}{
			"api_key":       coercedCfg["apiKey"],
			"secret":        coercedCfg["secret"],
			"user_identity": coercedCfg["userIdentity"],
			"environment":   coercedCfg["environment"],
		}
	case "segment":
		return map[string]interface{}{
			"write_key": coercedCfg["writeKey"],
		}
	case "google-pubsub":
		return coercedCfg // keys are the same in snake and camel case
	default:
		return nil
	}
}

// preserveObfuscatedConfigAttributes overwrites any obfuscated fields in the rawResourceConfig with fields from the resourceConfig provided by the user
func preserveObfuscatedConfigAttributes(originalResourceConfig map[string]interface{}, rawResourceConfig map[string]interface{}) map[string]interface{} {
	ret := rawResourceConfig

	obfuscatedKeys := []string{"api_key", "secret", "write_key"}
	for _, key := range obfuscatedKeys {
		if _, ok := rawResourceConfig[key]; ok {
			if original, ok := originalResourceConfig[key]; ok {
				ret[key] = original
			}
		}
	}

	return ret
}

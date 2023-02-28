package launchdarkly

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type destinationConversion struct {
	required, optional map[string]interface{}
}

func (d destinationConversion) allParameters() map[string]interface{} {
	a := make(map[string]interface{}, len(d.required)+len(d.optional))
	for k, v := range d.required {
		a[k] = v
	}
	for k, v := range d.optional {
		a[k] = v
	}
	return a
}

var (
	KINESIS_CONVERSION = destinationConversion{required: map[string]interface{}{
		"region":      "region",
		"role_arn":    "roleArn",
		"stream_name": "streamName",
	}, optional: map[string]interface{}{}}
	MPARTICLE_CONVERSION = destinationConversion{required: map[string]interface{}{
		"api_key":     "apiKey",
		"secret":      "secret",
		"environment": "environment",
	}, optional: map[string]interface{}{
		"user_identities":         "userIdentities", // A list of objects represented as json-encoded strings with ldContextKind and mparticleUserIdentity strings
		"anonymous_user_identity": "anonymousUserIdentity",
		"user_identity":           "userIdentity",
	}}
	SEGMENT_CONVERSION = destinationConversion{required: map[string]interface{}{
		"write_key": "writeKey",
	}, optional: map[string]interface{}{
		"anonymous_id_context_kind": "anonymousIDContextKind",
		"user_id_context_kind":      "userIDContextKind",
	}}
	GOOGLE_PUBSUB_CONVERSION = destinationConversion{required: map[string]interface{}{
		"project": "project",
		"topic":   "topic",
	}, optional: map[string]interface{}{}}
	AZURE_EVENT_HUBS_CONVERSION = destinationConversion{required: map[string]interface{}{
		"namespace":   "namespace",
		"name":        "name",
		"policy_name": "policyName",
		"policy_key":  "policyKey",
	}, optional: map[string]interface{}{}}
	CONFIG_CONVERSIONS = map[string]destinationConversion{
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
	attributes, ok := CONFIG_CONVERSIONS[destinationKind]
	if !ok {
		return resourceConfig, fmt.Errorf("%q is not one of the supported destination kinds", destinationKind)
	}
	requiredAttributes := attributes.required
	optionalAttributes := attributes.optional
	config := make(map[string]interface{}, len(requiredAttributes)+len(optionalAttributes))
	for k, v := range requiredAttributes {
		val, ok := resourceConfig[k]
		if !ok {
			return resourceConfig, fmt.Errorf("missing required config field %q for destination kind %q", k, destinationKind)
		}
		config[v.(string)] = val
	}
	for k, v := range optionalAttributes {
		val, ok := resourceConfig[k]
		if !ok {
			continue
		}
		if destinationKind == "mparticle" && k == "user_identities" {
			var identitiesObjectArray []map[string]interface{} // user_identities should be represented as an array of mParticle user objects json encoded as a string
			err := json.Unmarshal([]byte(val.(string)), &identitiesObjectArray)
			if err != nil {
				return resourceConfig, fmt.Errorf("config field %q for destination kind %q is not valid: %s", k, destinationKind, err.Error())
			}
			err = validateMParticleUserIdentities(identitiesObjectArray)
			if err != nil {
				return resourceConfig, fmt.Errorf("badly-formed mParticle user_identities field: %s", err.Error())
			}
			config[v.(string)] = identitiesObjectArray
		} else {
			config[v.(string)] = val
		}
	}
	return config, nil
}

// destinationConfigToResourceData transforms the response from to ld-api-go client into the terraform resource structure specified by the schema above.
func destinationConfigToResourceData(kind string, destinationConfig interface{}) map[string]interface{} {
	coercedCfg := destinationConfig.(map[string]interface{})
	config := make(map[string]interface{}, len(coercedCfg))
	for k, v := range CONFIG_CONVERSIONS[kind].allParameters() {
		if coercedCfg[v.(string)] != nil {
			if kind == "mparticle" && k == "user_identities" {
				identitiesByte, err := json.Marshal(coercedCfg[v.(string)])
				if err != nil {
					return config
				}
				config[k] = string(identitiesByte)
			} else {
				config[k] = coercedCfg[v.(string)]
			}
		}
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

func validateMParticleUserIdentities(identities []map[string]interface{}) error {
	for _, identity := range identities {
		ldContext, ok := identity["ldContextKind"]
		if !ok {
			return fmt.Errorf("missing field ldContextKind")
		}
		if _, ok := ldContext.(string); !ok {
			return fmt.Errorf("ldContextKind must be a string")
		}
		mParticleUserIdentity, ok := identity["mparticleUserIdentity"]
		if !ok {
			return fmt.Errorf("missing field mparticleUserIdentity")
		}
		if _, ok := mParticleUserIdentity.(string); !ok {
			return fmt.Errorf("mparticleUserIdentity must be a string")
		}
		for k := range identity {
			if k != "ldContextKind" && k != "mparticleUserIdentity" {
				return fmt.Errorf("key %s is invalid", k)
			}
		}
	}
	return nil
}

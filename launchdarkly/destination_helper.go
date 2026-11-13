package launchdarkly

import "fmt"

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

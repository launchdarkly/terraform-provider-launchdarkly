package launchdarkly

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v7"
)

func baseFeatureFlagEnvironmentSchema(forDataSource bool) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		FLAG_ID: {
			Type:         schema.TypeString,
			Required:     true,
			Description:  "The global feature flag's unique id in the format `<project_key>/<flag_key>`",
			ForceNew:     true,
			ValidateFunc: validateFlagID,
		},
		ENV_KEY: {
			Type:         schema.TypeString,
			Required:     true,
			Description:  "The LaunchDarkly environment key",
			ForceNew:     true,
			ValidateFunc: validateKey(),
		},
		ON: {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether targeting is enabled",
			Default:     false,
		},
		TARGETS:       targetsSchema(),
		RULES:         rulesSchema(),
		PREREQUISITES: prerequisitesSchema(),
		FALLTHROUGH:   fallthroughSchema(forDataSource),
		TRACK_EVENTS: {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether to send event data back to LaunchDarkly",
			Default:     false,
		},
		OFF_VARIATION: {
			Type:         schema.TypeInt,
			Required:     !forDataSource,
			Optional:     forDataSource,
			Description:  "The index of the variation to serve if targeting is disabled",
			ValidateFunc: validation.IntAtLeast(0),
		},
	}
}

// get FeatureFlagEnvironment uses a query parameter to get the ldapi.FeatureFlag with only a single environment.
func getFeatureFlagEnvironment(client *Client, projectKey, flagKey, environmentKey string) (ldapi.FeatureFlag, *http.Response, error) {
	return client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, flagKey).Env(environmentKey).Execute()
}

func featureFlagEnvironmentRead(d *schema.ResourceData, raw interface{}, isDataSource bool) error {
	client := raw.(*Client)
	flagId := d.Get(FLAG_ID).(string)
	projectKey, flagKey, err := flagIdToKeys(flagId)
	if err != nil {
		return err
	}
	envKey := d.Get(ENV_KEY).(string)

	flag, res, err := getFeatureFlagEnvironment(client, projectKey, flagKey, envKey)
	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[WARN] failed to find flag %q in project %q, removing from state", flagKey, projectKey)
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to get flag %q of project %q: %s", flagKey, projectKey, handleLdapiErr(err))
	}

	environment, ok := flag.Environments[envKey]
	if !ok {
		log.Printf("[WARN] failed to find environment %q for flag %q, removing from state", envKey, flagKey)
		d.SetId("")
		return nil
	}

	if isDataSource {
		d.SetId(projectKey + "/" + envKey + "/" + flagKey)
	}
	_ = d.Set(FLAG_ID, projectKey+"/"+flag.Key)

	// Computed values are set even if they do not exist on the config
	_ = d.Set(ON, environment.On)
	_ = d.Set(TRACK_EVENTS, environment.TrackEvents)
	_ = d.Set(PREREQUISITES, prerequisitesToResourceData(environment.Prerequisites))

	rules, err := rulesToResourceData(environment.Rules)
	if err != nil {
		return fmt.Errorf("failed to read rules on flag with key %q: %v", flagKey, err)
	}
	err = d.Set(RULES, rules)
	if err != nil {
		return fmt.Errorf("failed to set rules on flag with key %q: %v", flagKey, err)
	}

	err = d.Set(TARGETS, targetsToResourceData(environment.Targets))
	if err != nil {
		return fmt.Errorf("failed to set targets on flag with key %q: %v", flagKey, err)
	}

	err = d.Set(FALLTHROUGH, fallthroughToResourceData(environment.Fallthrough))
	if err != nil {
		return fmt.Errorf("failed to set flag fallthrough on flag with key %q: %v", flagKey, err)
	}

	err = d.Set(OFF_VARIATION, environment.OffVariation)
	if err != nil {
		return fmt.Errorf("failed to set off_variation on flag with key %q: %v", flagKey, err)
	}

	return nil
}

func patchFlagEnvPath(d *schema.ResourceData, op string) string {
	path := []string{"/environments"}
	path = append(path, d.Get(ENV_KEY).(string))
	path = append(path, op)

	return strings.Join(path, "/")
}

package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v15"
)

type featureFlagEnvSchemaOptions struct {
	isDataSource bool
}

func baseFeatureFlagEnvironmentSchema(options featureFlagEnvSchemaOptions) map[string]*schema.Schema {
	schemaMap := map[string]*schema.Schema{
		FLAG_ID: {
			Type:             schema.TypeString,
			Required:         true,
			Description:      addForceNewDescription("The feature flag's unique `id` in the format `project_key/flag_key`.", !options.isDataSource),
			ForceNew:         !options.isDataSource,
			ValidateDiagFunc: validation.ToDiagFunc(validateFlagID),
		},
		ENV_KEY: {
			Type:             schema.TypeString,
			Required:         true,
			Description:      addForceNewDescription("The environment key.", !options.isDataSource),
			ForceNew:         !options.isDataSource,
			ValidateDiagFunc: validateKey(),
		},
		ON: {
			Type:        schema.TypeBool,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "Whether targeting is enabled. Defaults to `false` if not set.",
			Default:     false,
		},
		TARGETS:         targetsSchema(options.isDataSource),
		CONTEXT_TARGETS: contextTargetsSchema(options.isDataSource),
		RULES:           rulesSchema(options.isDataSource),
		PREREQUISITES:   prerequisitesSchema(options.isDataSource),
		FALLTHROUGH:     fallthroughSchema(options.isDataSource),
		TRACK_EVENTS: {
			Type:        schema.TypeBool,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "Whether to send event data back to LaunchDarkly. Defaults to `false` if not set.",
			Default:     false,
		},
		OFF_VARIATION: {
			Type:             schema.TypeInt,
			Required:         !options.isDataSource,
			Computed:         options.isDataSource,
			Description:      "The index of the variation to serve if targeting is disabled.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
		},
	}
	if options.isDataSource {
		schemaMap = removeInvalidFieldsForDataSource(schemaMap)
	}
	return schemaMap
}

// get FeatureFlagEnvironment uses a query parameter to get the ldapi.FeatureFlag with only a single environment.
func getFeatureFlagEnvironment(client *Client, projectKey, flagKey, environmentKey string) (*ldapi.FeatureFlag, *http.Response, error) {
	return client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, flagKey).Env(environmentKey).Execute()
}

func featureFlagEnvironmentRead(ctx context.Context, d *schema.ResourceData, raw interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := raw.(*Client)
	flagId := d.Get(FLAG_ID).(string)
	projectKey, flagKey, err := flagIdToKeys(flagId)
	if err != nil {
		return diag.FromErr(err)
	}
	envKey := d.Get(ENV_KEY).(string)

	envExists, err := environmentExists(projectKey, envKey, client)

	if err != nil {
		return diag.FromErr(err)
	}

	if !envExists {
		log.Printf("[WARN] failed to find environment %q in project %q, removing resource from state", envKey, projectKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find environment %q in project %q, removing resource from state", envKey, projectKey),
		})
		d.SetId("")
		return diags
	}

	flag, res, err := getFeatureFlagEnvironment(client, projectKey, flagKey, envKey)
	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[WARN] failed to find flag %q in project %q, removing from state", flagKey, projectKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find flag %q in project %q, removing from state", flagKey, projectKey),
		})
		d.SetId("")
		return diags
	}

	if err != nil {
		return diag.Errorf("failed to get flag %q of project %q: %s", flagKey, projectKey, handleLdapiErr(err))
	}

	environment, ok := flag.Environments[envKey]
	if !ok {
		log.Printf("[WARN] failed to find environment %q for flag %q, removing from state", envKey, flagKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find environment %q for flag %q, removing from state", envKey, flagKey),
		})
		d.SetId("")
		return diags
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
		return diag.Errorf("failed to read rules on flag with key %q: %v", flagKey, err)
	}
	err = d.Set(RULES, rules)
	if err != nil {
		return diag.Errorf("failed to set rules on flag with key %q: %v", flagKey, err)
	}

	err = d.Set(TARGETS, targetsToResourceData(environment.Targets, targetOptions{isContextTarget: false}))
	if err != nil {
		return diag.Errorf("failed to set targets on flag with key %q: %v", flagKey, err)
	}

	err = d.Set(CONTEXT_TARGETS, targetsToResourceData(environment.ContextTargets, targetOptions{isContextTarget: true}))
	if err != nil {
		return diag.Errorf("failed to set context targets on flag with key %q: %v", flagKey, err)
	}

	err = d.Set(FALLTHROUGH, fallthroughToResourceData(*environment.Fallthrough))
	if err != nil {
		return diag.Errorf("failed to set flag fallthrough on flag with key %q: %v", flagKey, err)
	}

	err = d.Set(OFF_VARIATION, environment.OffVariation)
	if err != nil {
		return diag.Errorf("failed to set off_variation on flag with key %q: %v", flagKey, err)
	}

	return diags
}

func patchFlagEnvPath(d *schema.ResourceData, op string) string {
	path := []string{"/environments"}
	path = append(path, d.Get(ENV_KEY).(string))
	path = append(path, op)

	return strings.Join(path, "/")
}

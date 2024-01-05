package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v14"
)

func resourceFeatureFlagEnvironment() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFeatureFlagEnvironmentCreate,
		ReadContext:   resourceFeatureFlagEnvironmentRead,
		UpdateContext: resourceFeatureFlagEnvironmentUpdate,
		DeleteContext: resourceFeatureFlagEnvironmentDelete,

		Importer: &schema.ResourceImporter{
			State: resourceFeatureFlagEnvironmentImport,
		},
		Schema: baseFeatureFlagEnvironmentSchema(featureFlagEnvSchemaOptions{isDataSource: false}),

		Description: "Provides a LaunchDarkly environment-specific feature flag resource.\n\nThis resource allows you to create and manage environment-specific feature flags attributes within your LaunchDarkly organization.\n\n-> **Note:** If you intend to attach a feature flag to any experiments, we do _not_ recommend configuring environment-specific flag settings using Terraform. Subsequent applies may overwrite the changes made by experiments and break your experiment. An alternate workaround is to use the [lifecycle.ignore_changes](https://developer.hashicorp.com/terraform/language/meta-arguments/lifecycle#ignore_changes) Terraform meta-argument on the `fallthrough` field to prevent potential overwrites.",
	}
}

func validateFlagID(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	if strings.Count(v, "/") != 1 {
		return warns, append(errs, fmt.Errorf("%q must be in the format 'project_key/flag_key'. Got: %s", key, v))
	}
	for _, part := range strings.SplitN(v, "/", 2) {
		w, e := validateKeyNoDiag()(part, key)
		if len(e) > 0 {
			return w, e
		}
	}
	return warns, errs
}

func resourceFeatureFlagEnvironmentCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	flagId := d.Get(FLAG_ID).(string)

	projectKey, flagKey, err := flagIdToKeys(flagId)
	if err != nil {
		return diag.FromErr(err)
	}
	envKey := d.Get(ENV_KEY).(string)

	if exists, err := projectExists(projectKey, client); !exists {
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.Errorf("cannot find project with key %q", projectKey)
	}

	if exists, err := environmentExists(projectKey, envKey, client); !exists {
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.Errorf("failed to find environment with key %q", envKey)
	}

	patches := make([]ldapi.PatchOperation, 0)

	on := d.Get(ON)
	patches = append(patches, patchReplace(patchFlagEnvPath(d, "on"), on))

	// off_variation is required
	offVariation := d.Get(OFF_VARIATION)
	patches = append(patches, patchReplace(patchFlagEnvPath(d, "offVariation"), offVariation.(int)))

	trackEvents, ok := d.GetOk(TRACK_EVENTS)
	if ok {
		patches = append(patches, patchReplace(patchFlagEnvPath(d, "trackEvents"), trackEvents.(bool)))
	}

	_, ok = d.GetOk(RULES)
	if ok {
		rules, err := rulesFromResourceData(d)
		if err != nil {
			return diag.FromErr(err)
		}
		patches = append(patches, patchReplace(patchFlagEnvPath(d, "rules"), rules))
	}

	_, ok = d.GetOk(PREREQUISITES)
	if ok {
		prerequisites := prerequisitesFromResourceData(d, PREREQUISITES)
		patches = append(patches, patchReplace(patchFlagEnvPath(d, "prerequisites"), prerequisites))
	}

	_, ok = d.GetOk(TARGETS)
	if ok {
		targets := targetsFromResourceData(d, targetOptions{isContextTarget: false})
		patches = append(patches, patchReplace(patchFlagEnvPath(d, "targets"), targets))
	}

	_, ok = d.GetOk(CONTEXT_TARGETS)
	if ok {
		context_targets := targetsFromResourceData(d, targetOptions{isContextTarget: true})
		patches = append(patches, patchReplace(patchFlagEnvPath(d, "contextTargets"), context_targets))
	}

	// fallthrough is required
	fall, err := fallthroughFromResourceData(d)
	if err != nil {
		return diag.FromErr(err)
	}
	patches = append(patches, patchReplace(patchFlagEnvPath(d, "fallthrough"), fall))

	if len(patches) > 0 {
		comment := "Terraform"
		patch := ldapi.PatchWithComment{
			Comment: &comment,
			Patch:   patches,
		}
		log.Printf("[DEBUG] %+v\n", patch)

		_, _, err = client.ld.FeatureFlagsApi.PatchFeatureFlag(client.ctx, projectKey, flagKey).PatchWithComment(patch).Execute()
		if err != nil {
			return diag.Errorf("failed to update flag %q in project %q: %s", flagKey, projectKey, handleLdapiErr(err))
		}
	}

	d.SetId(projectKey + "/" + envKey + "/" + flagKey)
	return resourceFeatureFlagEnvironmentRead(ctx, d, metaRaw)
}

func resourceFeatureFlagEnvironmentRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return featureFlagEnvironmentRead(ctx, d, metaRaw, false)
}

func resourceFeatureFlagEnvironmentUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	flagId := d.Get(FLAG_ID).(string)
	projectKey, flagKey, err := flagIdToKeys(flagId)
	if err != nil {
		return diag.FromErr(err)
	}
	envKey := d.Get(ENV_KEY).(string)

	if exists, err := projectExists(projectKey, client); !exists {
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.Errorf("cannot find project with key %q", projectKey)
	}

	if exists, err := environmentExists(projectKey, envKey, client); !exists {
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.Errorf("failed to find environment with key %q", envKey)
	}

	patchOperations := make([]ldapi.PatchOperation, 0, 7)
	if d.HasChange(ON) {
		on := d.Get(ON)
		patchOperations = append(patchOperations, patchReplace(patchFlagEnvPath(d, "on"), on))
	}

	if d.HasChange(RULES) {
		rules, err := rulesFromResourceData(d)
		if err != nil {
			return diag.FromErr(err)
		}
		patchOperations = append(patchOperations, patchReplace(patchFlagEnvPath(d, "rules"), rules))
	}

	if d.HasChange(TRACK_EVENTS) {
		trackEvents := d.Get(TRACK_EVENTS).(bool)
		patchOperations = append(patchOperations, patchReplace(patchFlagEnvPath(d, "trackEvents"), trackEvents))
	}

	if d.HasChange(PREREQUISITES) {
		prerequisites := prerequisitesFromResourceData(d, PREREQUISITES)
		patchOperations = append(patchOperations, patchReplace(patchFlagEnvPath(d, "prerequisites"), prerequisites))
	}

	if d.HasChange(TARGETS) {
		targets := targetsFromResourceData(d, targetOptions{isContextTarget: false})
		patchOperations = append(patchOperations, patchReplace(patchFlagEnvPath(d, "targets"), targets))
	}

	if d.HasChange(CONTEXT_TARGETS) {
		contextTargets := targetsFromResourceData(d, targetOptions{isContextTarget: true})
		patchOperations = append(patchOperations, patchReplace(patchFlagEnvPath(d, "contextTargets"), contextTargets))
	}

	if d.HasChange(FALLTHROUGH) {
		fall, err := fallthroughFromResourceData(d)
		if err != nil {
			return diag.FromErr(err)
		}
		patchOperations = append(patchOperations, patchReplace(patchFlagEnvPath(d, "fallthrough"), fall))
	}

	if d.HasChange(OFF_VARIATION) {
		offVariation := d.Get(OFF_VARIATION)
		patchOperations = append(patchOperations, patchReplace(patchFlagEnvPath(d, "offVariation"), offVariation))
	}

	comment := "Terraform"
	patch := ldapi.PatchWithComment{Comment: &comment, Patch: patchOperations}
	log.Printf("[DEBUG] %+v\n", patch)

	if len(patchOperations) > 0 {
		_, _, err = client.ld.FeatureFlagsApi.PatchFeatureFlag(client.ctx, projectKey, flagKey).PatchWithComment(patch).Execute()
		if err != nil {
			return diag.Errorf("failed to update flag %q in project %q, environment %q: %s", flagKey, projectKey, envKey, handleLdapiErr(err))
		}
	}

	return resourceFeatureFlagEnvironmentRead(ctx, d, metaRaw)
}

func resourceFeatureFlagEnvironmentDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	flagId := d.Get(FLAG_ID).(string)
	projectKey, flagKey, err := flagIdToKeys(flagId)
	if err != nil {
		return diag.FromErr(err)
	}
	envKey := d.Get(ENV_KEY).(string)

	if exists, err := projectExists(projectKey, client); !exists {
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.Errorf("cannot find project with key %q", projectKey)
	}

	if exists, err := environmentExists(projectKey, envKey, client); !exists {
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.Errorf("failed to find environment with key %q", envKey)
	}

	flag, _, err := client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, flagKey).Execute()
	if err != nil {
		return diag.Errorf("failed to update flag %q in project %q, environment %q: %s", flagKey, projectKey, envKey, handleLdapiErr(err))
	}

	// Set off variation to match default with how a rule is created
	offVariation := len(flag.Variations) - 1

	comment := "Terraform"
	patch := ldapi.PatchWithComment{
		Comment: &comment,
		Patch: []ldapi.PatchOperation{
			patchReplace(patchFlagEnvPath(d, "on"), false),
			patchReplace(patchFlagEnvPath(d, "rules"), []ldapi.Rule{}),
			patchReplace(patchFlagEnvPath(d, "trackEvents"), false),
			patchReplace(patchFlagEnvPath(d, "prerequisites"), []ldapi.Prerequisite{}),
			patchReplace(patchFlagEnvPath(d, "offVariation"), offVariation),
			patchReplace(patchFlagEnvPath(d, "targets"), []ldapi.Target{}),
			patchReplace(patchFlagEnvPath(d, "contextTargets"), []ldapi.Target{}),
			patchReplace(patchFlagEnvPath(d, "fallthough"), fallthroughModel{Variation: intPtr(0)}),
		}}
	log.Printf("[DEBUG] %+v\n", patch)

	_, _, err = client.ld.FeatureFlagsApi.PatchFeatureFlag(client.ctx, projectKey, flagKey).PatchWithComment(patch).Execute()
	if err != nil {
		return diag.Errorf("failed to update flag %q in project %q, environment %q: %s", flagKey, projectKey, envKey, handleLdapiErr(err))
	}

	return diags
}

func resourceFeatureFlagEnvironmentImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()

	if strings.Count(id, "/") != 2 {
		return nil, fmt.Errorf("found unexpected flag id format: %q expected format: 'project_key/env_key/flag_key'", id)
	}
	parts := strings.SplitN(id, "/", 3)
	projectKey, envKey, flagKey := parts[0], parts[1], parts[2]
	_ = d.Set(FLAG_ID, projectKey+"/"+flagKey)
	_ = d.Set(ENV_KEY, envKey)

	return []*schema.ResourceData{d}, nil
}

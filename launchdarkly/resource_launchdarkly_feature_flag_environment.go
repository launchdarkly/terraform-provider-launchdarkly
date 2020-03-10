package launchdarkly

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go"
)

func resourceFeatureFlagEnvironment() *schema.Resource {
	return &schema.Resource{
		Create: resourceFeatureFlagEnvironmentCreate,
		Read:   resourceFeatureFlagEnvironmentRead,
		Update: resourceFeatureFlagEnvironmentUpdate,
		Delete: resourceFeatureFlagEnvironmentDelete,

		Importer: &schema.ResourceImporter{
			State: resourceFeatureFlagEnvironmentImport,
		},
		Schema: map[string]*schema.Schema{
			FLAG_ID: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateFlagID,
			},
			ENV_KEY: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateKey(),
			},
			TARGETING_ENABLED: {
				Type:     schema.TypeBool,
				Optional: true,
			},
			USER_TARGETS:     targetsSchema(),
			RULES:            rulesSchema(),
			PREREQUISITES:    prerequisitesSchema(),
			FLAG_FALLTHROUGH: fallthroughSchema(),
			TRACK_EVENTS: {
				Type:     schema.TypeBool,
				Optional: true,
			},
			OFF_VARIATION: {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(0),
			},
		},
	}
}

func validateFlagID(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	if strings.Count(v, "/") != 1 {
		return warns, append(errs, fmt.Errorf("%q must be in the format 'project_key/flag_key'. Got: %s", key, v))
	}
	for _, part := range strings.SplitN(v, "/", 2) {
		w, e := validateKey()(part, key)
		if len(e) > 0 {
			return w, e
		}
	}
	return warns, errs
}

func resourceFeatureFlagEnvironmentCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	flagId := d.Get(FLAG_ID).(string)

	projectKey, flagKey, err := flagIdToKeys(flagId)
	if err != nil {
		return err
	}
	envKey := d.Get(ENV_KEY).(string)

	if exists, err := projectExists(projectKey, client); !exists {
		if err != nil {
			return err
		}
		return fmt.Errorf("Cannot find project with key %q", projectKey)
	}

	if exists, err := environmentExists(projectKey, envKey, client); !exists {
		if err != nil {
			return err
		}
		return fmt.Errorf("failed to find environment with key %q", envKey)
	}

	enabled := d.Get(TARGETING_ENABLED).(bool)
	rules, err := rulesFromResourceData(d)
	if err != nil {
		return err
	}
	trackEvents := d.Get(TRACK_EVENTS).(bool)
	prerequisites := prerequisitesFromResourceData(d, PREREQUISITES)
	offVariation := d.Get(OFF_VARIATION).(int)
	targets := targetsFromResourceData(d, USER_TARGETS)

	fall, err := fallthroughFromResourceData(d)
	if err != nil {
		return err
	}

	patch := ldapi.PatchComment{
		Comment: "Terraform",
		Patch: []ldapi.PatchOperation{
			patchReplace(patchFlagEnvPath(d, "on"), enabled),
			patchReplace(patchFlagEnvPath(d, "rules"), rules),
			patchReplace(patchFlagEnvPath(d, "trackEvents"), trackEvents),
			patchReplace(patchFlagEnvPath(d, "prerequisites"), prerequisites),
			patchReplace(patchFlagEnvPath(d, "offVariation"), offVariation),
			patchReplace(patchFlagEnvPath(d, "targets"), targets),
			patchReplace(patchFlagEnvPath(d, "fallthrough"), fall),
		}}

	log.Printf("[DEBUG] %+v\n", patch)

	_, _, err = repeatUntilNoConflict(func() (interface{}, *http.Response, error) {
		return client.ld.FeatureFlagsApi.PatchFeatureFlag(client.ctx, projectKey, flagKey, patch)
	})
	if err != nil {
		return fmt.Errorf("failed to update flag %q in project %q: %s", flagKey, projectKey, handleLdapiErr(err))
	}

	d.SetId(projectKey + "/" + envKey + "/" + flagKey)
	return resourceFeatureFlagEnvironmentRead(d, metaRaw)
}

func resourceFeatureFlagEnvironmentRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	flagId := d.Get(FLAG_ID).(string)
	projectKey, flagKey, err := flagIdToKeys(flagId)
	if err != nil {
		return err
	}
	envKey := d.Get(ENV_KEY).(string)

	flag, _, err := client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, flagKey, nil)

	if err != nil {
		return fmt.Errorf("failed to get flag %q of project %q: %s", flagKey, projectKey, handleLdapiErr(err))
	}

	_ = d.Set(KEY, flag.Key)
	_ = d.Set(TARGETING_ENABLED, flag.Environments[envKey].On)

	err = d.Set(RULES, rulesToResourceData(flag.Environments[envKey].Rules))
	if err != nil {
		return fmt.Errorf("failed to set rules on flag with key %q: %v", flagKey, err)
	}

	err = d.Set(USER_TARGETS, targetsToResourceData(flag.Environments[envKey].Targets))
	if err != nil {
		return fmt.Errorf("failed to set targets on flag with key %q: %v", flagKey, err)
	}

	if _, ok := d.GetOk(FLAG_FALLTHROUGH); ok {
		err = d.Set(FLAG_FALLTHROUGH, fallthroughToResourceData(flag.Environments[envKey].Fallthrough_))
		if err != nil {
			return fmt.Errorf("failed to set flag fallthrough on flag with key %q: %v", flagKey, err)
		}
	}

	if _, ok := d.GetOk(RULES); ok {
		err = d.Set(RULES, rulesToResourceData(flag.Environments[envKey].Rules))
		if err != nil {
			return fmt.Errorf("failed to set targeting rules on flag with key %q: %v", flagKey, err)
		}
	}

	return nil
}

func resourceFeatureFlagEnvironmentUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	flagId := d.Get(FLAG_ID).(string)
	projectKey, flagKey, err := flagIdToKeys(flagId)
	if err != nil {
		return err
	}
	envKey := d.Get(ENV_KEY).(string)

	if exists, err := projectExists(projectKey, client); !exists {
		if err != nil {
			return err
		}
		return fmt.Errorf("Cannot find project with key %q", projectKey)
	}

	if exists, err := environmentExists(projectKey, envKey, client); !exists {
		if err != nil {
			return err
		}
		return fmt.Errorf("failed to find environment with key %q", envKey)
	}

	enabled := d.Get(TARGETING_ENABLED).(bool)
	rules, err := rulesFromResourceData(d)
	if err != nil {
		return err
	}
	trackEvents := d.Get(TRACK_EVENTS).(bool)
	prerequisites := prerequisitesFromResourceData(d, PREREQUISITES)
	targets := targetsFromResourceData(d, USER_TARGETS)
	offVariation := d.Get(OFF_VARIATION).(int)

	fall, err := fallthroughFromResourceData(d)
	if err != nil {
		return err
	}

	patch := ldapi.PatchComment{
		Comment: "Terraform",
		Patch: []ldapi.PatchOperation{
			patchReplace(patchFlagEnvPath(d, "on"), enabled),
			patchReplace(patchFlagEnvPath(d, "rules"), rules),
			patchReplace(patchFlagEnvPath(d, "trackEvents"), trackEvents),
			patchReplace(patchFlagEnvPath(d, "prerequisites"), prerequisites),
			patchReplace(patchFlagEnvPath(d, "offVariation"), offVariation),
			patchReplace(patchFlagEnvPath(d, "targets"), targets),
			patchReplace(patchFlagEnvPath(d, "fallthrough"), fall),
		}}

	log.Printf("[DEBUG] %+v\n", patch)
	_, _, err = repeatUntilNoConflict(func() (interface{}, *http.Response, error) {
		return client.ld.FeatureFlagsApi.PatchFeatureFlag(client.ctx, projectKey, flagKey, patch)
	})
	if err != nil {
		return fmt.Errorf("failed to update flag %q in project %q, environment %q: %s", flagKey, projectKey, envKey, handleLdapiErr(err))
	}
	return resourceFeatureFlagEnvironmentRead(d, metaRaw)
}

func resourceFeatureFlagEnvironmentDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	flagId := d.Get(FLAG_ID).(string)
	projectKey, flagKey, err := flagIdToKeys(flagId)
	if err != nil {
		return err
	}
	envKey := d.Get(ENV_KEY).(string)

	if exists, err := projectExists(projectKey, client); !exists {
		if err != nil {
			return err
		}
		return fmt.Errorf("Cannot find project with key %q", projectKey)
	}

	if exists, err := environmentExists(projectKey, envKey, client); !exists {
		if err != nil {
			return err
		}
		return fmt.Errorf("failed to find environment with key %q", envKey)
	}

	flag, _, err := client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, flagKey, nil)
	if err != nil {
		return fmt.Errorf("failed to update flag %q in project %q, environment %q: %s", flagKey, projectKey, envKey, handleLdapiErr(err))
	}

	// Set off variation to match default with how a rule is created
	offVariation := len(flag.Variations) - 1

	patch := ldapi.PatchComment{
		Comment: "Terraform",
		Patch: []ldapi.PatchOperation{
			patchReplace(patchFlagEnvPath(d, "on"), false),
			patchReplace(patchFlagEnvPath(d, "rules"), []ldapi.Rule{}),
			patchReplace(patchFlagEnvPath(d, "trackEvents"), false),
			patchReplace(patchFlagEnvPath(d, "prerequisites"), []ldapi.Prerequisite{}),
			patchReplace(patchFlagEnvPath(d, "offVariation"), offVariation),
			patchReplace(patchFlagEnvPath(d, "targets"), []ldapi.Target{}),
			patchReplace(patchFlagEnvPath(d, "fallthough"), fallthroughModel{Variation: intPtr(0)}),
		}}
	log.Printf("[DEBUG] %+v\n", patch)

	_, _, err = repeatUntilNoConflict(func() (interface{}, *http.Response, error) {
		return client.ld.FeatureFlagsApi.PatchFeatureFlag(client.ctx, projectKey, flagKey, patch)
	})
	if err != nil {
		return fmt.Errorf("failed to update flag %q in project %q, environment %q: %s", flagKey, projectKey, envKey, handleLdapiErr(err))
	}

	return nil
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

	err := resourceFeatureFlagEnvironmentRead(d, meta)
	if err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func patchFlagEnvPath(d *schema.ResourceData, op string) string {
	path := []string{"/environments"}
	path = append(path, d.Get(ENV_KEY).(string))
	path = append(path, op)

	return strings.Join(path, "/")
}

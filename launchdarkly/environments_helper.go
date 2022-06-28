package launchdarkly

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v10"
)

// baseEnvironmentSchema covers the overlap between the data source and resource schemas
// certain attributes are required for the resource that are not for the data source and so those
// will need to be differentiated
func baseEnvironmentSchema(forProject bool) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		KEY: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "A project-unique key for the new environment",
			// Don't force new if the environment schema will be nested in a project
			ForceNew:         !forProject,
			ValidateDiagFunc: validateKey(),
		},
		API_KEY: {
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
		MOBILE_KEY: {
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
		CLIENT_SIDE_ID: {
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
		DEFAULT_TTL: {
			Type:     schema.TypeInt,
			Optional: true,
			Default:  0,
			// Default TTL should be between 0 and 60 minutes: https://docs.launchdarkly.com/docs/environments
			Description:      "The TTL for the environment. This must be between 0 and 60 minutes. The TTL setting only applies to environments using the PHP SDK",
			ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 60)),
		},
		SECURE_MODE: {
			Default:     false,
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether or not to use secure mode. Secure mode ensures a user of the client-side SDK cannot impersonate another user",
		},
		DEFAULT_TRACK_EVENTS: {
			Default:     false,
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether or not to default to sending data export events for flags created in the environment",
		},
		REQUIRE_COMMENTS: {
			Default:     false,
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether or not to require comments for flag and segment changes in this environment",
		},
		CONFIRM_CHANGES: {
			Default:     false,
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether or not to require confirmation for flag and segment changes in this environment",
		},
		TAGS:              tagsSchema(),
		APPROVAL_SETTINGS: approvalSchema(),
	}
}

func getEnvironmentUpdatePatches(oldConfig, config map[string]interface{}) ([]ldapi.PatchOperation, error) {
	// Always include required fields
	name := config[NAME]
	color := config[COLOR]
	patches := []ldapi.PatchOperation{
		patchReplace("/name", name),
		patchReplace("/color", color),
	}

	// Add optional fields if they exist
	defaultTtl, ok := config[DEFAULT_TTL]
	if ok {
		patches = append(patches, patchReplace("/defaultTtl", defaultTtl))
	}

	secureMode, ok := config[SECURE_MODE]
	if ok {
		patches = append(patches, patchReplace("/secureMode", secureMode))
	}

	defaultTrackEvents, ok := config[DEFAULT_TRACK_EVENTS]
	if ok {
		patches = append(patches, patchReplace("/defaultTrackEvents", defaultTrackEvents))
	}

	requireComments, ok := config[REQUIRE_COMMENTS]
	if ok {
		patches = append(patches, patchReplace("/requireComments", requireComments))
	}

	confirmChanges, ok := config[CONFIRM_CHANGES]
	if ok {
		patches = append(patches, patchReplace("/confirmChanges", confirmChanges))
	}

	tags, ok := config[TAGS]
	if ok {
		envTags := stringsFromSchemaSet(tags.(*schema.Set))
		patches = append(patches, patchReplace("/tags", &envTags))
	}

	var oldApprovalSettings []interface{}
	if oldSettings, ok := oldConfig[APPROVAL_SETTINGS]; ok {
		oldApprovalSettings = oldSettings.([]interface{})
	}
	newApprovalSettings := config[APPROVAL_SETTINGS]
	approvalPatches, err := approvalPatchFromSettings(oldApprovalSettings, newApprovalSettings)
	if err != nil {
		return []ldapi.PatchOperation{}, err
	}
	patches = append(patches, approvalPatches...)
	return patches, nil
}

func environmentSchema(forProject bool) map[string]*schema.Schema {
	schemaMap := baseEnvironmentSchema(forProject)
	schemaMap[NAME] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The name of the new environment",
	}
	schemaMap[COLOR] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "A color swatch (as an RGB hex value with no leading '#', e.g. C8C8C8)",
	}
	return schemaMap
}

func dataSourceEnvironmentSchema(forProject bool) map[string]*schema.Schema {
	schemaMap := baseEnvironmentSchema(forProject)
	schemaMap[NAME] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	schemaMap[COLOR] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	return schemaMap
}

func environmentPostsFromResourceData(d *schema.ResourceData) []ldapi.EnvironmentPost {
	schemaEnvList := d.Get(ENVIRONMENTS).([]interface{})
	envs := make([]ldapi.EnvironmentPost, len(schemaEnvList))
	for i, env := range schemaEnvList {
		envs[i] = environmentPostFromResourceData(env)
	}
	return envs
}

func environmentPostFromResourceData(env interface{}) ldapi.EnvironmentPost {
	envMap := env.(map[string]interface{})
	envPost := ldapi.EnvironmentPost{
		Name:  envMap[NAME].(string),
		Key:   envMap[KEY].(string),
		Color: envMap[COLOR].(string),
	}

	if defaultTTL, ok := envMap[DEFAULT_TTL]; ok {
		envPost.DefaultTtl = ldapi.PtrInt32(int32(defaultTTL.(int)))
	}
	return envPost
}

type envResourceData map[string]interface{}

func environmentsToResourceDataMap(envs []ldapi.Environment) map[string]envResourceData {
	envMap := make(map[string]envResourceData, len(envs))
	for _, env := range envs {
		envMap[env.Key] = environmentToResourceData(env)
	}

	return envMap
}

func environmentToResourceData(env ldapi.Environment) envResourceData {
	envData := envResourceData{
		KEY:                  env.Key,
		NAME:                 env.Name,
		API_KEY:              env.ApiKey,
		MOBILE_KEY:           env.MobileKey,
		CLIENT_SIDE_ID:       env.Id,
		COLOR:                env.Color,
		DEFAULT_TTL:          int(env.DefaultTtl),
		SECURE_MODE:          env.SecureMode,
		DEFAULT_TRACK_EVENTS: env.DefaultTrackEvents,
		REQUIRE_COMMENTS:     env.RequireComments,
		CONFIRM_CHANGES:      env.ConfirmChanges,
		TAGS:                 env.Tags,
	}
	if env.ApprovalSettings != nil {
		envData[APPROVAL_SETTINGS] = approvalSettingsToResourceData(*env.ApprovalSettings)
	}
	return envData
}

func rawEnvironmentConfigsToKeyList(rawEnvs []interface{}) []string {
	keys := make([]string, 0, len(rawEnvs))
	for _, rawEnv := range rawEnvs {
		env := rawEnv.(map[string]interface{})
		envKey := env[KEY].(string)
		keys = append(keys, envKey)
	}
	return keys
}

func environmentRead(ctx context.Context, d *schema.ResourceData, meta interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	env, res, err := client.ld.EnvironmentsApi.GetEnvironment(client.ctx, projectKey, key).Execute()

	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[WARN] failed to find environment with key %q in project %q, removing from state", key, projectKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find environment with key %q in project %q, removing from state", key, projectKey),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get environment with key %q for project key: %q: %v", key, projectKey, handleLdapiErr(err))
	}

	d.SetId(projectKey + "/" + key)
	_ = d.Set(KEY, env.Key)
	_ = d.Set(NAME, env.Name)
	_ = d.Set(API_KEY, env.ApiKey)
	_ = d.Set(MOBILE_KEY, env.MobileKey)
	_ = d.Set(CLIENT_SIDE_ID, env.Id)
	_ = d.Set(COLOR, env.Color)
	_ = d.Set(DEFAULT_TTL, int(env.DefaultTtl))
	_ = d.Set(SECURE_MODE, env.SecureMode)
	_ = d.Set(DEFAULT_TRACK_EVENTS, env.DefaultTrackEvents)
	_ = d.Set(TAGS, env.Tags)
	_ = d.Set(REQUIRE_COMMENTS, env.RequireComments)
	_ = d.Set(CONFIRM_CHANGES, env.ConfirmChanges)

	if env.ApprovalSettings != nil {
		err = d.Set(APPROVAL_SETTINGS, approvalSettingsToResourceData(*env.ApprovalSettings))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

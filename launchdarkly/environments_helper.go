package launchdarkly

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v15"
)

type environmentSchemaOptions struct {
	forProject   bool
	isDataSource bool
}

// baseEnvironmentSchema covers the overlap between the data source and resource schemas
// certain attributes are required for the resource that are not for the data source and so those
// will need to be differentiated
func baseEnvironmentSchema(options environmentSchemaOptions) map[string]*schema.Schema {
	envSchema := map[string]*schema.Schema{
		KEY: {
			Type:        schema.TypeString,
			Required:    true,
			Description: addForceNewDescription("The project-unique key for the environment.", !options.isDataSource),
			// Don't force new if the environment schema will be nested in a project
			ForceNew:         !options.forProject,
			ValidateDiagFunc: validateKey(),
		},
		CRITICAL: {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Denotes whether the environment is critical.",
		},
		API_KEY: {
			Type:        schema.TypeString,
			Computed:    true,
			Sensitive:   true,
			Description: "The environment's SDK key.",
		},
		MOBILE_KEY: {
			Type:        schema.TypeString,
			Computed:    true,
			Sensitive:   true,
			Description: "The environment's mobile key.",
		},
		CLIENT_SIDE_ID: {
			Type:        schema.TypeString,
			Computed:    true,
			Sensitive:   true,
			Description: "The environment's client-side ID.",
		},
		DEFAULT_TTL: {
			Type:             schema.TypeInt,
			Optional:         !options.isDataSource,
			Computed:         options.isDataSource,
			Default:          0,
			ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 60)),
			// Default TTL should be between 0 and 60 minutes: https://docs.launchdarkly.com/home/organize/environments#ttl-settings
			Description: "The TTL for the environment. This must be between 0 and 60 minutes. The TTL setting only applies to environments using the PHP SDK. This field will default to `0` when not set. To learn more, read [TTL settings](https://docs.launchdarkly.com/home/organize/environments#ttl-settings).",
		},
		SECURE_MODE: {
			Default:     false,
			Type:        schema.TypeBool,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "Set to `true` to ensure a user of the client-side SDK cannot impersonate another user. This field will default to `false` when not set.",
		},
		DEFAULT_TRACK_EVENTS: {
			Default:     false,
			Type:        schema.TypeBool,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "Set to `true` to enable data export for every flag created in this environment after you configure this argument. This field will default to `false` when not set. To learn more, read [Data Export](https://docs.launchdarkly.com/home/data-export).",
		},
		REQUIRE_COMMENTS: {
			Default:     false,
			Type:        schema.TypeBool,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "Set to `true` if this environment requires comments for flag and segment changes. This field will default to `false` when not set.",
		},
		CONFIRM_CHANGES: {
			Type:        schema.TypeBool,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Default:     false,
			Description: "Set to `true` if this environment requires confirmation for flag and segment changes. This field will default to `false` when not set.",
		},
		TAGS:              tagsSchema(tagsSchemaOptions{isDataSource: options.isDataSource}),
		APPROVAL_SETTINGS: approvalSchema(approvalSchemaOptions{isDataSource: options.isDataSource}),
	}

	if options.isDataSource {
		envSchema = removeInvalidFieldsForDataSource(envSchema)
	}

	return envSchema
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

	kind, ok := config[KIND]
	if ok {
		patches = append(patches, patchReplace("/kind", kind))
	}

	tags, ok := config[TAGS]
	if ok {
		envTags := stringsFromSchemaSet(tags.(*schema.Set))
		patches = append(patches, patchReplace("/tags", &envTags))
	}

	critical, ok := config[CRITICAL]
	if ok {
		patches = append(patches, patchReplace("/critical", critical))
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

func environmentSchema(options environmentSchemaOptions) map[string]*schema.Schema {
	schemaMap := baseEnvironmentSchema(options)
	schemaMap[NAME] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The name of the environment.",
	}
	schemaMap[COLOR] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The color swatch as an RGB hex value with no leading `#`. For example: `000000`",
	}
	return schemaMap
}

func dataSourceEnvironmentSchema(forProject bool) map[string]*schema.Schema {
	schemaMap := baseEnvironmentSchema(environmentSchemaOptions{forProject: true, isDataSource: true})
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
		CRITICAL:             env.Critical,
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
	_ = d.Set(CRITICAL, env.Critical) // We need to update the LaunchDarkly go api client's version of Environment
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

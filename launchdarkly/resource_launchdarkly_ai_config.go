package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func resourceAIConfig() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAIConfigCreate,
		ReadContext:   resourceAIConfigRead,
		UpdateContext: resourceAIConfigUpdate,
		DeleteContext: resourceAIConfigDelete,
		Exists:        resourceAIConfigExists,

		Importer: &schema.ResourceImporter{
			State: resourceAIConfigImport,
		},

		Schema: baseAIConfigSchema(false),

		Description: `Provides a LaunchDarkly AI Config resource.

This resource allows you to create and manage AI configurations within your LaunchDarkly organization.`,
	}
}

func baseAIConfigSchema(isDataSource bool) map[string]*schema.Schema {
	schemaMap := map[string]*schema.Schema{
		PROJECT_KEY: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			Description:      addForceNewDescription("The AI Config's project key. A change in this field will force the destruction of the existing resource and the creation of a new one.", !isDataSource),
			ValidateDiagFunc: validateKey(),
		},
		KEY: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validateKey(),
			Description:      addForceNewDescription("The unique key that references the AI Config. A change in this field will force the destruction of the existing resource and the creation of a new one.", !isDataSource),
		},
		NAME: {
			Type:        schema.TypeString,
			Required:    !isDataSource,
			Computed:    isDataSource,
			Description: "The human-friendly name for the AI Config.",
		},
		DESCRIPTION: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "The description of the AI Config's purpose.",
		},
		TAGS: tagsSchema(tagsSchemaOptions{isDataSource: isDataSource}),
		"variations": {
			Type:        schema.TypeList,
			Required:    !isDataSource,
			Computed:    isDataSource,
			Description: "List of variations for the AI Config.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					KEY: {
						Type:             schema.TypeString,
						Required:         true,
						Description:      "The unique key for the variation.",
						ValidateDiagFunc: validateKey(),
					},
					NAME: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The name of the variation.",
					},
					DESCRIPTION: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "The description of the variation.",
					},
					"model": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The AI model to use for this variation.",
					},
					"parameters": {
						Type:        schema.TypeMap,
						Optional:    true,
						Description: "Parameters for the AI model.",
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
					},
				},
			},
		},
	}

	if isDataSource {
		return removeInvalidFieldsForDataSource(schemaMap)
	}

	return schemaMap
}

func resourceAIConfigCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)

	if exists, err := projectExists(projectKey, client); !exists {
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.Errorf("cannot find project with key %q", projectKey)
	}

	key := d.Get(KEY).(string)
	name := d.Get(NAME).(string)
	description := d.Get(DESCRIPTION).(string)
	tags := stringsFromResourceData(d, TAGS)
	variations := aiConfigVariationsFromResourceData(d)

	aiConfigBody := ldapi.AIConfigPost{
		Key:         key,
		Name:        name,
		Description: ldapi.PtrString(description),
		Tags:        tags,
		Variations:  variations,
	}

	_, _, err := client.ld.AIConfigsApi.PostAIConfig(client.ctx, projectKey).AIConfigPost(aiConfigBody).Execute()

	if err != nil {
		return diag.Errorf("failed to create AI Config with name %q: %s", name, handleLdapiErr(err))
	}

	d.SetId(projectKey + "/" + key)
	return resourceAIConfigRead(ctx, d, metaRaw)
}

func resourceAIConfigRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	projectKey, configKey, err := aiConfigIdToKeys(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	aiConfig, res, err := client.ld.AIConfigsApi.GetAIConfig(client.ctx, projectKey, configKey).Execute()

	if isStatusNotFound(res) {
		log.Printf("[WARN] failed to find AI Config with id %q, removing from state", d.Id())
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find AI Config with id %q, removing from state", d.Id()),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get AI Config with id %q: %s", d.Id(), handleLdapiErr(err))
	}

	_ = d.Set(KEY, aiConfig.Key)
	_ = d.Set(NAME, aiConfig.Name)
	_ = d.Set(DESCRIPTION, aiConfig.Description)
	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(TAGS, aiConfig.Tags)

	variations, err := aiConfigVariationsToResourceData(aiConfig.Variations)
	if err != nil {
		return diag.Errorf("failed to transform AI Config variations: %v", err)
	}
	err = d.Set("variations", variations)
	if err != nil {
		return diag.Errorf("failed to set variations on AI Config with key %q: %v", configKey, err)
	}

	return diags
}

func resourceAIConfigUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey, configKey, err := aiConfigIdToKeys(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get(NAME).(string)
	description := d.Get(DESCRIPTION).(string)
	tags := stringsFromResourceData(d, TAGS)
	variations := aiConfigVariationsFromResourceData(d)

	patch := ldapi.PatchWithComment{
		Patch: []ldapi.PatchOperation{
			patchReplace("/name", &name),
			patchReplace("/description", &description),
			patchReplace("/tags", &tags),
			patchReplace("/variations", &variations),
		},
	}

	_, _, err = client.ld.AIConfigsApi.PatchAIConfig(client.ctx, projectKey, configKey).PatchWithComment(patch).Execute()
	if err != nil {
		return diag.Errorf("failed to update AI Config with key %q: %s", configKey, handleLdapiErr(err))
	}

	return resourceAIConfigRead(ctx, d, metaRaw)
}

func resourceAIConfigDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	projectKey, configKey, err := aiConfigIdToKeys(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.ld.AIConfigsApi.DeleteAIConfig(client.ctx, projectKey, configKey).Execute()

	if err != nil {
		return diag.Errorf("failed to delete AI Config with key %q: %s", configKey, handleLdapiErr(err))
	}

	return diags
}

func resourceAIConfigExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	projectKey, configKey, err := aiConfigIdToKeys(d.Id())
	if err != nil {
		return false, err
	}
	return aiConfigExists(projectKey, configKey, metaRaw.(*Client))
}

func aiConfigExists(projectKey, configKey string, meta *Client) (bool, error) {
	_, res, err := meta.ld.AIConfigsApi.GetAIConfig(meta.ctx, projectKey, configKey).Execute()
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get AI Config with key %q: %s", configKey, handleLdapiErr(err))
	}

	return true, nil
}

func resourceAIConfigImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	projectKey, configKey, err := aiConfigIdToKeys(d.Id())
	if err != nil {
		return nil, err
	}

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(KEY, configKey)

	return []*schema.ResourceData{d}, nil
}

func aiConfigIdToKeys(id string) (projectKey string, configKey string, err error) {
	if strings.Count(id, "/") != 1 {
		return "", "", fmt.Errorf("found unexpected AI Config id format: %q expected format: 'project_key/config_key'", id)
	}
	parts := strings.SplitN(id, "/", 2)
	projectKey, configKey = parts[0], parts[1]
	return projectKey, configKey, nil
}

func aiConfigVariationsFromResourceData(d *schema.ResourceData) []ldapi.AIConfigVariationPost {
	schemaVariations := d.Get("variations").([]interface{})
	variations := make([]ldapi.AIConfigVariationPost, len(schemaVariations))

	for i, variation := range schemaVariations {
		v := variation.(map[string]interface{})
		key := v[KEY].(string)
		name := v[NAME].(string)
		description := v[DESCRIPTION].(string)
		model := v["model"].(string)
		
		parametersRaw := v["parameters"].(map[string]interface{})
		parameters := make(map[string]string)
		for k, v := range parametersRaw {
			parameters[k] = v.(string)
		}

		variations[i] = ldapi.AIConfigVariationPost{
			Key:         key,
			Name:        name,
			Description: ldapi.PtrString(description),
			Model:       model,
			Parameters:  parameters,
		}
	}

	return variations
}

func aiConfigVariationsToResourceData(variations []ldapi.AIConfigVariation) ([]interface{}, error) {
	result := make([]interface{}, len(variations))

	for i, v := range variations {
		variation := map[string]interface{}{
			KEY:         v.Key,
			NAME:        v.Name,
			DESCRIPTION: v.Description,
			"model":     v.Model,
			"parameters": v.Parameters,
		}
		result[i] = variation
	}

	return result, nil
}

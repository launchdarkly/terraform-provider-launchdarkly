package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ldapi "github.com/launchdarkly/api-client-go/v17"
)

type AIConfigVariation struct {
	Key         string            `json:"key"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Model       string            `json:"model"`
	Parameters  map[string]string `json:"parameters,omitempty"`
}

type AIConfig struct {
	Key         string              `json:"key"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
	Variations  []AIConfigVariation `json:"variations"`
}

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

	aiConfig := AIConfig{
		Key:         key,
		Name:        name,
		Description: description,
		Tags:        tags,
		Variations:  variations,
	}

	_, _, err := client.postAIConfig(projectKey, aiConfig)
	if err != nil {
		return diag.Errorf("failed to create AI Config with name %q: %s", name, err)
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

	aiConfig, res, err := client.getAIConfig(projectKey, configKey)
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
		return diag.Errorf("failed to get AI Config with id %q: %s", d.Id(), err)
	}

	_ = d.Set(KEY, aiConfig.Key)
	_ = d.Set(NAME, aiConfig.Name)
	_ = d.Set(DESCRIPTION, aiConfig.Description)
	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(TAGS, aiConfig.Tags)

	variations := make([]interface{}, len(aiConfig.Variations))
	for i, v := range aiConfig.Variations {
		variation := map[string]interface{}{
			KEY:          v.Key,
			NAME:         v.Name,
			DESCRIPTION:  v.Description,
			"model":      v.Model,
			"parameters": v.Parameters,
		}
		variations[i] = variation
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

	patch := []ldapi.PatchOperation{
		patchReplace("/name", name),
		patchReplace("/description", description),
		patchReplace("/tags", tags),
		patchReplace("/variations", variations),
	}

	_, err = client.patchAIConfig(projectKey, configKey, patch)
	if err != nil {
		return diag.Errorf("failed to update AI Config with key %q: %s", configKey, err)
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

	_, err = client.deleteAIConfig(projectKey, configKey)
	if err != nil {
		return diag.Errorf("failed to delete AI Config with key %q: %s", configKey, err)
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
	_, res, err := meta.getAIConfig(projectKey, configKey)
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get AI Config with key %q: %s", configKey, err)
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

func aiConfigVariationsFromResourceData(d *schema.ResourceData) []AIConfigVariation {
	schemaVariations := d.Get("variations").([]interface{})
	variations := make([]AIConfigVariation, len(schemaVariations))

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

		variations[i] = AIConfigVariation{
			Key:         key,
			Name:        name,
			Description: description,
			Model:       model,
			Parameters:  parameters,
		}
	}

	return variations
}

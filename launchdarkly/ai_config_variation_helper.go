package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

func aiConfigVariationSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		PROJECT_KEY: {
			Type:             schema.TypeString,
			Required:         true,
			Description:      addForceNewDescription("The project key.", true),
			ForceNew:         true,
			ValidateDiagFunc: validateKey(),
		},
		AI_CONFIG_KEY: {
			Type:             schema.TypeString,
			Required:         true,
			Description:      addForceNewDescription("The AI config key that this variation belongs to.", true),
			ForceNew:         true,
			ValidateDiagFunc: validateKey(),
		},
		KEY: {
			Type:             schema.TypeString,
			Required:         true,
			Description:      addForceNewDescription("The variation's unique key.", true),
			ForceNew:         true,
			ValidateDiagFunc: validateKey(),
		},
		NAME: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The variation's human-readable name.",
		},
		MESSAGES: {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "A list of messages for completion mode. Each message has a `role` and `content`.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					ROLE: {
						Type:             schema.TypeString,
						Required:         true,
						Description:      "The role of the message. Must be one of `system`, `user`, `assistant`, or `developer`.",
						ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"system", "user", "assistant", "developer"}, false)),
					},
					CONTENT: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The content of the message.",
					},
				},
			},
		},
		MODEL: {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "A JSON string representing the inline model configuration for the variation.",
			ValidateFunc:     validateJsonStringFunc,
			DiffSuppressFunc: suppressEquivalentJsonDiffs,
		},
		MODEL_CONFIG_KEY: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The key of a model config resource to use for this variation.",
		},
		DESCRIPTION: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The variation's description (used in agent mode).",
		},
		INSTRUCTIONS: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The variation's instructions (used in agent mode).",
		},
		TOOL_KEYS: {
			Type:        schema.TypeSet,
			Optional:    true,
			Description: "A set of AI tool keys to associate with this variation.",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		STATE: {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "The state of the variation. Must be `archived` or `published`.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(
				[]string{"archived", "published"}, false,
			)),
		},
		VARIATION_ID: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The internal ID of the variation.",
		},
		VERSION: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The version number of the variation.",
		},
		CREATION_DATE: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The creation timestamp of the variation.",
		},
	}
}

func aiConfigVariationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	configKey := d.Get(AI_CONFIG_KEY).(string)
	variationKey := d.Get(KEY).(string)

	var variationsResp *ldapi.AIConfigVariationsResponse
	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		variationsResp, res, err = client.ld.AIConfigsApi.GetAIConfigVariation(client.ctx, projectKey, configKey, variationKey).Execute()
		return err
	})

	if isStatusNotFound(res) {
		log.Printf("[WARN] failed to find AI config variation with key %q in config %q project %q, removing from state if present", variationKey, configKey, projectKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find AI config variation with key %q in config %q project %q, removing from state if present", variationKey, configKey, projectKey),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get AI config variation with key %q in config %q project %q: %s", variationKey, configKey, projectKey, handleLdapiErr(err))
	}

	if variationsResp == nil || len(variationsResp.Items) == 0 {
		log.Printf("[WARN] AI config variation with key %q in config %q project %q returned no items, removing from state", variationKey, configKey, projectKey)
		d.SetId("")
		return diags
	}

	// Items contains all versions of the variation. Find the one with the highest version number.
	variation := variationsResp.Items[0]
	for _, v := range variationsResp.Items[1:] {
		if v.Version > variation.Version {
			variation = v
		}
	}
	log.Printf("[DEBUG] AI config variation %q: found %d versions, using version %d", variationKey, len(variationsResp.Items), variation.Version)

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(AI_CONFIG_KEY, configKey)
	_ = d.Set(KEY, variation.Key)
	_ = d.Set(NAME, variation.Name)
	_ = d.Set(VARIATION_ID, variation.Id)
	_ = d.Set(VERSION, variation.Version)
	_ = d.Set(CREATION_DATE, variation.CreatedAt)

	// Handle optional pointer fields
	description := ""
	if variation.Description != nil {
		description = *variation.Description
	}
	_ = d.Set(DESCRIPTION, description)

	instructions := ""
	if variation.Instructions != nil {
		instructions = *variation.Instructions
	}
	_ = d.Set(INSTRUCTIONS, instructions)

	modelConfigKey := ""
	if variation.ModelConfigKey != nil {
		modelConfigKey = *variation.ModelConfigKey
	}
	_ = d.Set(MODEL_CONFIG_KEY, modelConfigKey)

	state := ""
	if variation.State != nil {
		state = *variation.State
	}
	_ = d.Set(STATE, state)

	// Serialize model map to JSON string.
	// The API returns a default model object (e.g. {"custom":{},"modelName":"","parameters":{}})
	// even when no model was configured. Only write to state if the model has meaningful content
	// or the user had previously set it.
	if len(variation.Model) > 0 && !isEmptyModelMap(variation.Model) {
		modelJSON, err := mapToJsonString(variation.Model)
		if err != nil {
			return diag.Errorf("failed to serialize model for AI config variation %q: %s", variationKey, err)
		}
		_ = d.Set(MODEL, modelJSON)
	} else {
		_ = d.Set(MODEL, nil)
	}

	// Flatten messages
	_ = d.Set(MESSAGES, flattenMessages(variation.Messages))

	// Extract tool keys from Tools
	toolKeys := make([]string, len(variation.Tools))
	for i, t := range variation.Tools {
		toolKeys[i] = t.Key
	}
	_ = d.Set(TOOL_KEYS, toolKeys)

	return diags
}

func messagesFromResourceData(d *schema.ResourceData) []ldapi.Message {
	raw := d.Get(MESSAGES).([]interface{})
	messages := make([]ldapi.Message, len(raw))
	for i, v := range raw {
		m := v.(map[string]interface{})
		messages[i] = *ldapi.NewMessage(m[CONTENT].(string), m[ROLE].(string))
	}
	return messages
}

func flattenMessages(messages []ldapi.Message) []map[string]interface{} {
	result := make([]map[string]interface{}, len(messages))
	for i, m := range messages {
		result[i] = map[string]interface{}{
			ROLE:    m.Role,
			CONTENT: m.Content,
		}
	}
	return result
}

// isEmptyModelMap returns true if the model map only contains empty/zero values
// (as returned by the API default: {"custom":{},"modelName":"","parameters":{}}).
func isEmptyModelMap(m map[string]interface{}) bool {
	for _, v := range m {
		switch val := v.(type) {
		case string:
			if val != "" {
				return false
			}
		case map[string]interface{}:
			if len(val) > 0 {
				return false
			}
		default:
			if v != nil {
				return false
			}
		}
	}
	return true
}

func variationIdToKeys(id string) (projectKey, configKey, variationKey string, err error) {
	parts := splitID(id, 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("import ID must be in the format project_key/config_key/variation_key, got: %q", id)
	}
	return parts[0], parts[1], parts[2], nil
}

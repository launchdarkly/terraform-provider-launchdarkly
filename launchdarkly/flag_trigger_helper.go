package launchdarkly

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func baseFlagTriggerSchema(isDataSource bool) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		PROJECT_KEY: {
			Type:             schema.TypeString,
			Required:         true,
			Description:      addForceNewDescription("The unique key of the project encompassing the associated flag.", !isDataSource),
			ForceNew:         true,
			ValidateDiagFunc: validateKey(),
		},
		ENV_KEY: {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: addForceNewDescription("The unique key of the environment the flag trigger will work in.", !isDataSource),
		},
		FLAG_KEY: {
			Type:             schema.TypeString,
			Required:         true,
			Description:      addForceNewDescription("The unique key of the associated flag.", !isDataSource),
			ForceNew:         true,
			ValidateDiagFunc: validateKey(),
		},
		INTEGRATION_KEY: {
			Type:     schema.TypeString,
			Required: !isDataSource,
			Computed: isDataSource,
			Description: addForceNewDescription(
				fmt.Sprintf("The unique identifier of the integration you intend to set your trigger up with. Currently supported are %s. `generic-trigger` should be used for integrations not explicitly supported.", oxfordCommaJoin(VALID_TRIGGER_INTEGRATIONS)), !isDataSource),
			ForceNew:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(VALID_TRIGGER_INTEGRATIONS, false)),
		},
		INSTRUCTIONS: {
			Type:        schema.TypeList,
			Required:    !isDataSource,
			Computed:    isDataSource,
			Description: "Instructions containing the action to perform when invoking the trigger. Currently supported flag actions are `turnFlagOn` and `turnFlagOff`. This must be passed as the key-value pair `{ kind = \"<flag_action>\" }`.",
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					KIND: {
						Type:             schema.TypeString,
						Required:         true,
						Description:      "The action to perform when triggering. Currently supported flag actions are `turnFlagOn` and `turnFlagOff`.",
						ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"turnFlagOn", "turnFlagOff"}, false)),
					},
				},
			},
		},
		TRIGGER_URL: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The unique URL used to invoke the trigger.",
			Sensitive:   true,
		},
		MAINTAINER_ID: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The ID of the member responsible for maintaining the flag trigger. If created via Terraform, this value will be the ID of the member associated with the API key used for your provider configuration.",
		},
		ENABLED: {
			Type:        schema.TypeBool,
			Required:    !isDataSource,
			Computed:    isDataSource,
			Description: "Whether the trigger is currently active or not.",
		},
	}
}

func flagTriggerRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := metaRaw.(*Client)
	integrationKey := d.Get(INTEGRATION_KEY).(string)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)
	flagKey := d.Get(FLAG_KEY).(string)

	var triggerId string
	if isDataSource {
		triggerId = d.Get(ID).(string)
	} else {
		triggerId = d.Id()
	}

	trigger, res, err := client.ld.FlagTriggersApi.GetTriggerWorkflowById(client.ctx, projectKey, flagKey, envKey, triggerId).Execute()
	// if the trigger does not exist it simply return an empty trigger object
	if (isStatusNotFound(res) || trigger.Id == nil) && !isDataSource {
		log.Printf("[WARN] failed to find %s trigger with ID %s, removing from state if present", integrationKey, triggerId)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find %s trigger with ID %s, removing from state if present", integrationKey, triggerId),
		})
		d.SetId("")
		return diags
	}
	if err != nil || trigger.Id == nil {
		return diag.Errorf("failed to get %s trigger with ID %s", integrationKey, triggerId)
	}

	if isDataSource {
		d.SetId(*trigger.Id)
	}
	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(ENV_KEY, envKey)
	_ = d.Set(FLAG_KEY, flagKey)
	_ = d.Set(INTEGRATION_KEY, *trigger.IntegrationKey)
	_ = d.Set(INSTRUCTIONS, trigger.Instructions)
	_ = d.Set(MAINTAINER_ID, trigger.MaintainerId)
	_ = d.Set(ENABLED, trigger.Enabled)
	// NOTE: we do not want to set the trigger url at any point past the create as it will always be obscured

	return diags
}

func instructionsFromResourceData(d *schema.ResourceData, method string) []map[string]interface{} {
	rawInstructions := d.Get(INSTRUCTIONS).([]interface{})
	var instructions []map[string]interface{}
	switch method {
	case "POST":
		for _, v := range rawInstructions {
			instructions = append(instructions, v.(map[string]interface{}))
		}
	case "PATCH":
		if d.HasChange(INSTRUCTIONS) {
			for _, v := range rawInstructions {
				oldInstruction := v.(map[string]interface{})
				value := oldInstruction[KIND]
				instructions = append(instructions, map[string]interface{}{
					KIND: "replaceTriggerActionInstructions",
					VALUE: []map[string]interface{}{{
						KIND: value,
					},
					}})
			}
		}
	}
	return instructions
}

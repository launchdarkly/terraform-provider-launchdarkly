package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

func baseAIToolSchema(isDataSource bool) map[string]*schema.Schema {
	schemaMap := map[string]*schema.Schema{
		PROJECT_KEY: {
			Type:             schema.TypeString,
			Required:         true,
			Description:      addForceNewDescription("The project key.", !isDataSource),
			ForceNew:         !isDataSource,
			ValidateDiagFunc: validateKey(),
		},
		KEY: {
			Type:             schema.TypeString,
			Required:         true,
			Description:      addForceNewDescription("The AI tool's unique key.", !isDataSource),
			ForceNew:         !isDataSource,
			ValidateDiagFunc: validateKey(),
		},
		DESCRIPTION: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "The AI tool's description.",
		},
		SCHEMA_JSON: {
			Type:             schema.TypeString,
			Required:         !isDataSource,
			Computed:         isDataSource,
			Description:      "A JSON string representing the JSON Schema for the tool's parameters.",
			ValidateDiagFunc: emptyValueIfDataSource(validateJsonStringDiagFunc(), isDataSource),
			DiffSuppressFunc: emptyValueIfDataSource(suppressEquivalentJsonDiffs, isDataSource),
		},
		CUSTOM_PARAMETERS: {
			Type:             schema.TypeString,
			Optional:         !isDataSource,
			Computed:         isDataSource,
			Description:      "A JSON string representing custom application-level metadata for the AI tool.",
			ValidateDiagFunc: emptyValueIfDataSource(validateJsonStringDiagFunc(), isDataSource),
			DiffSuppressFunc: emptyValueIfDataSource(suppressEquivalentJsonDiffs, isDataSource),
		},
		MAINTAINER_ID: {
			Type:          schema.TypeString,
			Optional:      !isDataSource,
			Computed:      true,
			Description:   "The member ID of the maintainer for this AI tool. Conflicts with `maintainer_team_key`.",
			ConflictsWith: []string{MAINTAINER_TEAM_KEY},
		},
		MAINTAINER_TEAM_KEY: {
			Type:          schema.TypeString,
			Optional:      !isDataSource,
			Computed:      true,
			Description:   "The team key of the maintainer team for this AI tool. Conflicts with `maintainer_id`.",
			ConflictsWith: []string{MAINTAINER_ID},
		},
		VERSION: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The version of the AI tool.",
		},
		CREATION_DATE: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The creation timestamp of the AI tool.",
		},
	}

	if isDataSource {
		schemaMap = removeInvalidFieldsForDataSource(schemaMap)
	}

	return schemaMap
}

func aiToolRead(ctx context.Context, d *schema.ResourceData, meta interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	toolKey := d.Get(KEY).(string)

	var tool *ldapi.AITool
	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		tool, res, err = client.ld.AIConfigsApi.GetAITool(client.ctx, projectKey, toolKey).Execute()
		return err
	})

	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[WARN] failed to find AI tool with key %q in project %q, removing from state if present", toolKey, projectKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find AI tool with key %q in project %q, removing from state if present", toolKey, projectKey),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get AI tool with key %q in project %q: %s", toolKey, projectKey, handleLdapiErr(err))
	}

	if isDataSource {
		d.SetId(fmt.Sprintf("%s/%s", projectKey, tool.GetKey()))
	}

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(KEY, tool.GetKey())

	description := ""
	if tool.Description != nil {
		description = *tool.Description
	}
	_ = d.Set(DESCRIPTION, description)

	schemaJSON, err := mapToJsonString(tool.GetSchema())
	if err != nil {
		return diag.Errorf("failed to serialize schema_json for AI tool %q: %s", toolKey, err)
	}
	_ = d.Set(SCHEMA_JSON, schemaJSON)

	customParamsJSON, err := mapToJsonString(tool.GetCustomParameters())
	if err != nil {
		return diag.Errorf("failed to serialize custom_parameters for AI tool %q: %s", toolKey, err)
	}
	_ = d.Set(CUSTOM_PARAMETERS, customParamsJSON)

	_ = d.Set(VERSION, tool.GetVersion())
	_ = d.Set(CREATION_DATE, tool.GetCreatedAt())

	// Handle maintainer union type
	maintainer := tool.GetMaintainer()
	if maintainer.MaintainerMember != nil {
		_ = d.Set(MAINTAINER_ID, maintainer.MaintainerMember.GetId())
	}
	if maintainer.AiConfigsMaintainerTeam != nil {
		_ = d.Set(MAINTAINER_TEAM_KEY, maintainer.AiConfigsMaintainerTeam.GetKey())
	}

	return diags
}

func aiToolIdToKeys(id string) (projectKey, toolKey string, err error) {
	parts := splitID(id, 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("import ID must be in the format project_key/tool_key, got: %q", id)
	}
	return parts[0], parts[1], nil
}

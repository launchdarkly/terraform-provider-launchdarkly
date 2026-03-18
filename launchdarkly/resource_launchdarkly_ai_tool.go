package launchdarkly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

func resourceAITool() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAIToolCreate,
		ReadContext:   resourceAIToolRead,
		UpdateContext: resourceAIToolUpdate,
		DeleteContext: resourceAIToolDelete,
		Exists:        resourceAIToolExists,

		Importer: &schema.ResourceImporter{
			StateContext: resourceAIToolImport,
		},

		Description: `Provides a LaunchDarkly AI tool resource.

This resource allows you to create and manage AI tools within your LaunchDarkly project.`,

		Schema: baseAIToolSchema(false),
	}
}

func resourceAIToolCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	toolKey := d.Get(KEY).(string)

	schemaMap, err := jsonStringToMap(d.Get(SCHEMA_JSON).(string))
	if err != nil {
		return diag.Errorf("failed to parse schema_json: %s", err)
	}

	post := ldapi.NewAIToolPost(toolKey, schemaMap)

	if description, ok := d.GetOk(DESCRIPTION); ok {
		post.Description = ldapi.PtrString(description.(string))
	}

	if customParams, ok := d.GetOk(CUSTOM_PARAMETERS); ok {
		customParamsMap, err := jsonStringToMap(customParams.(string))
		if err != nil {
			return diag.Errorf("failed to parse custom_parameters: %s", err)
		}
		post.CustomParameters = customParamsMap
	}

	if maintainerId, ok := d.GetOk(MAINTAINER_ID); ok {
		post.MaintainerId = ldapi.PtrString(maintainerId.(string))
	}

	if maintainerTeamKey, ok := d.GetOk(MAINTAINER_TEAM_KEY); ok {
		post.MaintainerTeamKey = ldapi.PtrString(maintainerTeamKey.(string))
	}

	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.AIConfigsApi.PostAITool(client.ctx, projectKey).AIToolPost(*post).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to create AI tool with key %q in project %q: %s", toolKey, projectKey, handleLdapiErr(err))
	}

	d.SetId(fmt.Sprintf("%s/%s", projectKey, toolKey))

	return resourceAIToolRead(ctx, d, metaRaw)
}

func resourceAIToolRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return aiToolRead(ctx, d, metaRaw, false)
}

func resourceAIToolUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	toolKey := d.Get(KEY).(string)

	patch := ldapi.NewAIToolPatch()

	if d.HasChange(DESCRIPTION) {
		description := d.Get(DESCRIPTION).(string)
		patch.Description = &description
	}

	if d.HasChange(SCHEMA_JSON) {
		schemaMap, err := jsonStringToMap(d.Get(SCHEMA_JSON).(string))
		if err != nil {
			return diag.Errorf("failed to parse schema_json: %s", err)
		}
		patch.Schema = schemaMap
	}

	if d.HasChange(CUSTOM_PARAMETERS) {
		customParamsStr := d.Get(CUSTOM_PARAMETERS).(string)
		customParamsMap, err := jsonStringToMap(customParamsStr)
		if err != nil {
			return diag.Errorf("failed to parse custom_parameters: %s", err)
		}
		patch.CustomParameters = customParamsMap
	}

	if d.HasChange(MAINTAINER_ID) {
		if maintainerId, ok := d.GetOk(MAINTAINER_ID); ok {
			patch.MaintainerId = ldapi.PtrString(maintainerId.(string))
		}
	}

	if d.HasChange(MAINTAINER_TEAM_KEY) {
		if maintainerTeamKey, ok := d.GetOk(MAINTAINER_TEAM_KEY); ok {
			patch.MaintainerTeamKey = ldapi.PtrString(maintainerTeamKey.(string))
		}
	}

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.AIConfigsApi.PatchAITool(client.ctx, projectKey, toolKey).AIToolPatch(*patch).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to update AI tool with key %q in project %q: %s", toolKey, projectKey, handleLdapiErr(err))
	}

	return resourceAIToolRead(ctx, d, metaRaw)
}

func resourceAIToolDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	toolKey := d.Get(KEY).(string)

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, err = client.ld.AIConfigsApi.DeleteAITool(client.ctx, projectKey, toolKey).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to delete AI tool with key %q in project %q: %s", toolKey, projectKey, handleLdapiErr(err))
	}

	return diags
}

func resourceAIToolExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	toolKey := d.Get(KEY).(string)

	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, res, err = client.ld.AIConfigsApi.GetAITool(client.ctx, projectKey, toolKey).Execute()
		return err
	})
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get AI tool with key %q in project %q: %s", toolKey, projectKey, handleLdapiErr(err))
	}

	return true, nil
}

func resourceAIToolImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()
	if id == "" {
		return nil, fmt.Errorf("import ID cannot be empty")
	}

	projectKey, toolKey, err := aiToolIdToKeys(id)
	if err != nil {
		return nil, err
	}

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(KEY, toolKey)

	return []*schema.ResourceData{d}, nil
}

package launchdarkly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceFlagTemplates() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFlagTemplatesCreate,
		ReadContext:   resourceFlagTemplatesRead,
		UpdateContext: resourceFlagTemplatesUpdate,
		DeleteContext: resourceFlagTemplatesDelete,
		Exists:        resourceFlagTemplatesExists,

		Importer: &schema.ResourceImporter{
			StateContext: resourceFlagTemplatesImport,
		},

		Description: `Provides a LaunchDarkly flag templates resource.

This resource allows you to manage the "Custom" flag template settings applied to new feature flags created within a LaunchDarkly project. LaunchDarkly projects include several built-in flag templates (Release, Kill switch, Experiment, Custom, Migration); this resource manages the Custom template only.

-> **Note:** Flag templates are a singleton per project. Destroying this resource only removes it from Terraform state; the flag templates will continue to exist in LaunchDarkly.`,

		Schema: baseFlagTemplatesSchema(false),
	}
}

func resourceFlagTemplatesCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)

	// Read current CSA from the API so we pass it through unchanged.
	// CSA is managed by the launchdarkly_project resource, not this one.
	csa, err := getCurrentCSA(client, projectKey)
	if err != nil {
		return diag.Errorf("failed to read current client-side availability for project %q: %s", projectKey, handleLdapiErr(err))
	}

	payload := flagTemplatesPayloadFromResourceData(d, *csa)
	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.ProjectsApi.PutFlagDefaultsByProject(client.ctx, projectKey).UpsertFlagDefaultsPayload(payload).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to create flag templates for project %q: %s", projectKey, handleLdapiErr(err))
	}

	d.SetId(projectKey)
	return resourceFlagTemplatesRead(ctx, d, metaRaw)
}

func resourceFlagTemplatesRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return flagTemplatesRead(ctx, d, metaRaw, false)
}

func resourceFlagTemplatesUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)

	// Read current CSA from the API so we pass it through unchanged.
	csa, err := getCurrentCSA(client, projectKey)
	if err != nil {
		return diag.Errorf("failed to read current client-side availability for project %q: %s", projectKey, handleLdapiErr(err))
	}

	payload := flagTemplatesPayloadFromResourceData(d, *csa)
	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.ProjectsApi.PutFlagDefaultsByProject(client.ctx, projectKey).UpsertFlagDefaultsPayload(payload).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to update flag templates for project %q: %s", projectKey, handleLdapiErr(err))
	}

	return resourceFlagTemplatesRead(ctx, d, metaRaw)
}

func resourceFlagTemplatesDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	// Flag templates always exist for a project and cannot be truly deleted.
	// On destroy, we simply remove the resource from Terraform state.
	d.SetId("")
	return diag.Diagnostics{}
}

func resourceFlagTemplatesExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	projectKey := d.Id()

	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, res, err = client.ld.ProjectsApi.GetFlagDefaultsByProject(client.ctx, projectKey).Execute()
		return err
	})
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get flag templates for project %q: %s", projectKey, handleLdapiErr(err))
	}
	return true, nil
}

func resourceFlagTemplatesImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	projectKey := d.Id()
	_ = d.Set(PROJECT_KEY, projectKey)
	return []*schema.ResourceData{d}, nil
}

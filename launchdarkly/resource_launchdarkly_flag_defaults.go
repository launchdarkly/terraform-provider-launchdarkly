package launchdarkly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceFlagDefaults() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFlagDefaultsCreate,
		ReadContext:   resourceFlagDefaultsRead,
		UpdateContext: resourceFlagDefaultsUpdate,
		DeleteContext: resourceFlagDefaultsDelete,
		Exists:        resourceFlagDefaultsExists,

		Importer: &schema.ResourceImporter{
			StateContext: resourceFlagDefaultsImport,
		},

		Description: `Provides a LaunchDarkly flag defaults resource.

This resource allows you to manage the default settings applied to new feature flags created within a LaunchDarkly project.

-> **Note:** Flag defaults are a singleton per project. Destroying this resource only removes it from Terraform state; the flag defaults will continue to exist in LaunchDarkly.`,

		Schema: baseFlagDefaultsSchema(false),
	}
}

func resourceFlagDefaultsCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)

	// Read current CSA from the API so we pass it through unchanged.
	// CSA is managed by the launchdarkly_project resource, not this one.
	csa, err := getCurrentCSA(client, projectKey)
	if err != nil {
		return diag.Errorf("failed to read current client-side availability for project %q: %s", projectKey, handleLdapiErr(err))
	}

	payload := flagDefaultsPayloadFromResourceData(d, *csa)
	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.ProjectsApi.PutFlagDefaultsByProject(client.ctx, projectKey).UpsertFlagDefaultsPayload(payload).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to create flag defaults for project %q: %s", projectKey, handleLdapiErr(err))
	}

	d.SetId(projectKey)
	return resourceFlagDefaultsRead(ctx, d, metaRaw)
}

func resourceFlagDefaultsRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return flagDefaultsRead(ctx, d, metaRaw, false)
}

func resourceFlagDefaultsUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)

	// Read current CSA from the API so we pass it through unchanged.
	csa, err := getCurrentCSA(client, projectKey)
	if err != nil {
		return diag.Errorf("failed to read current client-side availability for project %q: %s", projectKey, handleLdapiErr(err))
	}

	payload := flagDefaultsPayloadFromResourceData(d, *csa)
	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.ProjectsApi.PutFlagDefaultsByProject(client.ctx, projectKey).UpsertFlagDefaultsPayload(payload).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to update flag defaults for project %q: %s", projectKey, handleLdapiErr(err))
	}

	return resourceFlagDefaultsRead(ctx, d, metaRaw)
}

func resourceFlagDefaultsDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	// Flag defaults always exist for a project and cannot be truly deleted.
	// On destroy, we simply remove the resource from Terraform state.
	d.SetId("")
	return diag.Diagnostics{}
}

func resourceFlagDefaultsExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
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
		return false, fmt.Errorf("failed to get flag defaults for project %q: %s", projectKey, handleLdapiErr(err))
	}
	return true, nil
}

func resourceFlagDefaultsImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	projectKey := d.Id()
	_ = d.Set(PROJECT_KEY, projectKey)
	return []*schema.ResourceData{d}, nil
}

package launchdarkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func resourceEnvironment() *schema.Resource {
	envSchema := environmentSchema(environmentSchemaOptions{forProject: false, isDataSource: false})
	envSchema[PROJECT_KEY] = &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		Description:      addForceNewDescription("The LaunchDarkly project key.", true),
		ForceNew:         true,
		ValidateDiagFunc: validateKey(),
	}

	return &schema.Resource{
		CreateContext: resourceEnvironmentCreate,
		ReadContext:   resourceEnvironmentRead,
		UpdateContext: resourceEnvironmentUpdate,
		DeleteContext: resourceEnvironmentDelete,
		Exists:        resourceEnvironmentExists,

		Importer: &schema.ResourceImporter{
			State: resourceEnvironmentImport,
		},
		Schema: envSchema,

		Description: "Provides a LaunchDarkly environment resource.\n\nThis resource allows you to create and manage environments in your LaunchDarkly organization. This resource should _not_ be used if the encapsulated project is also managed via Terraform. In this case, you should _always_ use the nested environments config blocks on your `launchdarkly_project` resource to manage your environments.\n\n-> **Note:** Mixing the use of nested `environments` blocks in the [`launchdarkly_project`] resource and `launchdarkly_environment` resources is not recommended.",
	}
}

func resourceEnvironmentCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)
	name := d.Get(NAME).(string)
	color := d.Get(COLOR).(string)
	defaultTTL := int32(d.Get(DEFAULT_TTL).(int))
	secureMode := d.Get(SECURE_MODE).(bool)
	defaultTrackEvents := d.Get(DEFAULT_TRACK_EVENTS).(bool)
	tags := stringsFromSchemaSet(d.Get(TAGS).(*schema.Set))
	requireComments := d.Get(REQUIRE_COMMENTS).(bool)
	confirmChanges := d.Get(CONFIRM_CHANGES).(bool)

	envPost := ldapi.EnvironmentPost{
		Name:               name,
		Key:                key,
		Color:              color,
		DefaultTtl:         &defaultTTL,
		SecureMode:         &secureMode,
		DefaultTrackEvents: &defaultTrackEvents,
		Tags:               tags,
		RequireComments:    &requireComments,
		ConfirmChanges:     &confirmChanges,
	}

	_, _, err := client.ld.EnvironmentsApi.PostEnvironment(client.ctx, projectKey).EnvironmentPost(envPost).Execute()
	if err != nil {
		return diag.Errorf("failed to create environment: [%+v] for project key: %s: %s", envPost, projectKey, handleLdapiErr(err))
	}

	approvalSettings := d.Get(APPROVAL_SETTINGS)
	if len(approvalSettings.([]interface{})) > 0 {
		updateDiags := resourceEnvironmentUpdate(ctx, d, metaRaw)
		if updateDiags.HasError() {
			// if there was a problem in the update state, we need to clean up completely by deleting the env
			_, deleteErr := client.ld.EnvironmentsApi.DeleteEnvironment(client.ctx, projectKey, key).Execute()
			// TODO: Figure out if we can get the err out of updateDiag (not looking likely) to use in hanldeLdapiErr
			if deleteErr != nil {
				return updateDiags
				// return diag.Errorf("failed to clean up environment %q from project %q: %s", key, projectKey, handleLdapiErr(errs))
			}
			return diag.Errorf("failed to update environment with name %q key %q for projectKey %q: %s",
				name, key, projectKey, handleLdapiErr(err))
		}
	}

	d.SetId(projectKey + "/" + key)
	return resourceEnvironmentRead(ctx, d, metaRaw)
}

func resourceEnvironmentRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return environmentRead(ctx, d, metaRaw, false)
}

func resourceEnvironmentUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)

	//required fields
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)
	name := d.Get(NAME)
	color := d.Get(COLOR)
	requireComments := d.Get(REQUIRE_COMMENTS)
	confirmChanges := d.Get(CONFIRM_CHANGES)

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &name),
		patchReplace("/color", &color),
		patchReplace("/defaultTtl", d.Get(DEFAULT_TTL)),
		patchReplace("/secureMode", d.Get(SECURE_MODE)),
		patchReplace("/defaultTrackEvents", d.Get(DEFAULT_TRACK_EVENTS)),
		patchReplace("/requireComments", &requireComments),
		patchReplace("/confirmChanges", &confirmChanges),
	}

	if d.HasChange(TAGS) {
		tags := stringsFromResourceData(d, TAGS)
		patch = append(patch, patchReplace("/tags", &tags))
	}

	oldApprovalSettings, newApprovalSettings := d.GetChange(APPROVAL_SETTINGS)
	approvalPatch, err := approvalPatchFromSettings(oldApprovalSettings, newApprovalSettings)
	if err != nil {
		return diag.FromErr(err)
	}
	patch = append(patch, approvalPatch...)
	_, _, err = client.ld.EnvironmentsApi.PatchEnvironment(client.ctx, projectKey, key).PatchOperation(patch).Execute()
	if err != nil {
		return diag.Errorf("failed to update environment with key %q for project: %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return resourceEnvironmentRead(ctx, d, metaRaw)
}

func resourceEnvironmentDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	_, err := client.ld.EnvironmentsApi.DeleteEnvironment(client.ctx, projectKey, key).Execute()
	if err != nil {
		return diag.Errorf("failed to delete project with key %q for project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return diags
}

func resourceEnvironmentExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return environmentExists(d.Get(PROJECT_KEY).(string), d.Get(KEY).(string), metaRaw.(*Client))
}

func environmentExists(projectKey string, key string, meta *Client) (bool, error) {
	_, res, err := meta.ld.EnvironmentsApi.GetEnvironment(meta.ctx, projectKey, key).Execute()
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get environment with key %q for project %q: %v", key, projectKey, handleLdapiErr(err))
	}

	return true, nil
}

func environmentExistsInProject(project ldapi.Project, envKey string) bool {
	for _, env := range project.Environments.Items {
		if env.Key == envKey {
			return true
		}
	}
	return false
}

func resourceEnvironmentImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()

	if strings.Count(id, "/") != 1 {
		return nil, fmt.Errorf("found unexpected environment id format: %q expected format: 'project_key/env_key'", id)
	}

	parts := strings.SplitN(d.Id(), "/", 2)

	projectKey, envKey := parts[0], parts[1]

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(KEY, envKey)

	return []*schema.ResourceData{d}, nil
}

package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceView() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceViewCreate,
		ReadContext:   resourceViewRead,
		UpdateContext: resourceViewUpdate,
		DeleteContext: resourceViewDelete,
		Exists:        resourceViewExists,

		Importer: &schema.ResourceImporter{
			StateContext: resourceViewImport,
		},

		Description: `Provides a LaunchDarkly view resource.

This resource allows you to create and manage views within your LaunchDarkly project.`,

		Schema: map[string]*schema.Schema{
			PROJECT_KEY: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      addForceNewDescription("The project key.", true),
				ForceNew:         true,
				ValidateDiagFunc: validateKey(),
			},
			KEY: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      addForceNewDescription("The view's unique key.", true),
				ForceNew:         true,
				ValidateDiagFunc: validateKey(),
			},
			NAME: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The view's name.",
			},
			DESCRIPTION: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The view's description.",
			},
			GENERATE_SDK_KEYS: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to generate SDK keys for this view.",
			},
			MAINTAINER_ID: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The member ID of the maintainer for this view. Exactly one of `maintainer_id` and `maintainer_team_key` must be set.",
				ExactlyOneOf: []string{MAINTAINER_ID, MAINTAINER_TEAM_KEY},
			},
			MAINTAINER_TEAM_KEY: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The team key of the maintainer team for this view. Exactly one of `maintainer_id` and `maintainer_team_key` must be set.",
				ExactlyOneOf: []string{MAINTAINER_ID, MAINTAINER_TEAM_KEY},
			},
			TAGS: tagsSchema(tagsSchemaOptions{isDataSource: false}),
			ARCHIVED: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether the view is archived.",
			},
		},
	}
}

func resourceViewCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S)
	if err != nil {
		return diag.FromErr(err)
	}

	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(KEY).(string)

	viewPost := map[string]interface{}{
		"key":  viewKey,
		"name": d.Get(NAME).(string),
	}

	if description, ok := d.GetOk(DESCRIPTION); ok {
		viewPost["description"] = description.(string)
	}

	if generateSdkKeys, ok := d.GetOk(GENERATE_SDK_KEYS); ok {
		viewPost["generateSdkKeys"] = generateSdkKeys.(bool)
	}

	if maintainerId, ok := d.GetOk(MAINTAINER_ID); ok {
		viewPost["maintainerId"] = maintainerId.(string)
	}

	if maintainerTeamKey, ok := d.GetOk(MAINTAINER_TEAM_KEY); ok {
		viewPost["maintainerTeamKey"] = maintainerTeamKey.(string)
	}

	if tags, ok := d.GetOk(TAGS); ok {
		viewPost["tags"] = interfaceSliceToStringSlice(tags.(*schema.Set).List())
	}

	_, err = createView(betaClient, projectKey, viewPost)
	if err != nil {
		return diag.Errorf("failed to create view with key %q in project %q: %s", viewKey, projectKey, handleLdapiErr(err))
	}

	d.SetId(fmt.Sprintf("%s/%s", projectKey, viewKey))

	return resourceViewRead(ctx, d, metaRaw)
}

func resourceViewRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return viewRead(ctx, d, metaRaw, false)
}

func resourceViewUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S)
	if err != nil {
		return diag.FromErr(err)
	}

	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(KEY).(string)

	patch := make(map[string]interface{})

	if d.HasChange(NAME) {
		patch["name"] = d.Get(NAME).(string)
	}

	if d.HasChange(DESCRIPTION) {
		patch["description"] = d.Get(DESCRIPTION).(string)
	}

	if d.HasChange(GENERATE_SDK_KEYS) {
		patch["generateSdkKeys"] = d.Get(GENERATE_SDK_KEYS).(bool)
	}

	if d.HasChange(MAINTAINER_ID) {
		if maintainerId, ok := d.GetOk(MAINTAINER_ID); ok {
			patch["maintainerId"] = maintainerId.(string)
		}
		// Note: We don't set maintainerId to nil when removed, as the API doesn't accept null values
		// The field will be omitted from the patch, which is the correct behavior
	}

	if d.HasChange(MAINTAINER_TEAM_KEY) {
		if maintainerTeamKey, ok := d.GetOk(MAINTAINER_TEAM_KEY); ok {
			patch["maintainerTeamKey"] = maintainerTeamKey.(string)
		}
		// Note: We don't set maintainerTeamKey to nil when removed, as the API doesn't accept null values
		// The field will be omitted from the patch, which is the correct behavior
	}

	if d.HasChange(TAGS) {
		patch["tags"] = interfaceSliceToStringSlice(d.Get(TAGS).(*schema.Set).List())
	}

	if d.HasChange(ARCHIVED) {
		patch["archived"] = d.Get(ARCHIVED).(bool)
	}

	if len(patch) > 0 {
		err = patchView(betaClient, projectKey, viewKey, patch)
		if err != nil {
			return diag.Errorf("failed to update view with key %q in project %q: %s", viewKey, projectKey, handleLdapiErr(err))
		}
	}

	return resourceViewRead(ctx, d, metaRaw)
}

func resourceViewDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S)
	if err != nil {
		return diag.FromErr(err)
	}

	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(KEY).(string)

	err = deleteView(betaClient, projectKey, viewKey)
	if err != nil {
		return diag.Errorf("failed to delete view with key %q in project %q: %s", viewKey, projectKey, handleLdapiErr(err))
	}

	return diags
}

func resourceViewExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S)
	if err != nil {
		return false, err
	}
	return viewExists(d.Get(PROJECT_KEY).(string), d.Get(KEY).(string), betaClient)
}

func viewExists(projectKey, viewKey string, client *Client) (bool, error) {
	_, res, err := getViewRaw(client, projectKey, viewKey)
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get view with key %q in project %q: %s", viewKey, projectKey, handleLdapiErr(err))
	}

	return true, nil
}

func resourceViewImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()
	if id == "" {
		return nil, fmt.Errorf("import ID cannot be empty")
	}

	parts := splitID(id, 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("import ID must be in the format project_key/view_key")
	}

	projectKey, viewKey := parts[0], parts[1]
	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(KEY, viewKey)

	return []*schema.ResourceData{d}, nil
}

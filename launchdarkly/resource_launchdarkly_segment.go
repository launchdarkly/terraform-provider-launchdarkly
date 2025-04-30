package launchdarkly

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func resourceSegment() *schema.Resource {
	schemaMap := baseSegmentSchema(segmentSchemaOptions{isDataSource: false})
	schemaMap[PROJECT_KEY] = &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		ForceNew:         true,
		ValidateDiagFunc: validateKey(),
		Description:      addForceNewDescription("The segment's project key.", true),
	}
	schemaMap[ENV_KEY] = &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		ForceNew:         true,
		ValidateDiagFunc: validateKey(),
		Description:      addForceNewDescription("The segment's environment key.", true),
	}
	schemaMap[KEY] = &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		ForceNew:         true,
		ValidateDiagFunc: validateKey(),
		Description:      addForceNewDescription("The unique key that references the segment.", true),
	}
	schemaMap[NAME] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The human-friendly name for the segment.",
	}
	return &schema.Resource{
		CreateContext: resourceSegmentCreate,
		ReadContext:   resourceSegmentRead,
		UpdateContext: resourceSegmentUpdate,
		DeleteContext: resourceSegmentDelete,
		Exists:        resourceSegmentExists,

		Importer: &schema.ResourceImporter{
			State: resourceSegmentImport,
		},

		Schema: schemaMap,
		Description: `Provides a LaunchDarkly segment resource.

This resource allows you to create and manage segments within your LaunchDarkly organization.`,
	}
}

func resourceSegmentCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)

	key := d.Get(KEY).(string)
	description := d.Get(DESCRIPTION).(string)
	segmentName := d.Get(NAME).(string)
	tags := stringsFromResourceData(d, TAGS)
	unbounded := d.Get(UNBOUNDED).(bool)
	unboundedContextKind := d.Get(UNBOUNDED_CONTEXT_KIND).(string)

	segment := ldapi.SegmentBody{
		Name:                 segmentName,
		Key:                  key,
		Description:          &description,
		Tags:                 tags,
		Unbounded:            &unbounded,
		UnboundedContextKind: &unboundedContextKind,
	}

	_, _, err := client.ld.SegmentsApi.PostSegment(client.ctx, projectKey, envKey).SegmentBody(segment).Execute()
	if err != nil {
		return diag.Errorf("failed to create segment %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	// ld's api does not allow some fields to be passed in during segment creation so we do an update:
	// https://apidocs.launchdarkly.com/reference#create-segment
	updateDiags := resourceSegmentUpdate(ctx, d, metaRaw)
	if updateDiags.HasError() {
		// TODO: Figure out if we can get the err out of updateDiag (not looking likely) to use in handleLdapiErr
		return updateDiags
		// return diag.Errorf("failed to update segment with name %q key %q for projectKey %q: %s",
		// 	segmentName, key, projectKey, handleLdapiErr(errs))
	}

	d.SetId(projectKey + "/" + envKey + "/" + key)
	return resourceSegmentRead(ctx, d, metaRaw)
}

func resourceSegmentRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return segmentRead(ctx, d, metaRaw, false)
}

func resourceSegmentUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	key := d.Get(KEY).(string)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)
	description := d.Get(DESCRIPTION).(string)
	name := d.Get(NAME).(string)
	tags := stringsFromResourceData(d, TAGS)
	included := d.Get(INCLUDED).([]interface{})
	excluded := d.Get(EXCLUDED).([]interface{})
	includedContexts := segmentTargetsFromResourceData(d, segmentTargetOptions{Included: true})
	excludedContexts := segmentTargetsFromResourceData(d, segmentTargetOptions{Excluded: true})
	rules, err := segmentRulesFromResourceData(d, RULES)
	if err != nil {
		return diag.FromErr(err)
	}
	comment := "Terraform"
	patchOps := []ldapi.PatchOperation{
		patchReplace("/name", name),
		patchReplace("/description", description),
		patchReplace("/temporary", TEMPORARY),
		patchReplace("/included", included),
		patchReplace("/excluded", excluded),
		patchReplace("/rules", rules),
		patchReplace("/includedContexts", includedContexts),
		patchReplace("/excludedContexts", excludedContexts),
	}

	tagPatch := patchReplace("/tags", tags)
	if d.HasChange(TAGS) && len(tags) == 0 {
		tagPatch = patchRemove("/tags")
	}
	patchOps = append(patchOps, tagPatch)

	_, _, err = client.ld.SegmentsApi.PatchSegment(client.ctx, projectKey, envKey, key).PatchWithComment(ldapi.PatchWithComment{
		Comment: &comment,
		Patch:   patchOps}).Execute()
	if err != nil {
		return diag.Errorf("failed to update segment %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return resourceSegmentRead(ctx, d, metaRaw)
}

func resourceSegmentDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)
	key := d.Get(KEY).(string)

	_, err := client.ld.SegmentsApi.DeleteSegment(client.ctx, projectKey, envKey, key).Execute()
	if err != nil {
		return diag.Errorf("failed to delete segment %q from project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return diags
}

func resourceSegmentExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)
	key := d.Get(KEY).(string)

	_, res, err := client.ld.SegmentsApi.GetSegment(client.ctx, projectKey, envKey, key).Execute()
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check if segment %q exists in project %q: %s",
			key, projectKey, handleLdapiErr(err))
	}
	return true, nil
}

func resourceSegmentImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()

	if strings.Count(id, "/") != 2 {
		return nil, fmt.Errorf("found unexpected segment id format: %q expected format: 'project_key/env_key/segment_key'", id)
	}

	parts := strings.SplitN(d.Id(), "/", 3)

	projectKey, envKey, segmentKey := parts[0], parts[1], parts[2]

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(ENV_KEY, envKey)
	_ = d.Set(KEY, segmentKey)

	return []*schema.ResourceData{d}, nil
}

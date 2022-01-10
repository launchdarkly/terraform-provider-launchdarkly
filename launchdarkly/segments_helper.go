package launchdarkly

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func baseSegmentSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		DESCRIPTION: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The description of the segment's purpose.",
		},
		TAGS: tagsSchema(),
		INCLUDED: {
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Description: "List of user keys included in the segment.",
		},
		EXCLUDED: {
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Description: "List of user keys excluded from the segment",
		},
		CREATION_DATE: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The segment's creation date represented as a UNIX epoch timestamp.",
		},
		RULES: segmentRulesSchema(),
	}
}

func segmentRead(ctx context.Context, d *schema.ResourceData, raw interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := raw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)
	segmentKey := d.Get(KEY).(string)

	segment, res, err := client.ld.SegmentsApi.GetSegment(client.ctx, projectKey, envKey, segmentKey).Execute()
	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[WARN] failed to find segment %q in project %q, environment %q, removing from state", segmentKey, projectKey, envKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find segment %q in project %q, environment %q, removing from state", segmentKey, projectKey, envKey),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get segment %q of project %q: %s", segmentKey, projectKey, handleLdapiErr(err))
	}

	if isDataSource {
		d.SetId(projectKey + "/" + envKey + "/" + segmentKey)
	}
	_ = d.Set(NAME, segment.Name)
	_ = d.Set(DESCRIPTION, segment.Description)
	_ = d.Set(CREATION_DATE, segment.CreationDate)

	err = d.Set(TAGS, segment.Tags)
	if err != nil {
		return diag.Errorf("failed to set tags on segment with key %q: %v", segmentKey, err)
	}

	err = d.Set(INCLUDED, segment.Included)
	if err != nil {
		return diag.Errorf("failed to set included on segment with key %q: %v", segmentKey, err)
	}

	err = d.Set(EXCLUDED, segment.Excluded)
	if err != nil {
		return diag.Errorf("failed to set excluded on segment with key %q: %v", segmentKey, err)
	}

	rules, err := segmentRulesToResourceData(segment.Rules)
	if err != nil {
		return diag.Errorf("failed to read rules on segment with key %q: %v", segmentKey, err)
	}
	err = d.Set(RULES, rules)
	if err != nil {
		return diag.Errorf("failed to set excluded on segment with key %q: %v", segmentKey, err)
	}
	return diags
}

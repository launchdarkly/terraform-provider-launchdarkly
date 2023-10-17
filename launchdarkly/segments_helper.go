package launchdarkly

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v12"
)

type segmentSchemaOptions struct {
	isDataSource bool
}

func baseSegmentSchema(options segmentSchemaOptions) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		DESCRIPTION: {
			Type:        schema.TypeString,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "The description of the segment's purpose.",
		},
		TAGS: tagsSchema(tagsSchemaOptions(options)),
		INCLUDED: {
			Type:          schema.TypeList,
			Elem:          &schema.Schema{Type: schema.TypeString},
			Optional:      !options.isDataSource,
			Computed:      options.isDataSource,
			Description:   "List of user keys included in the segment. To target on other context kinds, use the included_contexts block attribute. This attribute is not valid when `unbounded` is set to `true`.",
			ConflictsWith: []string{UNBOUNDED},
		},
		EXCLUDED: {
			Type:          schema.TypeList,
			Elem:          &schema.Schema{Type: schema.TypeString},
			Optional:      !options.isDataSource,
			Computed:      options.isDataSource,
			Description:   "List of user keys excluded from the segment. To target on other context kinds, use the excluded_contexts block attribute. This attribute is not valid when `unbounded` is set to `true`.",
			ConflictsWith: []string{UNBOUNDED},
		},
		INCLUDED_CONTEXTS: {
			Type:          schema.TypeList,
			Elem:          &schema.Resource{Schema: segmentTargetsSchema()},
			Optional:      !options.isDataSource,
			Computed:      options.isDataSource,
			Description:   "List of non-user target objects included in the segment. This attribute is not valid when `unbounded` is set to `true`.",
			ConflictsWith: []string{UNBOUNDED},
		},
		EXCLUDED_CONTEXTS: {
			Type:          schema.TypeList,
			Elem:          &schema.Resource{Schema: segmentTargetsSchema()},
			Optional:      !options.isDataSource,
			Computed:      options.isDataSource,
			Description:   "List of non-user target objects excluded from the segment. This attribute is not valid when `unbounded` is set to `true`.",
			ConflictsWith: []string{UNBOUNDED},
		},
		CREATION_DATE: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The segment's creation date represented as a UNIX epoch timestamp.",
		},
		RULES: segmentRulesSchema(segmentRulesSchemaOptions(options)),
		UNBOUNDED: {
			Type:          schema.TypeBool,
			Required:      false,
			Optional:      !options.isDataSource,
			Computed:      options.isDataSource,
			Default:       false,
			Description:   "Whether to create a standard segment (`false`) or a Big Segment (`true`). Standard segments include rule-based and smaller list-based segments. Big Segments include larger list-based segments and synced segments. Only use a Big Segment if you need to add more than 15,000 individual targets. It is not possible to manage the list of targeted contexts for Big Segments with Terraform.",
			ForceNew:      !options.isDataSource,
			ConflictsWith: []string{INCLUDED, EXCLUDED, INCLUDED_CONTEXTS, EXCLUDED_CONTEXTS, RULES},
		},
		UNBOUNDED_CONTEXT_KIND: {
			Type:          schema.TypeString,
			Computed:      true,
			Optional:      !options.isDataSource,
			Description:   "For Big Segments, the targeted context kind. If this attribute is not specified it will default to `user`.",
			ForceNew:      !options.isDataSource,
			ConflictsWith: []string{INCLUDED, EXCLUDED, INCLUDED_CONTEXTS, EXCLUDED_CONTEXTS, RULES},
		},
	}
}

func segmentTargetsSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		VALUES: {
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Required:    true,
			Description: "List of target object keys included in or excluded from the segment.",
		},
		CONTEXT_KIND: {
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringNotInSlice([]string{"user"}, true)),
			Description:      "The context kind associated with this segment target. To target on user contexts, use the included and excluded attributes.",
		},
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

	err = d.Set(UNBOUNDED, segment.Unbounded)
	if err != nil {
		return diag.Errorf("failed to set unbounded on segment with key %q: %v", segmentKey, err)
	}

	err = d.Set(UNBOUNDED_CONTEXT_KIND, segment.UnboundedContextKind)
	if err != nil {
		return diag.Errorf("failed to set unboundedContextKind on segment with key %q: %v", segmentKey, err)
	}

	err = d.Set(INCLUDED, segment.Included)
	if err != nil {
		return diag.Errorf("failed to set included on segment with key %q: %v", segmentKey, err)
	}

	err = d.Set(EXCLUDED, segment.Excluded)
	if err != nil {
		return diag.Errorf("failed to set excluded on segment with key %q: %v", segmentKey, err)
	}

	err = d.Set(INCLUDED_CONTEXTS, segmentTargetsToResourceData(segment.IncludedContexts))
	if err != nil {
		return diag.Errorf("failed to set included_contexts on segment with key %q: %v", segmentKey, err)
	}

	err = d.Set(EXCLUDED_CONTEXTS, segmentTargetsToResourceData(segment.ExcludedContexts))
	if err != nil {
		return diag.Errorf("failed to set excluded_contexts on segment with key %q: %v", segmentKey, err)
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

type segmentTargetOptions struct {
	Included bool
	Excluded bool
}

func segmentTargetsFromResourceData(d *schema.ResourceData, options segmentTargetOptions) []ldapi.SegmentTarget {
	var schemaTargets []interface{}
	if options.Included {
		schemaTargets = d.Get(INCLUDED_CONTEXTS).([]interface{})
	} else if options.Excluded {
		schemaTargets = d.Get(EXCLUDED_CONTEXTS).([]interface{})
	}
	targets := make([]ldapi.SegmentTarget, len(schemaTargets))
	for _, t := range schemaTargets {
		target := segmentTargetFromResourceData(t)
		targets = append(targets, target)
	}
	return targets
}

func segmentTargetFromResourceData(val interface{}) ldapi.SegmentTarget {
	targetMap := val.(map[string]interface{})
	var values []string
	for _, t := range targetMap[VALUES].([]interface{}) {
		values = append(values, t.(string))
	}
	contextKind := targetMap[CONTEXT_KIND].(string)
	return ldapi.SegmentTarget{
		Values:      values,
		ContextKind: &contextKind,
	}
}

func segmentTargetsToResourceData(targets []ldapi.SegmentTarget) interface{} {
	transformed := make([]interface{}, 0, len(targets))

	for _, t := range targets {
		target := map[string]interface{}{
			VALUES:       t.Values,
			CONTEXT_KIND: *t.ContextKind,
		}
		transformed = append(transformed, target)
	}
	return transformed
}

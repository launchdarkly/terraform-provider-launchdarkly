package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v17"
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
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "List of user keys included in the segment. To target on other context kinds, use the included_contexts block attribute. This attribute is not valid when `unbounded` is set to `true`.",
		},
		EXCLUDED: {
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "List of user keys excluded from the segment. To target on other context kinds, use the excluded_contexts block attribute. This attribute is not valid when `unbounded` is set to `true`.",
		},
		INCLUDED_CONTEXTS: {
			Type:        schema.TypeList,
			Elem:        &schema.Resource{Schema: segmentTargetsSchema()},
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "List of non-user target objects included in the segment. This attribute is not valid when `unbounded` is set to `true`.",
		},
		EXCLUDED_CONTEXTS: {
			Type:        schema.TypeList,
			Elem:        &schema.Resource{Schema: segmentTargetsSchema()},
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "List of non-user target objects excluded from the segment. This attribute is not valid when `unbounded` is set to `true`.",
		},
		CREATION_DATE: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The segment's creation date represented as a UNIX epoch timestamp.",
		},
		RULES: segmentRulesSchema(segmentRulesSchemaOptions(options)),
		UNBOUNDED: {
			Type:        schema.TypeBool,
			Required:    false,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Default:     false,
			Description: addForceNewDescription("Whether to create a standard segment (`false`) or a Big Segment (`true`). Standard segments include rule-based and smaller list-based segments. Big Segments include larger list-based segments and synced segments. Only use a Big Segment if you need to add more than 15,000 individual targets. It is not possible to manage the list of targeted contexts for Big Segments with Terraform.", !options.isDataSource),
			ForceNew:    !options.isDataSource,
		},
		UNBOUNDED_CONTEXT_KIND: {
			Type:          schema.TypeString,
			Computed:      true,
			Optional:      !options.isDataSource,
			Description:   addForceNewDescription("For Big Segments, the targeted context kind. If this attribute is not specified it will default to `user`.", !options.isDataSource),
			ForceNew:      !options.isDataSource,
			ConflictsWith: []string{INCLUDED, EXCLUDED, INCLUDED_CONTEXTS, EXCLUDED_CONTEXTS, RULES},
		},
		VIEW_KEYS: {
			Type:     schema.TypeSet,
			Optional: !options.isDataSource,
			Computed: true, // Always computed to support import and drift detection
			Elem: &schema.Schema{
				Type:             schema.TypeString,
				ValidateDiagFunc: validateKey(),
			},
			Description: "A set of view keys to link this segment to. This is an alternative to using the `launchdarkly_view_links` resource for managing view associations. When set, this segment will be linked to the specified views. Note: Using both `view_keys` on the segment and `launchdarkly_view_links` to manage the same segment may cause conflicts.",
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

	var segment *ldapi.UserSegment
	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		segment, res, err = client.ld.SegmentsApi.GetSegment(client.ctx, projectKey, envKey, segmentKey).Execute()
		return err
	})
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

	// Fetch and set view associations
	// Always populate view_keys from the API (Optional+Computed behavior)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		// Log warning but don't fail the read for discovery data
		log.Printf("[WARN] failed to create beta client for segment %q in project %q, environment %q: %v", segmentKey, projectKey, envKey, err)
	} else {
		// Get the environment to retrieve its ID
		var env *ldapi.Environment
		err = client.withConcurrency(client.ctx, func() error {
			env, _, err = client.ld.EnvironmentsApi.GetEnvironment(client.ctx, projectKey, envKey).Execute()
			return err
		})
		if err != nil {
			log.Printf("[WARN] failed to get environment %q in project %q: %v", envKey, projectKey, err)
		} else {
			viewKeys, err := getViewsContainingSegment(betaClient, projectKey, env.Id, segmentKey)
			if err != nil {
				// Log warning but don't fail the read for discovery data
				log.Printf("[WARN] failed to get views for segment %q in project %q, environment %q: %v", segmentKey, projectKey, envKey, err)
			} else {
				// Set view_keys to the actual view associations
				err = d.Set(VIEW_KEYS, viewKeys)
				if err != nil {
					return diag.Errorf("could not set view_keys on segment with key %q: %v", segmentKey, err)
				}

				// For data sources, also set the legacy VIEWS field for backwards compatibility
				if isDataSource {
					err = d.Set(VIEWS, viewKeys)
					if err != nil {
						return diag.Errorf("could not set views on segment with key %q: %v", segmentKey, err)
					}
				}
			}
		}
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

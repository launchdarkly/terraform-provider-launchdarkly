package launchdarkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	ldapi "github.com/launchdarkly/api-client-go/v16"
)

func baseMetricSchema(isDataSource bool) map[string]*schema.Schema {
	schemaMap := map[string]*schema.Schema{
		PROJECT_KEY: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			Description:      addForceNewDescription("The metrics's project key. A change in this field will force the destruction of the existing resource and the creation of a new one.", !isDataSource),
			ValidateDiagFunc: validateKey(),
		},
		KEY: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validateKey(),
			Description:      addForceNewDescription("The unique key that references the metric. A change in this field will force the destruction of the existing resource and the creation of a new one.", !isDataSource),
		},
		NAME: {
			Type:        schema.TypeString,
			Required:    !isDataSource,
			Computed:    isDataSource,
			Description: "The human-friendly name for the metric.",
		},
		KIND: {
			Type:             schema.TypeString,
			Required:         !isDataSource,
			Computed:         isDataSource,
			Description:      addForceNewDescription("The metric type. Available choices are `click`, `custom`, and `pageview`.", !isDataSource),
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"pageview", "click", "custom"}, false)),
			ForceNew:         true,
		},
		MAINTAINER_ID: {
			Type:             schema.TypeString,
			Optional:         !isDataSource,
			Computed:         true,
			Description:      "The LaunchDarkly member ID of the member who will maintain the metric. If not set, the API will automatically apply the member associated with your Terraform API key or the most recently-set maintainer",
			ValidateDiagFunc: validateID(),
		},
		DESCRIPTION: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "The description of the metric's purpose.",
		},
		TAGS: tagsSchema(tagsSchemaOptions{isDataSource: isDataSource}),
		IS_ACTIVE: {
			Type:        schema.TypeBool,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "Ignored. All metrics are considered active.",
			Default:     false,
			Deprecated:  "No longer in use. This field will be removed in a future major release of the LaunchDarkly provider.",
		},
		IS_NUMERIC: {
			Type:        schema.TypeBool,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "Whether a `custom` metric is a numeric metric or not.",
			Default:     false,
		},
		UNIT: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "(Required for kind `custom`) The unit for numeric `custom` metrics.",
		},
		SELECTOR: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "The CSS selector for your metric (if click metric)",
		},
		EVENT_KEY: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "The event key for your metric (if custom metric)",
		},
		SUCCESS_CRITERIA: {
			Type:             schema.TypeString,
			Optional:         !isDataSource,
			Description:      "The success criteria for your metric (if numeric metric)",
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"HigherThanBaseline", "LowerThanBaseline"}, false)),
			Computed:         true,
			ComputedWhen:     []string{KIND},
		},
		URLS: {
			Type:        schema.TypeList,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "List of nested `url` blocks describing URLs that you want to associate with the metric.",
			Elem: &schema.Resource{
				Schema: metricUrlSchema(),
			},
		},
		RANDOMIZATION_UNITS: {
			Type: schema.TypeSet,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Optional:    !isDataSource,
			Computed:    true,
			Description: `A set of one or more context kinds that this metric can measure events from. Metrics can only use context kinds marked as "Available for experiments." For more information, read [Allocating experiment audiences](https://docs.launchdarkly.com/home/creating-experiments/allocation).`,
		},
		INCLUDE_UNITS_WITHOUT_EVENTS: {
			Type:        schema.TypeBool,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "Include units that did not send any events and set their value to 0.",
			Default:     true,
		},
		UNIT_AGGREGATION_TYPE: {
			Type:             schema.TypeString,
			Optional:         !isDataSource,
			Computed:         isDataSource,
			Description:      "The method by which multiple unit event values are aggregated.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"average", "sum"}, false)),
			Default:          "average",
		},
		ANALYSIS_TYPE: {
			Type:             schema.TypeString,
			Optional:         !isDataSource,
			Computed:         isDataSource,
			Description:      "The method for analyzing metric events.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"mean", "percentile"}, false)),
			Default:          "mean",
		},
		PERCENTILE_VALUE: {
			Type:             schema.TypeInt,
			Optional:         !isDataSource,
			Computed:         isDataSource,
			Description:      "The percentile for the analysis method. An integer denoting the target percentile between 0 and 100. Required when analysis_type is percentile.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 99)),
			Default:          nil,
		},
		VERSION: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Version of the metric",
		},
	}

	if isDataSource {
		return removeInvalidFieldsForDataSource(schemaMap)
	}

	return schemaMap
}

func metricRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}, isDataSource bool) diag.Diagnostics {
	client := metaRaw.(*Client)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	metric, res, err := client.ld.MetricsApi.GetMetric(client.ctx, projectKey, key).Execute()

	if isStatusNotFound(res) && !isDataSource {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Metric not found",
			Detail:   fmt.Sprintf("[WARN] metric %q in project %q not found, removing from state", key, projectKey),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set(KEY, metric.Key)
	_ = d.Set(NAME, metric.Name)
	_ = d.Set(DESCRIPTION, metric.Description)
	_ = d.Set(TAGS, metric.Tags)
	_ = d.Set(KIND, metric.Kind)
	_ = d.Set(IS_ACTIVE, metric.IsActive)
	_ = d.Set(IS_NUMERIC, metric.IsNumeric)
	_ = d.Set(SELECTOR, metric.Selector)
	_ = d.Set(URLS, metric.Urls)
	_ = d.Set(UNIT, metric.Unit)
	_ = d.Set(EVENT_KEY, metric.EventKey)
	_ = d.Set(SUCCESS_CRITERIA, metric.SuccessCriteria)
	_ = d.Set(RANDOMIZATION_UNITS, metric.RandomizationUnits)
	if metric.EventDefault != nil && metric.EventDefault.Disabled != nil {
		_ = d.Set(INCLUDE_UNITS_WITHOUT_EVENTS, !*metric.EventDefault.Disabled)
	}
	_ = d.Set(UNIT_AGGREGATION_TYPE, metric.UnitAggregationType)
	_ = d.Set(ANALYSIS_TYPE, metric.AnalysisType)
	_ = d.Set(PERCENTILE_VALUE, metric.PercentileValue)
	_ = d.Set(VERSION, metric.Version)

	d.SetId(projectKey + "/" + key)

	return diags
}

func metricUrlSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		KIND: {
			Type:             schema.TypeString,
			Required:         true,
			Description:      "The URL type. Available choices are `exact`, `canonical`, `substring` and `regex`.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"exact", "canonical", "substring", "regex"}, false)),
		},
		URL: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "(Required for kind `exact` and `canonical`) The exact or canonical URL.",
		},
		SUBSTRING: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "(Required for kind `substring`) The URL substring to match by.",
		},
		PATTERN: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "(Required for kind `regex`) The regex pattern to match by.",
		},
	}
}

func metricUrlsFromResourceData(d *schema.ResourceData) []ldapi.UrlPost {
	schemaUrlList := d.Get(URLS).([]interface{})
	urls := make([]ldapi.UrlPost, len(schemaUrlList))
	for i, url := range schemaUrlList {
		urls[i] = metricUrlPostFromResourceData(url)
	}
	return urls
}

func metricUrlPostFromResourceData(urlData interface{}) ldapi.UrlPost {
	urlMap := urlData.(map[string]interface{})
	kind := urlMap[KIND].(string)
	url := urlMap[URL].(string)
	substring := urlMap[SUBSTRING].(string)
	pattern := urlMap[PATTERN].(string)
	urlPost := ldapi.UrlPost{
		Kind:      &kind,
		Url:       &url,
		Substring: &substring,
		Pattern:   &pattern,
	}
	return urlPost
}

func metricIdToKeys(id string) (projectKey string, flagKey string, err error) {
	if strings.Count(id, "/") != 1 {
		return "", "", fmt.Errorf("found unexpected metric id format: %q expected format: 'project_key/metric_key'", id)
	}
	parts := strings.SplitN(id, "/", 2)
	projectKey, flagKey = parts[0], parts[1]
	return projectKey, flagKey, nil
}

// Checks each of the URL config entries to make sure that the required field for each kind is set
// If it isn't return true which breaks out of a forEach down the line
func checkUrlConfigValues(key cty.Value, val cty.Value) bool {
	urlKind := val.GetAttr("kind").AsString()
	substringNull := val.GetAttr("substring").IsNull()
	urlNull := val.GetAttr("url").IsNull()
	patternNull := val.GetAttr("pattern").IsNull()
	switch urlKind {
	case "canonical":
		// Ensure required value is set
		if urlNull {
			return true
		}
		// Disallow keys specific to other 'kind' values - these updates are ignored by the backend and lead to misleading plans being generated
		if !patternNull || !substringNull {
			return true
		}
	case "exact":
		// Ensure required value is set
		if urlNull {
			return true
		}
		// Disallow keys specific to other 'kind' values - these updates are ignored by the backend and lead to misleading plans being generated
		if !patternNull || !substringNull {
			return true
		}
	case "substring":
		// Ensure required value is set
		if substringNull {
			return true
		}
		// Disallow keys specific to other 'kind' values - these updates are ignored by the backend and lead to misleading plans being generated
		if !patternNull || !urlNull {
			return true
		}
	case "pattern":
		// Ensure required value is set
		if patternNull {
			return true
		}
		// Disallow keys specific to other 'kind' values - these updates are ignored by the backend and lead to misleading plans being generated
		if !substringNull || !urlNull {
			return true
		}
	}
	return false
}

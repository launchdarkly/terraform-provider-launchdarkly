package launchdarkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	ldapi "github.com/launchdarkly/api-client-go/v12"
)

func baseMetricSchema(isDataSource bool) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		PROJECT_KEY: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			Description:      "The LaunchDarkly project key",
			ValidateDiagFunc: validateKey(),
		},
		KEY: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validateKey(),
			Description:      "A unique key that will be used to reference the metric in your code",
		},
		NAME: {
			Type:        schema.TypeString,
			Required:    !isDataSource,
			Optional:    isDataSource,
			Description: "A human-readable name for your metric",
		},
		KIND: {
			Type:             schema.TypeString,
			Required:         !isDataSource,
			Optional:         isDataSource,
			Description:      "The metric type -available choices are 'pageview', 'click', and 'custom'",
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"pageview", "click", "custom"}, false)),
			ForceNew:         true,
		},
		MAINTAINER_ID: {
			Type:             schema.TypeString,
			Optional:         true,
			Computed:         true,
			Description:      "The LaunchDarkly ID of the user who will maintain the metric. If not set, the API will automatically apply the member associated with your Terraform API key or the most recently-set maintainer",
			ValidateDiagFunc: validateID(),
		},
		DESCRIPTION: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "A short description of what the metric will be used for",
		},
		TAGS: tagsSchema(),
		IS_ACTIVE: {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether the metric is active",
			Default:     false,
		},
		IS_NUMERIC: {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether the metric is numeric",
			Default:     false,
		},
		UNIT: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The unit for your metric (if numeric metric)",
		},
		SELECTOR: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The CSS selector for your metric (if click metric)",
		},
		EVENT_KEY: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The event key for your metric (if custom metric)",
		},
		SUCCESS_CRITERIA: {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "The success criteria for your metric (if numeric metric)",
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"HigherThanBaseline", "LowerThanBaseline"}, false)),
		},
		URLS: {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "List of nested `url` blocks describing URLs that you want to track the metric on",
			Elem: &schema.Resource{
				Schema: metricUrlSchema(),
			},
		},
	}
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

	d.SetId(projectKey + "/" + key)

	return diags
}

func metricUrlSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		KIND: {
			Type:             schema.TypeString,
			Required:         true,
			Description:      "The url type - vailable choices are 'exact', 'canonical', 'substring' and 'regex'",
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"exact", "canonical", "substring", "regex"}, false)),
		},
		URL: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The exact or canonical URL",
		},
		SUBSTRING: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The URL substring",
		},
		PATTERN: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The URL-matching regex",
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

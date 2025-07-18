package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ldapi "github.com/launchdarkly/api-client-go/v17"
)

const CUSTOM_METRIC_DEFAULT_SUCCESS_CRITERIA = "HigherThanBaseline"

// Our required fields for metrics depend on the value of the 'kind' enum.
// As of now, TF does not support validating multiple attributes at once, so our only options are
// Validating at runtime in Create/Update (and only alerting at apply stage)
// Using CustomizeDiff below (and alerting at plan stage)
// https://github.com/hashicorp/terraform-plugin-sdk/issues/233
func customizeMetricDiff(ctx context.Context, diff *schema.ResourceDiff, v interface{}) error {
	config := diff.GetRawConfig()

	// Kind enum is validated using validateFunc
	kindInConfig := diff.Get(KIND).(string)
	selectorInConfig := config.GetAttr(SELECTOR)
	urlsInConfig := config.GetAttr(URLS)
	successCriteriaInConfig := config.GetAttr(SUCCESS_CRITERIA)
	unitInConfig := config.GetAttr(UNIT)
	eventKeyInConfig := config.GetAttr(EVENT_KEY)
	analysisTypeInConfig := diff.Get(ANALYSIS_TYPE).(string)
	percentileValueInConfig := config.GetAttr(PERCENTILE_VALUE)
	includeUnitsWithoutEventsInConfig := config.GetAttr(INCLUDE_UNITS_WITHOUT_EVENTS)

	// Different validation logic depending on which kind of metric we are creating
	switch kindInConfig {
	case "click":
		if selectorInConfig.IsNull() {
			return fmt.Errorf("click metrics require 'selector' to be set")
		}
		// If we have no keys in the URLS block in the config (length is 0) we know the customer hasn't set any URL values
		urlsSlice := urlsInConfig.AsValueSlice()
		if len(urlsSlice) == 0 {
			return fmt.Errorf("click metrics require an 'urls' block to be set")
		}
		// Determine if the URL blocks have the correct subfields for their kind set
		earlyExit := urlsInConfig.ForEachElement(checkUrlConfigValues)
		if earlyExit {
			return fmt.Errorf("'urls' block is misconfigured, please check documentation for required fields")
		}
		// Disallow keys specific to other 'kind' values - these updates are ignored by the backend and lead to misleading plans being generated
		if !successCriteriaInConfig.IsNull() {
			return fmt.Errorf("click metrics do not accept 'success_criteria'")
		}
		if !unitInConfig.IsNull() {
			return fmt.Errorf("click metrics do not accept 'unit'")
		}
		if !eventKeyInConfig.IsNull() {
			return fmt.Errorf("click metrics do not accept 'event_key'")
		}
	case "custom":
		// enum validation is done in validateFunction against attribute
		if successCriteriaInConfig.IsNull() {
			err := diff.SetNew(SUCCESS_CRITERIA, CUSTOM_METRIC_DEFAULT_SUCCESS_CRITERIA)
			if err != nil {
				return err
			}
		}
		isNumericInConfig := config.GetAttr(IS_NUMERIC)
		// numeric custom metrics have extra required fields
		if isNumericInConfig.True() {
			if successCriteriaInConfig.IsNull() {
				return fmt.Errorf("numeric custom metrics require 'success_criteria' to be set")

			}
			if unitInConfig.IsNull() {
				return fmt.Errorf("numeric custom metrics require 'unit' to be set")
			}
		}
		if eventKeyInConfig.IsNull() {
			return fmt.Errorf("custom meterics require 'event_key' to be set")
		}
		// Disallow keys specific to other 'kind' values - these updates are ignored by the backend and lead to misleading plans being generated
		urlsSlice := urlsInConfig.AsValueSlice()
		if len(urlsSlice) != 0 {
			return fmt.Errorf("custom metrics do not accept a 'urls' block")
		}
		if !selectorInConfig.IsNull() {
			return fmt.Errorf("custom metrics do not accept 'selector'")
		}
	case "pageview":
		// If we have no keys in the URLS block in the config (length is 0) we know the customer hasn't set any URL values
		urlsSlice := urlsInConfig.AsValueSlice()
		if len(urlsSlice) == 0 {
			return fmt.Errorf("pageview metrics require an 'urls' block to be set")
		}
		// Determine if the URL blocks have the correct subfields for their kind set
		earlyExit := urlsInConfig.ForEachElement(checkUrlConfigValues)
		if earlyExit {
			return fmt.Errorf("'urls' block is misconfigured, please check documentation for required fields")
		}

		// Disallow keys specific to other 'kind' values - these updates are ignored by the backend and lead to misleading plans being generated
		if !successCriteriaInConfig.IsNull() {
			return fmt.Errorf("pageview metrics do not accept 'success_criteria'")
		}
		if !unitInConfig.IsNull() {
			return fmt.Errorf("pageview metrics do not accept 'unit'")
		}
		if !eventKeyInConfig.IsNull() {
			return fmt.Errorf("pageview metrics do not accept 'event_key'")
		}

		if !selectorInConfig.IsNull() {
			return fmt.Errorf("pageview metrics do not accept 'selector'")
		}
	}

	if analysisTypeInConfig == "percentile" {
		if percentileValueInConfig.IsNull() {
			return fmt.Errorf("percentile_value is required when analysis_type is percentile")
		}
		if includeUnitsWithoutEventsInConfig.True() {
			return fmt.Errorf("include_units_without_events is not supported for percentile metrics")
		}
	} else if !percentileValueInConfig.IsNull() {
		return fmt.Errorf("%s type metrics can not have percentile values", analysisTypeInConfig)
	}

	// If anything is changed at all, expect a new value will be computed for "version"
	if len(diff.GetChangedKeysPrefix("")) > 0 {
		err := diff.SetNewComputed(VERSION)
		if err != nil {
			return err
		}
	}

	if includeUnitsWithoutEventsInConfig.IsNull() {
		if analysisTypeInConfig == "percentile" {
			err := diff.SetNew(INCLUDE_UNITS_WITHOUT_EVENTS, false)
			if err != nil {
				return err
			}
		} else {
			err := diff.SetNew(INCLUDE_UNITS_WITHOUT_EVENTS, true)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func resourceMetric() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMetricCreate,
		ReadContext:   resourceMetricRead,
		UpdateContext: resourceMetricUpdate,
		DeleteContext: resourceMetricDelete,
		Schema:        baseMetricSchema(false),
		CustomizeDiff: customizeMetricDiff,
		Importer: &schema.ResourceImporter{
			State: resourceMetricImport,
		},

		Description: `Provides a LaunchDarkly metric resource.

This resource allows you to create and manage metrics within your LaunchDarkly organization.

To learn more about metrics and experimentation, read [Experimentation Documentation](https://docs.launchdarkly.com/home/experimentation).`,
	}
}

func resourceMetricCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	projectKey := d.Get(PROJECT_KEY).(string)

	if exists, err := projectExists(projectKey, client); !exists {
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.Errorf("cannot find project with key %q", projectKey)
	}

	key := d.Get(KEY).(string)
	name := d.Get(NAME).(string)
	kind := d.Get(KIND).(string)
	description := d.Get(DESCRIPTION).(string)
	tags := stringsFromResourceData(d, TAGS)
	isActive := d.Get(IS_ACTIVE).(bool)
	isNumeric := d.Get(IS_NUMERIC).(bool)
	urls := metricUrlsFromResourceData(d)
	randomizationUnits := stringsFromResourceData(d, RANDOMIZATION_UNITS)
	// Required depending on type
	unit := d.Get(UNIT).(string)
	selector := d.Get(SELECTOR).(string)
	eventKey := d.Get(EVENT_KEY).(string)
	unitAggregationType := d.Get(UNIT_AGGREGATION_TYPE).(string)
	analysisType := d.Get(ANALYSIS_TYPE).(string)
	includeUnitsWithoutEvents := d.Get(INCLUDE_UNITS_WITHOUT_EVENTS).(bool)
	eventDefaultDisabled := !includeUnitsWithoutEvents

	metric := ldapi.MetricPost{
		Name:                &name,
		Key:                 key,
		Description:         &description,
		Tags:                tags,
		Kind:                kind,
		IsActive:            &isActive,
		IsNumeric:           &isNumeric,
		Selector:            &selector,
		Urls:                urls,
		RandomizationUnits:  randomizationUnits,
		Unit:                &unit,
		EventKey:            &eventKey,
		UnitAggregationType: &unitAggregationType,
		AnalysisType:        &analysisType,
		EventDefault:        &ldapi.MetricEventDefaultRep{Disabled: &eventDefaultDisabled},
	}
	percentileValueData, hasPercentile := d.GetOk(PERCENTILE_VALUE)
	if hasPercentile {
		percentileValue := int32(percentileValueData.(int))
		metric.PercentileValue = &percentileValue
	}
	// Only add successCriteria if it has a value - empty string causes API errors
	_, ok := d.GetOk(SUCCESS_CRITERIA)
	if ok {
		successCriteria := d.Get(SUCCESS_CRITERIA).(string)
		metric.SuccessCriteria = &successCriteria
	} else {
		if kind == "custom" {
			successCriteria := CUSTOM_METRIC_DEFAULT_SUCCESS_CRITERIA
			metric.SuccessCriteria = &successCriteria
		}
	}

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.MetricsApi.PostMetric(client.ctx, projectKey).MetricPost(metric).Execute()
		return err
	})

	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error creating metric resource: %q", key),
			Detail:   fmt.Sprintf("Details: \n %q", handleLdapiErr(err)),
		})
		return diags
	}

	// Docs imply we can set another maintainer if wanted, this can't be done during create
	// So it has to be done in a subsequent update call
	maintainerId, maintainerIdOk := d.GetOk(MAINTAINER_ID)
	if maintainerIdOk {
		_ = d.Set(MAINTAINER_ID, maintainerId)
		diags = resourceMetricUpdate(ctx, d, metaRaw)
		if diags.HasError() {
			// if there was a problem in the update state, we need to clean up completely by deleting the flag
			var deleteErr error
			deleteErr = client.withConcurrency(client.ctx, func() error {
				_, deleteErr = client.ld.MetricsApi.DeleteMetric(client.ctx, projectKey, key).Execute()
				return deleteErr
			})
			if deleteErr != nil {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("Error creating metric resource: %q", key),
					Detail:   fmt.Sprintf("failed to clean up metric %q from project %q: %s", key, projectKey, handleLdapiErr(err)),
				})
				return diags
			}
			return diags
		}
	}

	d.SetId(projectKey + "/" + key)

	return resourceMetricRead(ctx, d, metaRaw)
}

func resourceMetricRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	return metricRead(ctx, d, metaRaw, false)
}

func resourceMetricUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)
	name := d.Get(NAME).(string)
	kind := d.Get(KIND).(string)
	description := d.Get(DESCRIPTION).(string)
	tags := stringsFromResourceData(d, TAGS)
	isActive := d.Get(IS_ACTIVE).(bool)
	isNumeric := d.Get(IS_NUMERIC).(bool)
	urls := metricUrlsFromResourceData(d)
	// Required depending on type
	unit := d.Get(UNIT).(string)
	selector := d.Get(SELECTOR).(string)
	eventKey := d.Get(EVENT_KEY).(string)
	unitAggregationType := d.Get(UNIT_AGGREGATION_TYPE).(string)
	analysisType := d.Get(ANALYSIS_TYPE).(string)
	includeUnitsWithoutEvents := d.Get(INCLUDE_UNITS_WITHOUT_EVENTS).(bool)

	patch := []ldapi.PatchOperation{
		patchReplace("/name", name),
		patchReplace("/description", description),
		patchReplace("/tags", tags),
		patchReplace("/kind", kind),
		patchReplace("/isActive", isActive),
		patchReplace("/isNumeric", isNumeric),
		patchReplace("/urls", urls),
		patchReplace("/unit", unit),
		patchReplace("/selector", selector),
		patchReplace("/eventKey", eventKey),
		patchReplace("/unitAggregationType", unitAggregationType),
		patchReplace("/analysisType", analysisType),
		patchReplace("/eventDefault/disabled", !includeUnitsWithoutEvents),
	}

	percentileValueData, ok := d.GetOk(PERCENTILE_VALUE)
	if ok {
		patch = append(patch, patchReplace("/percentileValue", int32(percentileValueData.(int))))
	} else {
		patch = append(patch, patchReplace("/percentileValue", nil))
	}

	// Only update successCriteria if it is specified in the schema (enum values)
	successCriteria, ok := d.GetOk(SUCCESS_CRITERIA)
	if ok {
		patch = append(patch, patchReplace("/successCriteria", successCriteria.(string)))
	} else {
		if kind == "custom" {
			patch = append(patch, patchReplace("/successCriteria", CUSTOM_METRIC_DEFAULT_SUCCESS_CRITERIA))
		}
	}

	// Only update the maintainer ID if is specified in the schema
	maintainerID, ok := d.GetOk(MAINTAINER_ID)
	if ok {
		patch = append(patch, patchReplace("/maintainerId", maintainerID.(string)))
	}

	// Only update randomization units if it is specified in the schema
	if _, ok := d.GetOk(RANDOMIZATION_UNITS); ok {
		patch = append(patch, patchReplace("/randomizationUnits", stringsFromResourceData(d, RANDOMIZATION_UNITS)))
	}

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.MetricsApi.PatchMetric(client.ctx, projectKey, key).PatchOperation(patch).Execute()
		return err
	})

	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error updating metric resource %q from project %q", key, projectKey),
			Detail:   fmt.Sprintf("Details: \n %q", handleLdapiErr(err)),
		})
		return diags
	}
	return resourceMetricRead(ctx, d, metaRaw)
}

func resourceMetricDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, err = client.ld.MetricsApi.DeleteMetric(client.ctx, projectKey, key).Execute()
		return err
	})
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error deleting metric resource %q from project %q", key, projectKey),
			Detail:   fmt.Sprintf("Details: \n %q", handleLdapiErr(err)),
		})
		return diags
	}

	return resourceMetricRead(ctx, d, metaRaw)
}

func resourceMetricImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()

	projectKey, metricKey, err := metricIdToKeys(id)
	if err != nil {
		return nil, err
	}
	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(KEY, metricKey)

	return []*schema.ResourceData{d}, nil
}

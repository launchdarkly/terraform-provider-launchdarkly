package launchdarkly

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	ldapi "github.com/launchdarkly/api-client-go/v12"
)

func baseTargetsSchema(isDataSource bool) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		VALUES: {
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Required:    true,
			Description: "List of `user` strings to target.",
		},
		VARIATION: {
			Type:             schema.TypeInt,
			Required:         true,
			Description:      "The index of the variation to serve if a user target value is matched.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
		},
	}
}

func targetsSchema(isDataSource bool) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeSet,
		Optional:    !isDataSource,
		Computed:    isDataSource,
		Description: "Set of nested blocks describing the individual user targets for each variation.",
		Elem: &schema.Resource{
			Schema: baseTargetsSchema(isDataSource),
		},
	}
}

func contextTargetsSchema(isDataSource bool) *schema.Schema {
	schemaMap := baseTargetsSchema(isDataSource)
	schemaMap[CONTEXT_KIND] = &schema.Schema{
		Type:             schema.TypeString,
		Required:         !isDataSource,
		Computed:         isDataSource,
		Description:      "The context kind on which the flag should target in this environment. User (`user`) targets should be specified as `targets` attribute blocks.",
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringNotInSlice([]string{"user"}, true)),
	}
	if isDataSource {
		schemaMap = removeInvalidFieldsForDataSource(schemaMap)
	}
	return &schema.Schema{
		Type:        schema.TypeSet,
		Optional:    !isDataSource,
		Computed:    isDataSource,
		Description: "The set of nested blocks describing the individual targets for non-user context kinds for each variation.",
		Elem: &schema.Resource{
			Schema: schemaMap,
		},
	}
}

type targetOptions struct {
	isContextTarget bool
}

func targetsFromResourceData(d *schema.ResourceData, options targetOptions) []ldapi.Target {
	tgts, ok := d.GetOk(TARGETS)
	if options.isContextTarget {
		tgts, ok = d.GetOk(CONTEXT_TARGETS)
	}
	if !ok {
		return []ldapi.Target{}
	}
	schemaTargets := tgts.(*schema.Set).List()
	targets := make([]ldapi.Target, 0, len(schemaTargets))
	for _, target := range schemaTargets {
		targetMap := target.(map[string]interface{})
		targets = append(targets, targetFromResourceData(targetMap, options))
	}
	return targets
}

func targetFromResourceData(targetMap map[string]interface{}, options targetOptions) ldapi.Target {
	contextKind := "user"
	resourceValues := targetMap[VALUES].([]interface{})
	values := make([]string, 0, len(resourceValues))
	for _, v := range resourceValues {
		values = append(values, v.(string))
	}
	if options.isContextTarget {
		contextKind = targetMap[CONTEXT_KIND].(string)
	}
	target := ldapi.Target{
		Variation:   int32(targetMap[VARIATION].(int)),
		Values:      values,
		ContextKind: &contextKind, // default to user if a regular target
	}
	return target
}

// targetToResourceData converts the `target` information returned
// by the LaunchDarkly API into a format suitable for Terraform
func targetsToResourceData(targets []ldapi.Target, options targetOptions) []interface{} {
	transformed := make([]interface{}, 0, len(targets))
	for _, target := range targets {
		if options.isContextTarget && *target.ContextKind == "user" {
			// the API client returns an empty target with "user" context kind
			// on the ContextTargets attribute to maintain order when evaluating
			// we want to skip these because they will mess up the states
			continue
		}
		resourceTarget := map[string]interface{}{
			VALUES:    target.Values,
			VARIATION: int(target.Variation),
		}
		if options.isContextTarget {
			resourceTarget[CONTEXT_KIND] = *target.ContextKind
		}
		transformed = append(transformed, resourceTarget)
	}
	return transformed
}

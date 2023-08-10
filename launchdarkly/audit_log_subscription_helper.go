package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	strcase "github.com/stoewer/go-strcase"
)

//go:generate codegen -o audit_log_subscription_configs_generated.go

var KEBAB_CASE_INTEGRATIONS = []string{"splunk"}

type IntegrationConfig map[string]FormVariable

type FormVariable struct {
	Type          string
	IsOptional    bool
	AllowedValues []string
	DefaultValue  interface{}
	Description   string
	IsSecret      bool
}

// There is not currently a manifest for slack webhooks so we have to use this for now.
var EXTRA_SUBSCRIPTION_CONFIGURATION_FIELDS = map[string]IntegrationConfig{
	"slack": {
		"url": {
			Type:          "uri",
			IsOptional:    false,
			AllowedValues: []string{},
			DefaultValue:  nil,
			IsSecret:      false,
		},
	},
}

func getSubscriptionConfigurationMap() map[string]IntegrationConfig {
	configs := make(map[string]IntegrationConfig, len(SUBSCRIPTION_CONFIGURATION_FIELDS)+len(EXTRA_SUBSCRIPTION_CONFIGURATION_FIELDS))
	for k, v := range SUBSCRIPTION_CONFIGURATION_FIELDS {
		configs[k] = v
	}
	for k, v := range EXTRA_SUBSCRIPTION_CONFIGURATION_FIELDS {
		configs[k] = v
	}
	return configs
}

func getValidIntegrationKeys() []string {
	configMap := getSubscriptionConfigurationMap()
	integrationKeys := make([]string, 0, len(configMap))
	for k := range configMap {
		integrationKeys = append(integrationKeys, k)
	}
	sort.Strings(integrationKeys)
	return integrationKeys
}

func formatIntegrationKeysForDescription(integrationKeys []string) string {
	output := ""
	for idx, key := range integrationKeys {
		output += fmt.Sprintf("`%s`", key)
		if idx < len(integrationKeys)-2 {
			output += ", "
		} else if idx == len(integrationKeys)-2 {
			output += ", and "
		}
	}
	return output
}

func auditLogSubscriptionSchema(isDataSource bool) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		INTEGRATION_KEY: {
			// validated as part of the config validation
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringInSlice(getValidIntegrationKeys(), false),
			ForceNew:     true,
			Description:  fmt.Sprintf("The integration key. Supported integration keys are %s. A change in this field will force the destruction of the existing resource and the creation of a new one.", formatIntegrationKeysForDescription(getValidIntegrationKeys())),
		},
		NAME: {
			Type:        schema.TypeString,
			Required:    !isDataSource,
			Computed:    isDataSource,
			Description: "A human-friendly name for your audit log subscription viewable from within the LaunchDarkly Integrations page.",
		},
		CONFIG: {
			Type:        schema.TypeMap,
			Required:    !isDataSource,
			Computed:    isDataSource,
			Description: "The set of configuration fields corresponding to the value defined for `integration_key`. Refer to the `formVariables` field in the corresponding `integrations/<integration_key>/manifest.json` file in [this repo](https://github.com/launchdarkly/integration-framework/tree/master/integrations) for a full list of fields for the integration you wish to configure. **IMPORTANT**: Please note that Terraform will only accept these in snake case, regardless of the case shown in the manifest.",
		},
		STATEMENTS: policyStatementsSchema(policyStatementSchemaOptions{
			required:    !isDataSource,
			computed:    isDataSource,
			description: "A block representing the resources to which you wish to subscribe.",
		},
		),
		ON: {
			Type:        schema.TypeBool,
			Required:    !isDataSource,
			Computed:    isDataSource,
			Description: "Whether or not you want your subscription enabled, i.e. to actively send events.",
		},
		TAGS: tagsSchema(tagsSchemaOptions{isDataSource: isDataSource}),
	}
}

func getConfigFieldKey(integrationKey, resourceKey string) string {
	// a select number of integrations take fields in kebab case, ex. "skip-ca-verification"
	// currently this only applies to splunk
	for _, integration := range KEBAB_CASE_INTEGRATIONS {
		if integrationKey == integration {
			return strcase.KebabCase(resourceKey)
		}
	}
	return strcase.LowerCamelCase(resourceKey)
}

// configFromResourceData uses the configuration generated into audit_log_subscription_config.json
// to validate and generate the config the API expects
func configFromResourceData(d *schema.ResourceData) (map[string]interface{}, error) {
	// TODO: refactor to return list of diags warnings with all formatting errors
	integrationKey := d.Get(INTEGRATION_KEY).(string)
	config := d.Get(CONFIG).(map[string]interface{})
	configMap := getSubscriptionConfigurationMap()
	configFormat, ok := configMap[integrationKey]
	if !ok {
		return config, fmt.Errorf("%s is not a valid integration_key for audit log subscriptions", integrationKey)
	}
	for k := range config {
		// error if an incorrect config variable has been set
		key := getConfigFieldKey(integrationKey, k) // convert casing to compare to required config format
		if integrationKey == "datadog" && key == "hostUrl" {
			// this is a one-off for now
			key = "hostURL"
		}
		if _, ok := configFormat[key]; !ok {
			return config, fmt.Errorf("config variable %s not valid for integration type %s", k, integrationKey)
		}
	}
	convertedConfig := make(map[string]interface{}, len(config))
	for k, v := range configFormat {
		key := strcase.SnakeCase(k) // convert to snake case to validate user config
		rawValue, ok := config[key]
		if !ok {
			if !v.IsOptional {
				return config, fmt.Errorf("config variable %s must be set", key)
			}
			// we will let the API handle default configs for now since it otherwise messes
			// up the plan if we set an attribute a user has not set on a non-computed attribute
			continue
		}
		// type will be one of ["string", "boolean", "uri", "enum", "oauth", "dynamicEnum"]
		// for now we do not need to handle oauth or dynamicEnum
		switch v.Type {
		case "string", "uri":
			// we'll let the API handle the URI validation for now
			value := rawValue.(string)
			convertedConfig[k] = value
		case "boolean":
			value, err := strconv.ParseBool(rawValue.(string)) // map values may only be one type, so all non-string types have to be converted
			if err != nil {
				return config, fmt.Errorf("config value %s for %v must be of type bool", rawValue, k)
			}
			convertedConfig[k] = value
		case "enum":
			value := rawValue.(string)
			if !stringInSlice(value, v.AllowedValues) {
				return config, fmt.Errorf("config value %s for %v must be one of the following approved string values: %v", rawValue, k, v.AllowedValues)
			}
			convertedConfig[k] = value
		default:
			// just set to the existing value
			convertedConfig[k] = rawValue
		}
	}
	return convertedConfig, nil
}

func configToResourceData(d *schema.ResourceData, config map[string]interface{}, isDataSource bool) (map[string]interface{}, error) {
	integrationKey := d.Get(INTEGRATION_KEY).(string)
	configMap := getSubscriptionConfigurationMap()
	configFormat, ok := configMap[integrationKey]
	if !ok {
		return config, fmt.Errorf("%s is not a currently supported integration_key for audit log subscriptions", integrationKey)
	}
	originalConfig := d.Get(CONFIG).(map[string]interface{})
	convertedConfig := make(map[string]interface{}, len(config))
	for k, v := range config {
		key := strcase.SnakeCase(k)
		// some attributes have defaults that the API will return and terraform will complain since config
		// is not a computed attribute (cannot be both required & computed). This does not apply for data sources.
		// TODO: handle this in a SuppressDiff function
		if _, setByUser := originalConfig[key]; !setByUser && !isDataSource {
			continue
		}
		convertedConfig[key] = v
		if value, isBool := v.(bool); isBool {
			convertedConfig[key] = strconv.FormatBool(value)
		}
		if configFormat[k].IsSecret {
			// if the user didn't put it in as obfuscated, we don't want to set it as obfuscated
			convertedConfig[key] = originalConfig[key]
		}
	}
	return convertedConfig, nil
}

func auditLogSubscriptionRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := metaRaw.(*Client)
	var id string
	if isDataSource {
		id = d.Get(ID).(string)
	} else {
		id = d.Id()
	}
	integrationKey := d.Get(INTEGRATION_KEY).(string)

	sub, res, err := client.ld.IntegrationAuditLogSubscriptionsApi.GetSubscriptionByID(client.ctx, integrationKey, id).Execute()

	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[WARN] failed to find integration with ID %q, removing from state if present", id)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find integration with ID %q, removing from state if present", id),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get integration with ID %q: %v", id, err)
	}

	if isDataSource {
		d.SetId(*sub.Id)
	}

	_ = d.Set(NAME, sub.Name)
	_ = d.Set(ON, sub.On)
	cfg, err := configToResourceData(d, sub.Config, isDataSource)
	if err != nil {
		return diag.Errorf("failed to set config on integration with id %q: %v", *sub.Id, err)
	}
	err = d.Set(CONFIG, cfg)
	if err != nil {
		return diag.Errorf("failed to set config on integration with id %q: %v", *sub.Id, err)
	}
	err = d.Set(STATEMENTS, policyStatementsToResourceData(sub.Statements))
	if err != nil {
		return diag.Errorf("failed to set statements on integration with id %q: %v", *sub.Id, err)
	}
	err = d.Set(TAGS, sub.Tags)
	if err != nil {
		return diag.Errorf("failed to set tags on integration with id %q: %v", *sub.Id, err)
	}
	return diags
}

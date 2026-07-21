package launchdarkly

import (
	"sort"

	strcase "github.com/stoewer/go-strcase"
)

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

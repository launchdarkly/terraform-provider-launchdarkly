package launchdarkly

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func baseFeatureFlagSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		PROJECT_KEY: {
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			Description:  "The feature flag's project key",
			ValidateFunc: validateKey(),
		},
		KEY: {
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: validateKey(),
			Description:  "The human-readable name of the feature flag",
		},
		MAINTAINER_ID: {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validateID(),
		},
		DESCRIPTION: {
			Type:     schema.TypeString,
			Optional: true,
		},
		VARIATIONS: variationsSchema(),
		TEMPORARY: {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},
		INCLUDE_IN_SNIPPET: {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},
		TAGS:              tagsSchema(),
		CUSTOM_PROPERTIES: customPropertiesSchema(),
		DEFAULT_ON_VARIATION: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The value of the variation served when the flag is on for new environments",
		},
		DEFAULT_OFF_VARIATION: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The value of the variation served when the flag is off for new environments",
		},
	}
}

func featureFlagRead(d *schema.ResourceData, raw interface{}, isDataSource bool) error {
	client := raw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	flagRaw, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, key, nil)
	})
	flag := flagRaw.(ldapi.FeatureFlag)
	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[WARN] feature flag %q in project %q not found, removing from state", key, projectKey)
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to get flag %q of project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	transformedCustomProperties := customPropertiesToResourceData(flag.CustomProperties)
	_ = d.Set(key, flag.Key)
	_ = d.Set(NAME, flag.Name)
	_ = d.Set(DESCRIPTION, flag.Description)
	_ = d.Set(INCLUDE_IN_SNIPPET, flag.IncludeInSnippet)
	_ = d.Set(TEMPORARY, flag.Temporary)

	// Only set the maintainer ID if is specified in the schema
	_, ok := d.GetOk(MAINTAINER_ID)
	if ok {
		_ = d.Set(MAINTAINER_ID, flag.MaintainerId)
	}

	variationType, err := variationsToVariationType(flag.Variations)
	if err != nil {
		return fmt.Errorf("failed to determine variation type on flag with key %q: %v", flag.Key, err)
	}
	err = d.Set(VARIATION_TYPE, variationType)
	if err != nil {
		return fmt.Errorf("failed to set variation type on flag with key %q: %v", flag.Key, err)
	}

	parsedVariations, err := variationsToResourceData(flag.Variations, variationType)
	if err != nil {
		return fmt.Errorf("failed to parse variations on flag with key %q: %v", flag.Key, err)
	}
	err = d.Set(VARIATIONS, parsedVariations)
	if err != nil {
		return fmt.Errorf("failed to set variations on flag with key %q: %v", flag.Key, err)
	}

	err = d.Set(TAGS, flag.Tags)
	if err != nil {
		return fmt.Errorf("failed to set tags on flag with key %q: %v", flag.Key, err)
	}

	err = d.Set(CUSTOM_PROPERTIES, transformedCustomProperties)
	if err != nil {
		return fmt.Errorf("failed to set custom properties on flag with key %q: %v", flag.Key, err)
	}

	if flag.Defaults != nil {
		onValue, err := variationValueToString(flag.Variations[flag.Defaults.OnVariation].Value, variationType)
		if err != nil {
			return err
		}
		_ = d.Set(DEFAULT_ON_VARIATION, onValue)
		offValue, err := variationValueToString(flag.Variations[flag.Defaults.OffVariation].Value, variationType)
		if err != nil {
			return err
		}
		_ = d.Set(DEFAULT_OFF_VARIATION, offValue)
	}

	d.SetId(projectKey + "/" + key)
	return nil
}

func flagIdToKeys(id string) (projectKey string, flagKey string, err error) {
	if strings.Count(id, "/") != 1 {
		return "", "", fmt.Errorf("found unexpected flag id format: %q expected format: 'project_key/flag_key'", id)
	}
	parts := strings.SplitN(id, "/", 2)
	projectKey, flagKey = parts[0], parts[1]
	return projectKey, flagKey, nil
}

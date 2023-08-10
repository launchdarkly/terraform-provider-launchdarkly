package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v12"
)

type featureFlagSchemaOptions struct {
	isDataSource bool
}

func baseFeatureFlagSchema(options featureFlagSchemaOptions) map[string]*schema.Schema {
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
			Description:      "A unique key that will be used to reference the flag in your code",
		},
		MAINTAINER_ID: {
			Type:             schema.TypeString,
			Optional:         true,
			Computed:         true,
			Description:      "The LaunchDarkly id of the user who will maintain the flag. If not set, the API will automatically apply the member associated with your Terraform API key or the most recently set maintainer",
			ValidateDiagFunc: validateID(),
		},
		DESCRIPTION: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "A short description of what the flag will be used for",
		},
		VARIATIONS: variationsSchema(),
		TEMPORARY: {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether or not the flag is a temporary flag",
			Default:     false,
		},
		INCLUDE_IN_SNIPPET: {
			Type:          schema.TypeBool,
			Optional:      true,
			Computed:      true,
			Description:   "Whether or not this flag should be made available to the client-side JavaScript SDK",
			Deprecated:    "'include_in_snippet' is now deprecated. Please migrate to 'client_side_availability' to maintain future compatability.",
			ConflictsWith: []string{CLIENT_SIDE_AVAILABILITY},
		},
		// Annoying that we can't define a typemap to have specific keys https://www.terraform.io/docs/extend/schemas/schema-types.html#typemap
		CLIENT_SIDE_AVAILABILITY: {
			Type:          schema.TypeList,
			Optional:      true,
			Computed:      true,
			ConflictsWith: []string{INCLUDE_IN_SNIPPET},
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					USING_ENVIRONMENT_ID: {
						Type:     schema.TypeBool,
						Optional: true,
						Computed: true,
					},
					USING_MOBILE_KEY: {
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
				},
			},
		},
		TAGS:              tagsSchema(tagsSchemaOptions(options)),
		CUSTOM_PROPERTIES: customPropertiesSchema(),
		DEFAULTS: {
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			Description: "The default variations used for this flag in new environments. If omitted, the first and last variation will be used",
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					ON_VARIATION: {
						Type:             schema.TypeInt,
						Required:         true,
						Description:      "The index of the variation served when the flag is on for new environments",
						ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
					},
					OFF_VARIATION: {
						Type:             schema.TypeInt,
						Required:         true,
						Description:      "The index of the variation served when the flag is off for new environments",
						ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
					},
				},
			},
		},
		ARCHIVED: {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether to archive the flag",
			Default:     false,
		},
	}
}

func featureFlagRead(ctx context.Context, d *schema.ResourceData, raw interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := raw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	flag, res, err := client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, key).Execute()

	if isStatusNotFound(res) && !isDataSource {
		// TODO: Can probably get rid of all of these WARN logs?
		log.Printf("[WARN] feature flag %q in project %q not found, removing from state", key, projectKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] feature flag %q in project %q not found, removing from state", key, projectKey),
		})
		d.SetId("")
		return diags
	}

	if err != nil {
		return diag.Errorf("failed to get flag %q of project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	transformedCustomProperties := customPropertiesToResourceData(flag.CustomProperties)
	_ = d.Set(KEY, flag.Key)
	_ = d.Set(NAME, flag.Name)
	_ = d.Set(DESCRIPTION, flag.Description)
	_ = d.Set(TEMPORARY, flag.Temporary)
	_ = d.Set(ARCHIVED, flag.Archived)

	CSA := *flag.ClientSideAvailability
	clientSideAvailability := []map[string]interface{}{{
		USING_ENVIRONMENT_ID: CSA.UsingEnvironmentId,
		USING_MOBILE_KEY:     CSA.UsingMobileKey,
	}}
	// Always set both CSA and IIS to state in order to correctly represent the flag resource as it exists in LD
	_ = d.Set(CLIENT_SIDE_AVAILABILITY, clientSideAvailability)
	_ = d.Set(INCLUDE_IN_SNIPPET, CSA.UsingEnvironmentId)

	// Only set the maintainer ID if is specified in the schema
	_, ok := d.GetOk(MAINTAINER_ID)
	if ok {
		_ = d.Set(MAINTAINER_ID, flag.MaintainerId)
	}

	variationType, err := variationsToVariationType(flag.Variations)
	if err != nil {
		return diag.Errorf("failed to determine variation type on flag with key %q: %v", flag.Key, err)
	}
	err = d.Set(VARIATION_TYPE, variationType)
	if err != nil {
		return diag.Errorf("failed to set variation type on flag with key %q: %v", flag.Key, err)
	}

	parsedVariations, err := variationsToResourceData(flag.Variations, variationType)
	if err != nil {
		return diag.Errorf("failed to parse variations on flag with key %q: %v", flag.Key, err)
	}
	err = d.Set(VARIATIONS, parsedVariations)
	if err != nil {
		return diag.Errorf("failed to set variations on flag with key %q: %v", flag.Key, err)
	}

	err = d.Set(TAGS, flag.Tags)
	if err != nil {
		return diag.Errorf("failed to set tags on flag with key %q: %v", flag.Key, err)
	}

	err = d.Set(CUSTOM_PROPERTIES, transformedCustomProperties)
	if err != nil {
		return diag.Errorf("failed to set custom properties on flag with key %q: %v", flag.Key, err)
	}

	var defaults []map[string]interface{}
	if flag.Defaults != nil {
		defaults = []map[string]interface{}{{
			ON_VARIATION:  flag.Defaults.OnVariation,
			OFF_VARIATION: flag.Defaults.OffVariation,
		}}
	} else {
		defaults = []map[string]interface{}{{
			ON_VARIATION:  0,
			OFF_VARIATION: len(flag.Variations) - 1,
		}}
	}
	_ = d.Set(DEFAULTS, defaults)

	d.SetId(projectKey + "/" + key)
	return diags
}

func flagIdToKeys(id string) (projectKey string, flagKey string, err error) {
	if strings.Count(id, "/") != 1 {
		return "", "", fmt.Errorf("found unexpected flag id format: %q expected format: 'project_key/flag_key'", id)
	}
	parts := strings.SplitN(id, "/", 2)
	projectKey, flagKey = parts[0], parts[1]
	return projectKey, flagKey, nil
}

func getProjectDefaultCSAandIncludeInSnippet(client *Client, projectKey string) (ldapi.ClientSideAvailability, bool, error) {
	project, _, err := client.ld.ProjectsApi.GetProject(client.ctx, projectKey).Execute()
	if err != nil {
		return ldapi.ClientSideAvailability{}, false, err
	}

	return *project.DefaultClientSideAvailability, project.IncludeInSnippetByDefault, nil
}

package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

type featureFlagSchemaOptions struct {
	isDataSource bool
}

func baseFeatureFlagSchema(options featureFlagSchemaOptions) map[string]*schema.Schema {
	schemaMap := map[string]*schema.Schema{
		PROJECT_KEY: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         !options.isDataSource,
			Description:      addForceNewDescription("The feature flag's project key.", !options.isDataSource),
			ValidateDiagFunc: validateKey(),
		},
		KEY: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         !options.isDataSource,
			ValidateDiagFunc: validateKey(),
			Description:      addForceNewDescription("The unique feature flag key that references the flag in your application code.", !options.isDataSource),
		},
		MAINTAINER_ID: {
			Type:             schema.TypeString,
			Optional:         true,
			Computed:         true,
			Description:      "The feature flag maintainer's 24 character alphanumeric team member ID. `maintainer_team_key` cannot be set if `maintainer_id` is set. If neither is set, it will automatically be or stay set to the member ID associated with the API key used by your LaunchDarkly Terraform provider or the most recently-set maintainer.",
			ValidateDiagFunc: validateID(),
			ConflictsWith:    []string{MAINTAINER_TEAM_KEY},
		},
		MAINTAINER_TEAM_KEY: {
			Type:             schema.TypeString,
			Optional:         true,
			Computed:         true,
			Description:      "The key of the associated team that maintains this feature flag. `maintainer_id` cannot be set if `maintainer_team_key` is set",
			ValidateDiagFunc: validateKeyAndLength(int(1), int(256)),
			ConflictsWith:    []string{MAINTAINER_ID},
		},
		DESCRIPTION: {
			Type:        schema.TypeString,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "The feature flag's description.",
		},
		VARIATIONS: variationsSchema(options.isDataSource),
		TEMPORARY: {
			Type:        schema.TypeBool,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "Specifies whether the flag is a temporary flag.",
			Default:     false,
		},
		INCLUDE_IN_SNIPPET: {
			Type:          schema.TypeBool,
			Optional:      !options.isDataSource,
			Computed:      true,
			Description:   "Specifies whether this flag should be made available to the client-side JavaScript SDK using the client-side Id. This value gets its default from your project configuration if not set. `include_in_snippet` is now deprecated. Please migrate to `client_side_availability.using_environment_id` to maintain future compatibility.",
			Deprecated:    "'include_in_snippet' is now deprecated. Please migrate to 'client_side_availability' to maintain future compatability.",
			ConflictsWith: []string{CLIENT_SIDE_AVAILABILITY},
		},
		// Annoying that we can't define a typemap to have specific keys https://www.terraform.io/docs/extend/schemas/schema-types.html#typemap
		CLIENT_SIDE_AVAILABILITY: {
			Type:          schema.TypeList,
			Optional:      !options.isDataSource,
			Computed:      true,
			ConflictsWith: []string{INCLUDE_IN_SNIPPET},
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					USING_ENVIRONMENT_ID: {
						Type:        schema.TypeBool,
						Optional:    !options.isDataSource,
						Computed:    true,
						Description: "Whether this flag is available to SDKs using the client-side ID.",
					},
					USING_MOBILE_KEY: {
						Type:        schema.TypeBool,
						Optional:    !options.isDataSource,
						Computed:    options.isDataSource,
						Default:     emptyValueIfDataSource(false, options.isDataSource),
						Description: "Whether this flag is available to SDKs using a mobile key.",
					},
				},
				Description: "A block describing whether this flag should be made available to the client-side JavaScript SDK using the client-side Id, mobile key, or both. This value gets its default from your project configuration if not set. Once set, if removed, it will retain its last set value.",
			},
		},
		TAGS:              tagsSchema(tagsSchemaOptions(options)),
		CUSTOM_PROPERTIES: customPropertiesSchema(options.isDataSource),
		DEFAULTS: {
			Type:        schema.TypeList,
			Optional:    !options.isDataSource,
			Computed:    true,
			Description: "A block containing the indices of the variations to be used as the default on and off variations in all new environments. Flag configurations in existing environments will not be changed nor updated if the configuration block is removed.",
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					ON_VARIATION: {
						Type:             schema.TypeInt,
						Required:         true,
						Description:      "The index of the variation the flag will default to in all new environments when on.",
						ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
					},
					OFF_VARIATION: {
						Type:             schema.TypeInt,
						Required:         true,
						Description:      "The index of the variation the flag will default to in all new environments when off.",
						ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
					},
				},
			},
		},
		ARCHIVED: {
			Type:        schema.TypeBool,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "Specifies whether the flag is archived or not. Note that you cannot create a new flag that is archived, but can update a flag to be archived.",
			Default:     false,
		},
	}

	if options.isDataSource {
		schemaMap = removeInvalidFieldsForDataSource(schemaMap)
	}

	return schemaMap
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

	// Only set the maintainer if is specified in the schema
	_, maintainerIdOk := d.GetOk(MAINTAINER_ID)
	_, maintainerTeamKeyOk := d.GetOk(MAINTAINER_TEAM_KEY)
	if maintainerIdOk || maintainerTeamKeyOk {
		_ = d.Set(MAINTAINER_TEAM_KEY, flag.MaintainerTeamKey)
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

	// For data sources, also fetch and set linked views for discovery
	if isDataSource {
		betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S)
		if err != nil {
			log.Printf("[WARN] failed to create beta client for views lookup: %v", err)
		} else {
			viewsWithFlag, err := getViewsContainingFlag(betaClient, projectKey, key)
			if err != nil {
				// Log warning but don't fail the read for discovery data
				log.Printf("[WARN] failed to get views for flag %q in project %q: %v", key, projectKey, err)
			} else {
				err = d.Set(VIEWS, viewsWithFlag)
				if err != nil {
					return diag.Errorf("could not set views on flag with key %q: %v", key, err)
				}
			}
		}
	}

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

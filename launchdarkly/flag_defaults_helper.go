package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

func baseFlagDefaultsSchema(isDataSource bool) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		PROJECT_KEY: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         !isDataSource,
			Description:      addForceNewDescription("The project key.", !isDataSource),
			ValidateDiagFunc: validateKey(),
		},
		TAGS: tagsSchema(tagsSchemaOptions{isDataSource: isDataSource}),
		TEMPORARY: {
			Type:        schema.TypeBool,
			Required:    !isDataSource,
			Computed:    isDataSource,
			Description: "Whether new flags should be temporary by default.",
		},
		BOOLEAN_DEFAULTS: {
			Type:        schema.TypeList,
			Required:    !isDataSource,
			Computed:    isDataSource,
			MaxItems:    1,
			Description: "A block describing the default boolean flag variation settings.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					TRUE_DISPLAY_NAME: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The display name for the true variation.",
					},
					FALSE_DISPLAY_NAME: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The display name for the false variation.",
					},
					TRUE_DESCRIPTION: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The description for the true variation.",
					},
					FALSE_DESCRIPTION: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The description for the false variation.",
					},
					ON_VARIATION: {
						Type:             schema.TypeInt,
						Required:         true,
						Description:      "The variation index of the boolean flag variation to serve when the flag's targeting is on.",
						ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 1)),
					},
					OFF_VARIATION: {
						Type:             schema.TypeInt,
						Required:         true,
						Description:      "The variation index of the boolean flag variation to serve when the flag's targeting is off.",
						ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 1)),
					},
				},
			},
		},
	}
}

func flagDefaultsRead(ctx context.Context, d *schema.ResourceData, meta interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)

	var flagDefaults *ldapi.FlagDefaultsRep
	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		flagDefaults, res, err = client.ld.ProjectsApi.GetFlagDefaultsByProject(client.ctx, projectKey).Execute()
		return err
	})

	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[WARN] failed to find flag defaults for project %q, removing from state if present", projectKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find flag defaults for project %q, removing from state if present", projectKey),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get flag defaults for project %q: %s", projectKey, handleLdapiErr(err))
	}

	if isDataSource {
		d.SetId(projectKey)
	}

	_ = d.Set(PROJECT_KEY, projectKey)

	if flagDefaults.Tags != nil {
		err = d.Set(TAGS, flagDefaults.Tags)
		if err != nil {
			return diag.Errorf("failed to set tags on flag defaults for project %q: %v", projectKey, err)
		}
	}

	if flagDefaults.Temporary != nil {
		_ = d.Set(TEMPORARY, *flagDefaults.Temporary)
	}

	if flagDefaults.BooleanDefaults != nil {
		bd := flagDefaults.BooleanDefaults
		bdMap := []map[string]interface{}{{
			TRUE_DISPLAY_NAME:  bd.GetTrueDisplayName(),
			FALSE_DISPLAY_NAME: bd.GetFalseDisplayName(),
			TRUE_DESCRIPTION:   bd.GetTrueDescription(),
			FALSE_DESCRIPTION:  bd.GetFalseDescription(),
			ON_VARIATION:       int(bd.GetOnVariation()),
			OFF_VARIATION:      int(bd.GetOffVariation()),
		}}
		err = d.Set(BOOLEAN_DEFAULTS, bdMap)
		if err != nil {
			return diag.Errorf("failed to set boolean_defaults on flag defaults for project %q: %v", projectKey, err)
		}
	}

	return diags
}

// getCurrentCSA reads the current default_client_side_availability from the API
// so it can be passed through unchanged on PUT requests. This avoids conflicting
// with the launchdarkly_project resource which owns CSA settings.
func getCurrentCSA(client *Client, projectKey string) (*ldapi.DefaultClientSideAvailability, error) {
	var flagDefaults *ldapi.FlagDefaultsRep
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		flagDefaults, _, err = client.ld.ProjectsApi.GetFlagDefaultsByProject(client.ctx, projectKey).Execute()
		return err
	})
	if err != nil {
		return nil, err
	}

	// Convert from the GET response type (pointer fields) to the PUT request type (value fields)
	csa := ldapi.NewDefaultClientSideAvailability(false, false)
	if flagDefaults.DefaultClientSideAvailability != nil {
		if flagDefaults.DefaultClientSideAvailability.UsingMobileKey != nil {
			csa.UsingMobileKey = *flagDefaults.DefaultClientSideAvailability.UsingMobileKey
		}
		if flagDefaults.DefaultClientSideAvailability.UsingEnvironmentId != nil {
			csa.UsingEnvironmentId = *flagDefaults.DefaultClientSideAvailability.UsingEnvironmentId
		}
	}

	return csa, nil
}

func flagDefaultsPayloadFromResourceData(d *schema.ResourceData, csa ldapi.DefaultClientSideAvailability) ldapi.UpsertFlagDefaultsPayload {
	tags := stringsFromResourceData(d, TAGS)
	temporary := d.Get(TEMPORARY).(bool)

	trueDisplayName := d.Get(fmt.Sprintf("%s.0.%s", BOOLEAN_DEFAULTS, TRUE_DISPLAY_NAME)).(string)
	falseDisplayName := d.Get(fmt.Sprintf("%s.0.%s", BOOLEAN_DEFAULTS, FALSE_DISPLAY_NAME)).(string)
	trueDescription := d.Get(fmt.Sprintf("%s.0.%s", BOOLEAN_DEFAULTS, TRUE_DESCRIPTION)).(string)
	falseDescription := d.Get(fmt.Sprintf("%s.0.%s", BOOLEAN_DEFAULTS, FALSE_DESCRIPTION)).(string)
	onVariation := int32(d.Get(fmt.Sprintf("%s.0.%s", BOOLEAN_DEFAULTS, ON_VARIATION)).(int))
	offVariation := int32(d.Get(fmt.Sprintf("%s.0.%s", BOOLEAN_DEFAULTS, OFF_VARIATION)).(int))

	return *ldapi.NewUpsertFlagDefaultsPayload(
		tags,
		temporary,
		*ldapi.NewBooleanFlagDefaults(
			trueDisplayName,
			falseDisplayName,
			trueDescription,
			falseDescription,
			onVariation,
			offVariation,
		),
		csa,
	)
}

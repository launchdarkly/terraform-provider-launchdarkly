package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceReleasePolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceReleasePolicyCreate,
		ReadContext:   resourceReleasePolicyRead,
		UpdateContext: resourceReleasePolicyUpdate,
		DeleteContext: resourceReleasePolicyDelete,
		Exists:        resourceReleasePolicyExists,

		Importer: &schema.ResourceImporter{
			StateContext: resourceReleasePolicyImport,
		},

		Description: `Provides a LaunchDarkly release policy resource. This resource is still in beta.

This resource allows you to create and manage release policies within your LaunchDarkly organization.

Learn more about [release policies here](https://launchdarkly.com/docs/home/releases/release-policies), and read our [API docs here](https://launchdarkly.com/docs/api/release-policies-beta/).`,

		Schema: map[string]*schema.Schema{
			PROJECT_KEY: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      addForceNewDescription("The project key.", true),
				ForceNew:         true,
				ValidateDiagFunc: validateKeyAndLength(1, 140),
			},
			KEY: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      addForceNewDescription("The human-readable key of the release policy.", true),
				ForceNew:         true,
				ValidateDiagFunc: validateKeyAndLength(1, 256),
			},
			NAME: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The name of the release policy. Maximum length is 256 characters.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(1, 256)),
			},
			RELEASE_METHOD: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The release method for the release policy. Must be either 'guarded-release' or 'progressive-release'.",
				ValidateDiagFunc: validateReleaseMethod(),
			},
			SCOPE: {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "The scope configuration for the release policy.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						SCOPE_ENVIRONMENT_KEYS: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "The environment keys for environments the release policy will be applied to.",
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								ValidateDiagFunc: validateKeyAndLength(1, 100),
							},
						},
						SCOPE_FLAG_TAG_KEYS: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "The flag tag keys that the release policy will be applied to.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			GUARDED_RELEASE_CONFIG: {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Configuration for guarded release.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						ROLLBACK_ON_REGRESSION: {
							Type:        schema.TypeBool,
							Optional:    false,
							Required:    true,
							Description: "Whether to automatically rollback on regression.",
						},
						MIN_SAMPLE_SIZE: {
							Type:             schema.TypeInt,
							Optional:         true,
							Required:         false,
							Default:          0, // This is so we "know" if the user set it or not
							Description:      "The minimum sample size for the release policy.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(5)),
						},
					},
				},
			},
		},
	}
}

func resourceReleasePolicyCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	policyKey := d.Get(KEY).(string)

	releasePolicyPost := resourceDataToAPIBody(d, policyKey)

	_, err := createReleasePolicy(client, projectKey, releasePolicyPost)
	if err != nil {
		return diag.Errorf("failed to create release policy with key %q in project %q: %s", policyKey, projectKey, handleLdapiErr(err))
	}

	d.SetId(releasePolicyID(projectKey, policyKey))

	return resourceReleasePolicyRead(ctx, d, metaRaw)
}

func resourceReleasePolicyRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return releasePolicyRead(ctx, d, metaRaw, false)
}

func resourceReleasePolicyUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	policyKey := d.Get(KEY).(string)

	// If the key or project key changes then delete the old resource and create a new one
	if d.HasChange(KEY) || d.HasChange(PROJECT_KEY) {
		oldKey, _ := d.GetChange(KEY)
		oldProjectKey, _ := d.GetChange(PROJECT_KEY)
		err := deleteReleasePolicy(client, oldProjectKey.(string), oldKey.(string))
		if err != nil {
			return diag.Errorf("failed to delete release policy with key %q in project %q: %s", policyKey, projectKey, handleLdapiErr(err))
		}

		return resourceReleasePolicyCreate(ctx, d, metaRaw)
	}

	if d.HasChange(NAME) || d.HasChange(RELEASE_METHOD) || d.HasChange(SCOPE) || d.HasChange(GUARDED_RELEASE_CONFIG) {
		updatedPolicy := resourceDataToAPIBody(d, policyKey)
		err := putReleasePolicy(client, projectKey, policyKey, updatedPolicy)
		if err != nil {
			return diag.Errorf("failed to update release policy with key %q in project %q: %s", policyKey, projectKey, handleLdapiErr(err))
		}
	}

	return resourceReleasePolicyRead(ctx, d, metaRaw)
}

func resourceReleasePolicyDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	policyKey := d.Get(KEY).(string)

	err := deleteReleasePolicy(client, projectKey, policyKey)
	if err != nil {
		return diag.Errorf("failed to delete release policy with key %q in project %q: %s", policyKey, projectKey, handleLdapiErr(err))
	}

	return diags
}

func resourceReleasePolicyExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	return releasePolicyExists(d.Get(PROJECT_KEY).(string), d.Get(KEY).(string), client)
}

func releasePolicyExists(projectKey, policyKey string, client *Client) (bool, error) {
	_, res, err := getReleasePolicyRaw(client, projectKey, policyKey)
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get release policy with key %q in project %q: %s", policyKey, projectKey, handleLdapiErr(err))
	}

	return true, nil
}

func resourceReleasePolicyImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()
	if id == "" {
		return nil, fmt.Errorf("import ID cannot be empty")
	}

	parts := splitID(id, 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("import ID must be in the format project_key/release_policy_key")
	}

	projectKey, policyKey := parts[0], parts[1]
	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(KEY, policyKey)

	return []*schema.ResourceData{d}, nil
}

// releasePolicyID constructs the ID for a release policy resource
func releasePolicyID(projectKey, policyKey string) string {
	return fmt.Sprintf("%s/%s", projectKey, policyKey)
}

// convertScopeToAPI converts Terraform scope data to API format
func convertScopeToAPI(scopeData map[string]interface{}) map[string]interface{} {
	scopeAPI := make(map[string]interface{})
	if envKeys, ok := scopeData[SCOPE_ENVIRONMENT_KEYS]; ok {
		scopeAPI["environmentKeys"] = envKeys
	}
	if flagTagKeys, ok := scopeData[SCOPE_FLAG_TAG_KEYS]; ok {
		keys := flagTagKeys.([]interface{})
		if len(keys) > 0 {
			scopeAPI["flagTagKeys"] = keys
		}
	}
	return scopeAPI
}

// convertGuardedConfigToAPI converts Terraform guarded config data to API format
func convertGuardedConfigToAPI(config map[string]interface{}) map[string]interface{} {
	guardedConfigAPI := make(map[string]interface{})

	if rollbackOnRegression, ok := config[ROLLBACK_ON_REGRESSION]; ok {
		guardedConfigAPI["rollbackOnRegression"] = rollbackOnRegression
	}

	if minSampleSize, ok := config[MIN_SAMPLE_SIZE]; ok {
		// I don't think it's possible for this to be unset if the user doesn't provide it, so...if they set it to 0 it will be ignored.
		if minSampleSize.(int) > 0 {
			guardedConfigAPI["minSampleSize"] = minSampleSize
		}
	}

	return guardedConfigAPI
}

// resourceDataToAPIBody converts Terraform resource data to the API request body format
func resourceDataToAPIBody(d *schema.ResourceData, policyKey string) map[string]interface{} {
	releasePolicyPost := map[string]interface{}{
		"key":           policyKey,
		"name":          d.Get(NAME).(string),
		"releaseMethod": d.Get(RELEASE_METHOD).(string),
	}

	if scope, ok := d.GetOk(SCOPE); ok {
		scopeList := scope.([]interface{})
		if len(scopeList) > 0 {
			scopeData := scopeList[0].(map[string]interface{})
			releasePolicyPost["scope"] = convertScopeToAPI(scopeData)
		}
	}

	if guardedConfig, ok := d.GetOk(GUARDED_RELEASE_CONFIG); ok {
		configList := guardedConfig.([]interface{})
		if len(configList) > 0 {
			config := configList[0].(map[string]interface{})
			releasePolicyPost["guardedReleaseConfig"] = convertGuardedConfigToAPI(config)
		}
	}

	return releasePolicyPost
}

package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v14"
)

func resourceDestination() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDestinationCreate,
		ReadContext:   resourceDestinationRead,
		UpdateContext: resourceDestinationUpdate,
		DeleteContext: resourceDestinationDelete,
		Exists:        resourceDestinationExists,

		Importer: &schema.ResourceImporter{
			State: resourceDestinationImport,
		},

		Description: `Provides a LaunchDarkly Data Export Destination resource.

-> **Note:** Data Export is available to customers on an Enterprise LaunchDarkly plan. To learn more, read about our pricing. To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

Data Export Destinations are locations that receive exported data. This resource allows you to configure destinations for the export of raw analytics data, including feature flag requests, analytics events, custom events, and more.

To learn more about data export, read [Data Export Documentation](https://docs.launchdarkly.com/integrations/data-export).`,

		Schema: map[string]*schema.Schema{
			PROJECT_KEY: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      addForceNewDescription("The LaunchDarkly project key.", true),
				ValidateDiagFunc: validateKey(),
			},
			ENV_KEY: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: addForceNewDescription("The environment key.", true),
			},
			NAME: {
				Type:        schema.TypeString,
				Description: "A human-readable name for your data export destination.",
				Required:    true,
			},
			// kind can only be one of five types (kinesis, google-pubsub, mparticle, azure-event-hubs, or segment)
			KIND: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      addForceNewDescription("The data export destination type. Available choices are `kinesis`, `google-pubsub`, `mparticle`, `azure-event-hubs`, and `segment`.", true),
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"kinesis", "google-pubsub", "mparticle", "azure-event-hubs", "segment"}, false)),
				ForceNew:         true,
			},
			CONFIG: {
				Type:             schema.TypeMap,
				Required:         true,
				Description:      "The destination-specific configuration. To learn more, read [Destination-Specific Configs](#destination-specific-configs)",
				Elem:             &schema.Schema{Type: schema.TypeString},
				DiffSuppressFunc: configDiffSuppressFunc(),
			},
			ON: {
				Type:        schema.TypeBool,
				Description: "Whether the data export destination is on or not.",
				Optional:    true,
			},
			TAGS: tagsSchema(tagsSchemaOptions{isDataSource: false}),
		},
	}
}

func resourceDestinationCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	destinationProjKey := d.Get(PROJECT_KEY).(string)
	destinationEnvKey := d.Get(ENV_KEY).(string)
	destinationName := d.Get(NAME).(string)
	destinationKind := d.Get(KIND).(string)
	destinationOn := d.Get(ON).(bool)

	destinationConfig, err := destinationConfigFromResourceData(d)
	if err != nil {
		return diag.FromErr(err)
	}

	destinationBody := ldapi.DestinationPost{
		Name:   &destinationName,
		Kind:   &destinationKind,
		Config: &destinationConfig,
		On:     &destinationOn,
	}

	destination, _, err := client.ld.DataExportDestinationsApi.PostDestination(client.ctx, destinationProjKey, destinationEnvKey).DestinationPost(destinationBody).Execute()
	if err != nil {
		d.SetId("")
		return diag.Errorf("failed to create destination with project key %q and env key %q: %s", destinationProjKey, destinationEnvKey, handleLdapiErr(err))
	}

	// destination defined in api-client-go/model_destination.go
	d.SetId(strings.Join([]string{destinationProjKey, destinationEnvKey, *destination.Id}, "/"))

	return resourceDestinationRead(ctx, d, metaRaw)
}

func resourceDestinationRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := metaRaw.(*Client)
	_, _, destinationID, err := destinationImportIDtoKeys(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	destinationProjKey := d.Get(PROJECT_KEY).(string)
	destinationEnvKey := d.Get(ENV_KEY).(string)

	destination, res, err := client.ld.DataExportDestinationsApi.GetDestination(client.ctx, destinationProjKey, destinationEnvKey, destinationID).Execute()

	if isStatusNotFound(res) {
		log.Printf("[WARN] failed to find destination with id: %q in project %q, environment: %q, removing from state", destinationID, destinationProjKey, destinationEnvKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find destination with id: %q in project %q, environment: %q, removing from state", destinationID, destinationProjKey, destinationEnvKey),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get destination with id %q: %s", destinationID, handleLdapiErr(err))
	}

	cfg := destinationConfigToResourceData(*destination.Kind, destination.Config)
	preservedCfg := preserveObfuscatedConfigAttributes(d.Get(CONFIG).(map[string]interface{}), cfg)

	_ = d.Set(NAME, destination.Name)
	_ = d.Set(KIND, destination.Kind)
	_ = d.Set(CONFIG, preservedCfg)
	_ = d.Set(ON, destination.On)

	d.SetId(strings.Join([]string{destinationProjKey, destinationEnvKey, *destination.Id}, "/"))
	return diags
}

func resourceDestinationUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	_, _, destinationID, err := destinationImportIDtoKeys(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	destinationProjKey := d.Get(PROJECT_KEY).(string)
	destinationEnvKey := d.Get(ENV_KEY).(string)
	destinationName := d.Get(NAME).(string)
	destinationKind := d.Get(KIND).(string)
	destinationConfig, err := destinationConfigFromResourceData(d)
	if err != nil {
		return diag.FromErr(err)
	}
	destinationOn := d.Get(ON).(bool)

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &destinationName),
		patchReplace("/kind", &destinationKind),
		patchReplace("/on", &destinationOn),
		patchReplace("/config", &destinationConfig),
	}

	_, _, err = client.ld.DataExportDestinationsApi.PatchDestination(client.ctx, destinationProjKey, destinationEnvKey, destinationID).PatchOperation(patch).Execute()
	if err != nil {
		return diag.Errorf("failed to update destination with id %q: %s", destinationID, handleLdapiErr(err))
	}

	return resourceDestinationRead(ctx, d, metaRaw)
}

func resourceDestinationDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	_, _, destinationID, err := destinationImportIDtoKeys(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	destinationProjKey := d.Get(PROJECT_KEY).(string)
	destinationEnvKey := d.Get(ENV_KEY).(string)

	_, err = client.ld.DataExportDestinationsApi.DeleteDestination(client.ctx, destinationProjKey, destinationEnvKey, destinationID).Execute()
	if err != nil {
		return diag.Errorf("failed to delete destination with id %q: %s", destinationID, handleLdapiErr(err))
	}

	return diags
}

func resourceDestinationExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	_, _, destinationID, err := destinationImportIDtoKeys(d.Id())
	if err != nil {
		return false, err
	}
	destinationProjKey := d.Get(PROJECT_KEY).(string)
	destinationEnvKey := d.Get(ENV_KEY).(string)

	_, res, err := client.ld.DataExportDestinationsApi.GetDestination(client.ctx, destinationProjKey, destinationEnvKey, destinationID).Execute()
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get destination with id %q: %s", destinationID, handleLdapiErr(err))
	}

	return true, nil
}

func resourceDestinationImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	projKey, envKey, _, err := destinationImportIDtoKeys(d.Id())
	if err != nil {
		return nil, err
	}

	_ = d.Set(PROJECT_KEY, projKey)
	_ = d.Set(ENV_KEY, envKey)

	return []*schema.ResourceData{d}, nil
}

func destinationImportIDtoKeys(importID string) (projKey, envKey, destinationID string, err error) {
	if strings.Count(importID, "/") != 2 {
		return "", "", "", fmt.Errorf("found unexpected destination import id format: %q expected format: 'project_key/env_key/destination_id'", importID)
	}
	parts := strings.SplitN(importID, "/", 3)
	projKey, envKey, destinationID = parts[0], parts[1], parts[2]
	return projKey, envKey, destinationID, nil
}

func configDiffSuppressFunc() schema.SchemaDiffSuppressFunc {
	return func(k, old, new string, d *schema.ResourceData) bool {
		if d.Get(KIND).(string) == "mparticle" {
			// ignore changes to user_identity if user_identities is set
			if k == fmt.Sprintf("%s.user_identity", CONFIG) {
				if _, ok := d.GetOk(fmt.Sprintf("%s.user_identities", CONFIG)); ok {
					return true
				}
			}
		}
		return old == new
	}
}

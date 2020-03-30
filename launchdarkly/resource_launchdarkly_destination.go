package launchdarkly

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func resourceDestination() *schema.Resource {
	return &schema.Resource{
		Create: resourceDestinationCreate,
		Read:   resourceDestinationRead,
		Update: resourceDestinationUpdate,
		Delete: resourceDestinationDelete,
		Exists: resourceDestinationExists,

		Importer: &schema.ResourceImporter{
			State: resourceDestinationImport,
		},

		Schema: map[string]*schema.Schema{
			PROJECT_KEY: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The LaunchDarkly project key",
				ValidateFunc: validateKey(),
			},
			ENV_KEY: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The LaunchDarkly environment key",
			},
			NAME: {
				Type:        schema.TypeString,
				Description: "a human-readable name for your data export destination",
				Required:    true,
			},
			// kind can only be one of three types (kinesis, google-pubsub, or mparticle)
			KIND: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The data export destination type - must be 'kinesis', 'google-pubsub', or 'mparticle'",
				ValidateFunc: validateDestinationKind(),
				ForceNew:     true,
			},
			CONFIG: {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"google-pubsub": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"project": {
										Type:     schema.TypeString,
										Required: true,
									},
									"topic": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"kinesis": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"region": {
										Type:     schema.TypeString,
										Required: true,
									},
									"role_arn": {
										Type:     schema.TypeString,
										Required: true,
									},
									"stream_name": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"mparticle": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"api_key": {
										Type:      schema.TypeString,
										Required:  true,
										Sensitive: true,
									},
									"secret": {
										Type:      schema.TypeString,
										Required:  true,
										Sensitive: true,
									},
									"user_identity": {
										Type:     schema.TypeString,
										Required: true,
									},
									"environment": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},

			ENABLED: {
				Type:     schema.TypeBool,
				Optional: true,
			},
			TAGS: tagsSchema(),
		},
	}
}

func resourceDestinationCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	destinationProjKey := d.Get(PROJECT_KEY).(string)
	destinationEnvKey := d.Get(ENV_KEY).(string)
	destinationName := d.Get(NAME).(string)
	destinationKind := d.Get(KIND).(string)
	destinationOn := d.Get(ENABLED).(bool)

	destinationConfig, err := destinationConfigFromResourceData(d)
	if err != nil {
		return err
	}

	destinationBody := ldapi.DestinationBody{
		Name:   destinationName,
		Kind:   destinationKind,
		Config: &destinationConfig,
		On:     destinationOn,
	}

	destination, _, err := client.ld.DataExportDestinationsApi.PostDestination(client.ctx, destinationProjKey, destinationEnvKey, destinationBody)
	if err != nil {
		return fmt.Errorf("failed to create destination with project key %q and env key %q: %s", destinationProjKey, destinationEnvKey, handleLdapiErr(err))
	}

	// destination defined in api-client-go/model_destination.go
	d.SetId(destination.Id)

	return resourceDestinationRead(d, metaRaw)
}

func resourceDestinationRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	destinationID := d.Id()
	destinationProjKey := d.Get(PROJECT_KEY).(string)
	destinationEnvKey := d.Get(ENV_KEY).(string)

	destination, res, err := client.ld.DataExportDestinationsApi.GetDestination(client.ctx, destinationProjKey, destinationEnvKey, destinationID)
	if isStatusNotFound(res) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get destination with id %q: %s", destinationID, handleLdapiErr(err))
	}

	cfg := destinationConfigToResourceData(destination.Kind, *destination.Config)
	preservedCfg := preserveObfuscatedConfigAttributes(d.Get(CONFIG).(map[string]interface{}), cfg)

	_ = d.Set(NAME, destination.Name)
	_ = d.Set(KIND, destination.Kind)
	_ = d.Set(CONFIG, preservedCfg)
	_ = d.Set(ENABLED, destination.On)

	return nil
}

func resourceDestinationUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	destinationID := d.Id()
	destinationProjKey := d.Get(PROJECT_KEY).(string)
	destinationEnvKey := d.Get(ENV_KEY).(string)
	destinationName := d.Get(NAME).(string)
	destinationKind := d.Get(KIND).(string)
	destinationConfig, err := destinationConfigFromResourceData(d)
	if err != nil {
		return err
	}
	destinationOn := d.Get(ENABLED).(bool)

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &destinationName),
		patchReplace("/kind", &destinationKind),
		patchReplace("/on", &destinationOn),
		patchReplace("/config", &destinationConfig),
	}

	_, _, err = repeatUntilNoConflict((func() (interface{}, *http.Response, error) {
		return client.ld.DataExportDestinationsApi.PatchDestination(client.ctx, destinationProjKey, destinationEnvKey, destinationID, patch)
	}))
	if err != nil {
		return fmt.Errorf("failed to update destination with id %q: %s", destinationID, handleLdapiErr(err))
	}

	return resourceDestinationRead(d, metaRaw)
}

func resourceDestinationDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	destinationID := d.Id()
	destinationProjKey := d.Get(PROJECT_KEY).(string)
	destinationEnvKey := d.Get(ENV_KEY).(string)

	_, err := client.ld.DataExportDestinationsApi.DeleteDestination(client.ctx, destinationProjKey, destinationEnvKey, destinationID)
	if err != nil {
		return fmt.Errorf("failed to delete destination with id %q: %s", destinationID, handleLdapiErr(err))
	}

	return nil
}

func resourceDestinationExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	destinationID := d.Id()
	destinationProjKey := d.Get(PROJECT_KEY).(string)
	destinationEnvKey := d.Get(ENV_KEY).(string)

	_, res, err := client.ld.DataExportDestinationsApi.GetDestination(client.ctx, destinationProjKey, destinationEnvKey, destinationID)
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get destination with id %q: %s", destinationID, handleLdapiErr(err))
	}

	return true, nil
}

func resourceDestinationImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.SetId(d.Id())
	err := resourceDestinationRead(d, meta)
	if err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

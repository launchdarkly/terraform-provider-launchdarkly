package launchdarkly

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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
				ForceNew:    true,
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
				Description:  "The data export destination type - must be 'kinesis', 'google-pubsub', 'mparticle', or 'segment'",
				ValidateFunc: validation.StringInSlice([]string{"kinesis", "google-pubsub", "mparticle", "segment"}, false),
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
						"segment": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"write_key": {
										Type:      schema.TypeString,
										Required:  true,
										Sensitive: true,
										ForceNew:  true,
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

	destinationRaw, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.DataExportDestinationsApi.PostDestination(client.ctx, destinationProjKey, destinationEnvKey, destinationBody)
	})
	destination := destinationRaw.(ldapi.Destination)
	if err != nil {
		d.SetId("")
		return fmt.Errorf("failed to create destination with project key %q and env key %q: %s", destinationProjKey, destinationEnvKey, handleLdapiErr(err))
	}

	// destination defined in api-client-go/model_destination.go
	d.SetId(strings.Join([]string{destinationProjKey, destinationEnvKey, destination.Id}, "/"))

	return resourceDestinationRead(d, metaRaw)
}

func resourceDestinationRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	_, _, destinationID, err := destinationImportIDtoKeys(d.Id())
	if err != nil {
		return err
	}

	destinationProjKey := d.Get(PROJECT_KEY).(string)
	destinationEnvKey := d.Get(ENV_KEY).(string)

	destinationRaw, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.DataExportDestinationsApi.GetDestination(client.ctx, destinationProjKey, destinationEnvKey, destinationID)
	})
	destination := destinationRaw.(ldapi.Destination)
	if isStatusNotFound(res) {
		log.Printf("[WARN] failed to find destination with id: %q in project %q, environment: %q, removing from state", destinationID, destinationProjKey, destinationEnvKey)
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

	d.SetId(strings.Join([]string{destinationProjKey, destinationEnvKey, destination.Id}, "/"))
	return nil
}

func resourceDestinationUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	_, _, destinationID, err := destinationImportIDtoKeys(d.Id())
	if err != nil {
		return err
	}
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

	_, _, err = handleRateLimit(func() (interface{}, *http.Response, error) {
		return handleNoConflict((func() (interface{}, *http.Response, error) {
			return client.ld.DataExportDestinationsApi.PatchDestination(client.ctx, destinationProjKey, destinationEnvKey, destinationID, patch)
		}))
	})
	if err != nil {
		return fmt.Errorf("failed to update destination with id %q: %s", destinationID, handleLdapiErr(err))
	}

	return resourceDestinationRead(d, metaRaw)
}

func resourceDestinationDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	_, _, destinationID, err := destinationImportIDtoKeys(d.Id())
	if err != nil {
		return err
	}
	destinationProjKey := d.Get(PROJECT_KEY).(string)
	destinationEnvKey := d.Get(ENV_KEY).(string)

	_, _, err = handleRateLimit(func() (interface{}, *http.Response, error) {
		res, err := client.ld.DataExportDestinationsApi.DeleteDestination(client.ctx, destinationProjKey, destinationEnvKey, destinationID)
		return nil, res, err
	})

	if err != nil {
		return fmt.Errorf("failed to delete destination with id %q: %s", destinationID, handleLdapiErr(err))
	}

	return nil
}

func resourceDestinationExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	_, _, destinationID, err := destinationImportIDtoKeys(d.Id())
	if err != nil {
		return false, err
	}
	destinationProjKey := d.Get(PROJECT_KEY).(string)
	destinationEnvKey := d.Get(ENV_KEY).(string)

	_, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.DataExportDestinationsApi.GetDestination(client.ctx, destinationProjKey, destinationEnvKey, destinationID)
	})
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

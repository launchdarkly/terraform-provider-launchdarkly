package launchdarkly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// AiConfig represents the AI Config response from the LaunchDarkly API.
type AiConfig struct {
	Key                 string              `json:"key"`
	Name                string              `json:"name"`
	Description         string              `json:"description,omitempty"`
	Tags                []string            `json:"tags,omitempty"`
	Mode                string              `json:"mode,omitempty"`
	Version             int                 `json:"version,omitempty"`
	CreatedAt           int64               `json:"createdAt,omitempty"`
	UpdatedAt           int64               `json:"updatedAt,omitempty"`
	Variations          []AiConfigVariation `json:"variations,omitempty"`
	Maintainer          *AiConfigMaintainer `json:"_maintainer,omitempty"`
	EvaluationMetricKey string              `json:"evaluationMetricKey,omitempty"`
	IsInverted          *bool               `json:"isInverted,omitempty"`
}

// AiConfigVariation represents a variation within an AI Config.
type AiConfigVariation struct {
	ID   string `json:"_id,omitempty"`
	Key  string `json:"key"`
	Name string `json:"name,omitempty"`
}

// AiConfigMaintainer represents the maintainer of an AI Config.
type AiConfigMaintainer struct {
	ID    string `json:"_id,omitempty"`
	Email string `json:"email,omitempty"`
	Kind  string `json:"kind,omitempty"`
}

// AiConfigPost is the request body for creating an AI Config.
type AiConfigPost struct {
	Key                 string   `json:"key"`
	Name                string   `json:"name"`
	Description         string   `json:"description,omitempty"`
	MaintainerId        string   `json:"maintainerId,omitempty"`
	MaintainerTeamKey   string   `json:"maintainerTeamKey,omitempty"`
	Mode                string   `json:"mode,omitempty"`
	Tags                []string `json:"tags,omitempty"`
	EvaluationMetricKey string   `json:"evaluationMetricKey,omitempty"`
	IsInverted          *bool    `json:"isInverted,omitempty"`
}

// AiConfigPatch is the request body for updating an AI Config.
type AiConfigPatch struct {
	Name                *string  `json:"name,omitempty"`
	Description         *string  `json:"description,omitempty"`
	MaintainerId        *string  `json:"maintainerId,omitempty"`
	MaintainerTeamKey   *string  `json:"maintainerTeamKey,omitempty"`
	Tags                []string `json:"tags,omitempty"`
	EvaluationMetricKey *string  `json:"evaluationMetricKey,omitempty"`
	IsInverted          *bool    `json:"isInverted,omitempty"`
}

func baseAiConfigSchema(isDataSource bool) map[string]*schema.Schema {
	schemaMap := map[string]*schema.Schema{
		PROJECT_KEY: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			Description:      addForceNewDescription("The AI Config's project key.", !isDataSource),
			ValidateDiagFunc: validateKey(),
		},
		KEY: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validateKey(),
			Description:      addForceNewDescription("The unique key that references the AI Config.", !isDataSource),
		},
		NAME: {
			Type:        schema.TypeString,
			Required:    !isDataSource,
			Computed:    isDataSource,
			Description: "The human-friendly name for the AI Config.",
		},
		DESCRIPTION: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "The description of the AI Config.",
		},
		TAGS: tagsSchema(tagsSchemaOptions{isDataSource: isDataSource}),
		MODE: {
			Type:             schema.TypeString,
			Optional:         !isDataSource,
			Computed:         true,
			Description:      "The mode for the AI Config. Available choices are `agent`, `completion`, and `judge`. Defaults to `completion`.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"agent", "completion", "judge"}, false)),
		},
		MAINTAINER_ID: {
			Type:             schema.TypeString,
			Optional:         !isDataSource,
			Computed:         true,
			Description:      "The LaunchDarkly member ID of the member who will maintain the AI Config. If not set, the API will automatically apply the member associated with your Terraform API key or the most recently-set maintainer.",
			ValidateDiagFunc: validateID(),
		},
		MAINTAINER_TEAM_KEY: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "The key of the team that will maintain the AI Config.",
		},
		EVALUATION_METRIC_KEY: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "The evaluation metric key for this AI Config.",
		},
		IS_INVERTED: {
			Type:        schema.TypeBool,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "Whether the evaluation metric is inverted, meaning a lower value is better if set as true.",
		},
		VERSION: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The version of the AI Config.",
		},
	}

	if isDataSource {
		return removeInvalidFieldsForDataSource(schemaMap)
	}

	return schemaMap
}

// setAiConfigRequestHeaders sets common headers for AI Config API requests.
func setAiConfigRequestHeaders(req *http.Request, apiKey string) {
	req.Header.Set("Authorization", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", APIVersion)
	req.Header.Set("User-Agent", fmt.Sprintf("launchdarkly-terraform-provider/%s", version))
}

// aiConfigAPIURL builds the URL for AI Config API requests.
func aiConfigAPIURL(client *Client, projectKey string, configKey ...string) string {
	host := client.apiHost
	if host == "" {
		host = "app.launchdarkly.com"
	}
	base := fmt.Sprintf("https://%s/api/v2/projects/%s/ai-configs", host, projectKey)
	if len(configKey) > 0 && configKey[0] != "" {
		base = fmt.Sprintf("%s/%s", base, configKey[0])
	}
	return base
}

func createAiConfig(client *Client, projectKey string, post AiConfigPost) (*AiConfig, error) {
	body, err := json.Marshal(post)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal AI Config create request: %w", err)
	}

	url := aiConfigAPIURL(client, projectKey)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create AI Config request: %w", err)
	}
	setAiConfigRequestHeaders(req, client.apiKey)

	var aiConfig AiConfig
	err = client.withConcurrency(client.ctx, func() error {
		resp, httpErr := client.fallbackClient.Do(req)
		if httpErr != nil {
			return fmt.Errorf("failed to create AI Config: %w", httpErr)
		}
		defer func() { _ = resp.Body.Close() }()

		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("failed to read AI Config create response: %w", readErr)
		}

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to create AI Config, status %d: %s", resp.StatusCode, string(respBody))
		}

		return json.Unmarshal(respBody, &aiConfig)
	})
	if err != nil {
		return nil, err
	}

	return &aiConfig, nil
}

func getAiConfig(client *Client, projectKey, configKey string) (*AiConfig, *http.Response, error) {
	url := aiConfigAPIURL(client, projectKey, configKey)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create AI Config get request: %w", err)
	}
	setAiConfigRequestHeaders(req, client.apiKey)

	var aiConfig AiConfig
	var resp *http.Response
	err = client.withConcurrency(client.ctx, func() error {
		var httpErr error
		resp, httpErr = client.fallbackClient.Do(req)
		if httpErr != nil {
			return fmt.Errorf("failed to get AI Config: %w", httpErr)
		}
		defer func() { _ = resp.Body.Close() }()

		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("failed to read AI Config get response: %w", readErr)
		}

		if resp.StatusCode == http.StatusNotFound {
			return nil
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get AI Config, status %d: %s", resp.StatusCode, string(respBody))
		}

		return json.Unmarshal(respBody, &aiConfig)
	})
	if err != nil {
		return nil, resp, err
	}

	return &aiConfig, resp, nil
}

func patchAiConfig(client *Client, projectKey, configKey string, patch AiConfigPatch) (*AiConfig, error) {
	body, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal AI Config patch request: %w", err)
	}

	url := aiConfigAPIURL(client, projectKey, configKey)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create AI Config patch request: %w", err)
	}
	setAiConfigRequestHeaders(req, client.apiKey)

	var aiConfig AiConfig
	err = client.withConcurrency(client.ctx, func() error {
		resp, httpErr := client.fallbackClient.Do(req)
		if httpErr != nil {
			return fmt.Errorf("failed to patch AI Config: %w", httpErr)
		}
		defer func() { _ = resp.Body.Close() }()

		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("failed to read AI Config patch response: %w", readErr)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to patch AI Config, status %d: %s", resp.StatusCode, string(respBody))
		}

		return json.Unmarshal(respBody, &aiConfig)
	})
	if err != nil {
		return nil, err
	}

	return &aiConfig, nil
}

func deleteAiConfig(client *Client, projectKey, configKey string) error {
	url := aiConfigAPIURL(client, projectKey, configKey)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create AI Config delete request: %w", err)
	}
	setAiConfigRequestHeaders(req, client.apiKey)

	return client.withConcurrency(client.ctx, func() error {
		resp, httpErr := client.fallbackClient.Do(req)
		if httpErr != nil {
			return fmt.Errorf("failed to delete AI Config: %w", httpErr)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to delete AI Config, status %d: %s", resp.StatusCode, string(respBody))
		}

		return nil
	})
}

func aiConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	configKey := d.Get(KEY).(string)

	aiConfig, resp, err := getAiConfig(client, projectKey, configKey)

	if isStatusNotFound(resp) && !isDataSource {
		log.Printf("[WARN] AI Config %q in project %q not found, removing from state", configKey, projectKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] AI Config %q in project %q not found, removing from state", configKey, projectKey),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get AI Config %q in project %q: %v", configKey, projectKey, err)
	}

	_ = d.Set(KEY, aiConfig.Key)
	_ = d.Set(NAME, aiConfig.Name)
	_ = d.Set(DESCRIPTION, aiConfig.Description)

	if aiConfig.Tags != nil {
		_ = d.Set(TAGS, aiConfig.Tags)
	} else {
		_ = d.Set(TAGS, []string{})
	}

	_ = d.Set(MODE, aiConfig.Mode)
	_ = d.Set(VERSION, aiConfig.Version)
	_ = d.Set(EVALUATION_METRIC_KEY, aiConfig.EvaluationMetricKey)

	if aiConfig.IsInverted != nil {
		_ = d.Set(IS_INVERTED, *aiConfig.IsInverted)
	}

	if aiConfig.Maintainer != nil {
		if aiConfig.Maintainer.Kind == "team" {
			// For team maintainers, we don't have the key from the API response directly
			// Keep the existing value from state if it's set
		} else {
			_ = d.Set(MAINTAINER_ID, aiConfig.Maintainer.ID)
		}
	}

	d.SetId(projectKey + "/" + configKey)

	return diags
}

func aiConfigIdToKeys(id string) (projectKey string, configKey string, err error) {
	parts := splitID(id, 2)
	if parts == nil {
		return "", "", fmt.Errorf("found unexpected AI Config id format: %q expected format: 'project_key/config_key'", id)
	}
	return parts[0], parts[1], nil
}

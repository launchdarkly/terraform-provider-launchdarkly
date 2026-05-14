package launchdarkly

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// alertAPIResponse mirrors the JSON shape returned by the observability backend.
type alertAPIResponse struct {
	ID                        int                       `json:"id"`
	ProjectID                 int                       `json:"project_id"`
	Name                      string                    `json:"name"`
	MessageContent            *string                   `json:"message_content"`
	ProductType               string                    `json:"product_type"`
	FunctionType              string                    `json:"function_type"`
	FunctionColumn            *string                   `json:"function_column"`
	Query                     *string                   `json:"query"`
	GroupByKeys               []string                  `json:"group_by_keys"`
	Disabled                  bool                      `json:"disabled"`
	SlackChannels             []string                  `json:"slack_channels"`
	Emails                    []string                  `json:"emails"`
	Triggers                  []alertTriggerAPIResponse `json:"triggers"`
	ThresholdValue            *float64                  `json:"threshold_value"`
	ThresholdWindow           *int                      `json:"threshold_window"`
	ThresholdCooldown         *int                      `json:"threshold_cooldown"`
	ThresholdType             string                    `json:"threshold_type"`
	ThresholdCondition        string                    `json:"threshold_condition"`
	AutoInvestigationEnabled  bool                      `json:"auto_investigation_enabled"`
	InvestigationCooldown     *int                      `json:"investigation_cooldown"`
	InvestigationMode         *string                   `json:"investigation_mode"`
	InvestigationRepositories []string                  `json:"investigation_repositories"`
	InvestigationPrompt       *string                   `json:"investigation_prompt"`
	EvaluationDelaySeconds    *int                      `json:"evaluation_delay_seconds"`
	HideGraph                 bool                      `json:"hide_graph"`
}

type alertTriggerAPIResponse struct {
	Type       string   `json:"type"`
	Condition  string   `json:"condition"`
	InfoValue  *float64 `json:"info_value"`
	WarnValue  *float64 `json:"warn_value"`
	AlertValue *float64 `json:"alert_value"`
}

func resourceAlert() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAlertCreate,
		ReadContext:   resourceAlertRead,
		UpdateContext: resourceAlertUpdate,
		DeleteContext: resourceAlertDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceAlertImport,
		},
		Description: "Manages a LaunchDarkly Observability alert. Requires `observability_host` to be set on the provider.",
		Schema: map[string]*schema.Schema{
			PROJECT_ID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The observability project ID.",
			},
			NAME: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The display name for the alert.",
			},
			PRODUCT_TYPE: {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
				Description:      "The product type for the alert (e.g. `Errors`, `Logs`, `Traces`, `Sessions`, `Metrics`).",
			},
			FUNCTION_TYPE: {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
				Description:      "The aggregation function (e.g. `Count`, `P50`, `P75`, `P90`, `P95`, `P99`, `Max`, `Avg`, `Sum`).",
			},
			"slack_channels": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Slack channel names or IDs to notify when the alert fires (e.g. `#alerts` or `C12345ABC`). The backend resolves each value against the workspace's integrated Slack channels.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			EMAILS: {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Email addresses to notify when the alert fires.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			TRIGGERS: {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Alert trigger thresholds.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						THRESHOLD_TYPE: {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
							Description:      "Threshold type: `Constant`, `Percent`, or `PercentChange`.",
						},
						THRESHOLD_CONDITION: {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
							Description:      "Threshold condition: `Above` or `Below`.",
						},
						INFO_VALUE: {
							Type:        schema.TypeFloat,
							Optional:    true,
							Description: "Threshold value for an info-level firing.",
						},
						WARN_VALUE: {
							Type:        schema.TypeFloat,
							Optional:    true,
							Description: "Threshold value for a warn-level firing.",
						},
						ALERT_VALUE: {
							Type:        schema.TypeFloat,
							Optional:    true,
							Description: "Threshold value for an alert-level firing.",
						},
					},
				},
			},
			FUNCTION_COLUMN: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The numeric column to aggregate (e.g. `duration`).",
			},
			"query": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Search query used to filter telemetry for this alert.",
			},
			GROUP_BY_KEYS: {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Attribute keys to group results by.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			MESSAGE_CONTENT: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Custom message included in alert notifications.",
			},
			DISABLED: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "When `true`, the alert is disabled and will not fire.",
			},
			HIDE_GRAPH: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "When `true`, suppresses the inline graph image from alert notifications.",
			},
			EVALUATION_DELAY_SECONDS: {
				Type:             schema.TypeInt,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 3600)),
				Description:      "Seconds of evaluation delay (0–3600) to allow late-arriving data.",
			},
			AUTO_INVESTIGATION_ENABLED: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "When `true`, Vega automatically investigates fired alerts.",
			},
			INVESTIGATION_COOLDOWN: {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Cooldown in seconds before starting a new AI investigation (default 86400).",
			},
			INVESTIGATION_MODE: {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
				Description:      "AI investigation mode (e.g. `Investigate`, `InvestigateWithCode`, `Fix`).",
			},
			INVESTIGATION_REPOSITORIES: {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "GitHub repository slugs available to the AI investigation.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			INVESTIGATION_PROMPT: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Custom prompt prepended when starting an AI investigation.",
			},
		},
	}
}

func alertSchemaToBody(d *schema.ResourceData) map[string]interface{} {
	body := map[string]interface{}{
		"project_id":                 d.Get(PROJECT_ID).(string),
		"name":                       d.Get(NAME).(string),
		"product_type":               d.Get(PRODUCT_TYPE).(string),
		"function_type":              d.Get(FUNCTION_TYPE).(string),
		"disabled":                   d.Get(DISABLED).(bool),
		"hide_graph":                 d.Get(HIDE_GRAPH).(bool),
		"auto_investigation_enabled": d.Get(AUTO_INVESTIGATION_ENABLED).(bool),
	}

	if v, ok := d.GetOk(FUNCTION_COLUMN); ok {
		body["function_column"] = v.(string)
	}
	if v, ok := d.GetOk("query"); ok {
		body["query"] = v.(string)
	}
	if v, ok := d.GetOk(MESSAGE_CONTENT); ok {
		body["message_content"] = v.(string)
	}
	if v, ok := d.GetOk(EVALUATION_DELAY_SECONDS); ok {
		body["evaluation_delay_seconds"] = v.(int)
	}
	if v, ok := d.GetOk(INVESTIGATION_COOLDOWN); ok {
		body["investigation_cooldown"] = v.(int)
	}
	if v, ok := d.GetOk(INVESTIGATION_MODE); ok {
		body["investigation_mode"] = v.(string)
	}
	if v, ok := d.GetOk(INVESTIGATION_PROMPT); ok {
		body["investigation_prompt"] = v.(string)
	}

	if v, ok := d.GetOk(GROUP_BY_KEYS); ok {
		raw := v.([]interface{})
		keys := make([]string, len(raw))
		for i, k := range raw {
			keys[i] = k.(string)
		}
		body["group_by_keys"] = keys
	}

	if v, ok := d.GetOk(INVESTIGATION_REPOSITORIES); ok {
		raw := v.([]interface{})
		repos := make([]string, len(raw))
		for i, r := range raw {
			repos[i] = r.(string)
		}
		body["investigation_repositories"] = repos
	}

	if rawChannels, ok := d.GetOk("slack_channels"); ok {
		raw := rawChannels.([]interface{})
		channels := make([]string, len(raw))
		for i, ch := range raw {
			channels[i] = ch.(string)
		}
		body["slack_channels"] = channels
	}

	if rawEmails, ok := d.GetOk(EMAILS); ok {
		raw := rawEmails.([]interface{})
		emails := make([]string, len(raw))
		for i, e := range raw {
			emails[i] = e.(string)
		}
		body["emails"] = emails
	}

	if rawTriggers, ok := d.GetOk(TRIGGERS); ok {
		triggers := make([]map[string]interface{}, 0)
		for _, rt := range rawTriggers.([]interface{}) {
			tm := rt.(map[string]interface{})
			t := map[string]interface{}{
				"type":      tm[THRESHOLD_TYPE].(string),
				"condition": tm[THRESHOLD_CONDITION].(string),
			}
			if v, ok := tm[INFO_VALUE]; ok && v.(float64) != 0 {
				t["info_value"] = v.(float64)
			}
			if v, ok := tm[WARN_VALUE]; ok && v.(float64) != 0 {
				t["warn_value"] = v.(float64)
			}
			if v, ok := tm[ALERT_VALUE]; ok && v.(float64) != 0 {
				t["alert_value"] = v.(float64)
			}
			triggers = append(triggers, t)
		}
		body["triggers"] = triggers
	}

	return body
}

func alertResponseToState(d *schema.ResourceData, alert *alertAPIResponse) diag.Diagnostics {
	// project_id is not set from the response here — the API returns a numeric
	// project ID, while Terraform state stores the string verbose ID.  Callers
	// are responsible for setting project_id from the resource ID before calling
	// this function (see resourceAlertRead).
	if err := d.Set(NAME, alert.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(PRODUCT_TYPE, alert.ProductType); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(FUNCTION_TYPE, alert.FunctionType); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(DISABLED, alert.Disabled); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(HIDE_GRAPH, alert.HideGraph); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(AUTO_INVESTIGATION_ENABLED, alert.AutoInvestigationEnabled); err != nil {
		return diag.FromErr(err)
	}

	if alert.FunctionColumn != nil {
		if err := d.Set(FUNCTION_COLUMN, *alert.FunctionColumn); err != nil {
			return diag.FromErr(err)
		}
	}
	if alert.Query != nil {
		if err := d.Set("query", *alert.Query); err != nil {
			return diag.FromErr(err)
		}
	}
	if alert.MessageContent != nil {
		if err := d.Set(MESSAGE_CONTENT, *alert.MessageContent); err != nil {
			return diag.FromErr(err)
		}
	}
	if alert.EvaluationDelaySeconds != nil {
		if err := d.Set(EVALUATION_DELAY_SECONDS, *alert.EvaluationDelaySeconds); err != nil {
			return diag.FromErr(err)
		}
	}
	if alert.InvestigationCooldown != nil {
		if err := d.Set(INVESTIGATION_COOLDOWN, *alert.InvestigationCooldown); err != nil {
			return diag.FromErr(err)
		}
	}
	if alert.InvestigationMode != nil {
		if err := d.Set(INVESTIGATION_MODE, *alert.InvestigationMode); err != nil {
			return diag.FromErr(err)
		}
	}
	if alert.InvestigationPrompt != nil {
		if err := d.Set(INVESTIGATION_PROMPT, *alert.InvestigationPrompt); err != nil {
			return diag.FromErr(err)
		}
	}

	if len(alert.GroupByKeys) > 0 {
		if err := d.Set(GROUP_BY_KEYS, alert.GroupByKeys); err != nil {
			return diag.FromErr(err)
		}
	}
	if len(alert.InvestigationRepositories) > 0 {
		if err := d.Set(INVESTIGATION_REPOSITORIES, alert.InvestigationRepositories); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set("slack_channels", alert.SlackChannels); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(EMAILS, alert.Emails); err != nil {
		return diag.FromErr(err)
	}

	triggers := make([]map[string]interface{}, 0, len(alert.Triggers))
	for _, t := range alert.Triggers {
		tm := map[string]interface{}{
			THRESHOLD_TYPE:      t.Type,
			THRESHOLD_CONDITION: t.Condition,
		}
		if t.InfoValue != nil {
			tm[INFO_VALUE] = *t.InfoValue
		}
		if t.WarnValue != nil {
			tm[WARN_VALUE] = *t.WarnValue
		}
		if t.AlertValue != nil {
			tm[ALERT_VALUE] = *t.AlertValue
		}
		triggers = append(triggers, tm)
	}
	if err := d.Set(TRIGGERS, triggers); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func doAlertRequest(client *Client, method, path string, body interface{}) (*alertAPIResponse, int, error) {
	resp, err := client.observabilityRequest(client.ctx, method, path, body)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, resp.StatusCode, nil
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("observability API returned %d: %s", resp.StatusCode, string(raw))
	}

	var alert alertAPIResponse
	if err := json.Unmarshal(raw, &alert); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to decode alert response: %w", err)
	}
	return &alert, resp.StatusCode, nil
}

func parseAlertID(id string) (string, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("alert ID %q must be in the format {project_id}/{alert_id}", id)
	}
	alertID, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("alert ID %q has non-integer alert_id component: %w", id, err)
	}
	return parts[0], alertID, nil
}

func resourceAlertCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	body := alertSchemaToBody(d)

	alert, _, err := doAlertRequest(client, http.MethodPost, "/private/terraform/alerts", body)
	if err != nil {
		return diag.Errorf("failed to create alert: %s", err)
	}

	d.SetId(fmt.Sprintf("%s/%d", d.Get(PROJECT_ID).(string), alert.ID))
	return resourceAlertRead(ctx, d, metaRaw)
}

func resourceAlertRead(_ context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectID, alertID, err := parseAlertID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Set project_id from the resource ID so the state always has the string
	// verbose ID, not the numeric ID returned by the API.
	if err := d.Set(PROJECT_ID, projectID); err != nil {
		return diag.FromErr(err)
	}

	path := fmt.Sprintf("/private/terraform/alerts/%d?project_id=%s", alertID, projectID)
	alert, statusCode, err := doAlertRequest(client, http.MethodGet, path, nil)
	if err != nil {
		if statusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("failed to read alert %s: %s", d.Id(), err)
	}

	return alertResponseToState(d, alert)
}

func resourceAlertUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	_, alertID, err := parseAlertID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	body := alertSchemaToBody(d)
	path := fmt.Sprintf("/private/terraform/alerts/%d", alertID)

	_, _, err = doAlertRequest(client, http.MethodPut, path, body)
	if err != nil {
		return diag.Errorf("failed to update alert %s: %s", d.Id(), err)
	}

	return resourceAlertRead(ctx, d, metaRaw)
}

func resourceAlertDelete(_ context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectID, alertID, err := parseAlertID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	path := fmt.Sprintf("/private/terraform/alerts/%d?project_id=%s", alertID, projectID)
	_, _, err = doAlertRequest(client, http.MethodDelete, path, nil)
	if err != nil {
		return diag.Errorf("failed to delete alert %s: %s", d.Id(), err)
	}

	return nil
}

func resourceAlertImport(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	projectID, _, err := parseAlertID(d.Id())
	if err != nil {
		return nil, err
	}
	// Set project_id so it's in state before resourceAlertRead is called.
	if err := d.Set(PROJECT_ID, projectID); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

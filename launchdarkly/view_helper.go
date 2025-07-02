package launchdarkly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func viewRead(ctx context.Context, d *schema.ResourceData, meta interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(KEY).(string)

	view, res, err := getView(client, projectKey, viewKey)

	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[WARN] failed to find view with key %q in project %q, removing from state if present", viewKey, projectKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find view with key %q in project %q, removing from state if present", viewKey, projectKey),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get view with key %q in project %q: %v", viewKey, projectKey, err)
	}

	if isDataSource {
		d.SetId(view.Id)
	}
	_ = d.Set(PROJECT_KEY, view.ProjectKey)
	_ = d.Set(KEY, view.Key)
	_ = d.Set(NAME, view.Name)
	_ = d.Set(DESCRIPTION, view.Description)
	_ = d.Set(GENERATE_SDK_KEYS, view.GenerateSdkKeys)
	_ = d.Set(ARCHIVED, view.Archived)

	if view.Maintainer != nil {
		if view.Maintainer.Kind == "member" && view.Maintainer.MaintainerMember != nil {
			_ = d.Set(MAINTAINER_ID, view.Maintainer.MaintainerMember.Id)
			_ = d.Set(MAINTAINER_TEAM_KEY, "")
		} else if view.Maintainer.Kind == "team" && view.Maintainer.MaintainerTeam != nil {
			_ = d.Set(MAINTAINER_TEAM_KEY, view.Maintainer.MaintainerTeam.Key)
			_ = d.Set(MAINTAINER_ID, "")
		}
	} else {
		_ = d.Set(MAINTAINER_ID, "")
		_ = d.Set(MAINTAINER_TEAM_KEY, "")
	}

	err = d.Set(TAGS, view.Tags)
	if err != nil {
		return diag.Errorf("could not set tags on view with key %q: %v", view.Key, err)
	}

	return diags
}

type View struct {
	Id              string          `json:"id"`
	Key             string          `json:"key"`
	Name            string          `json:"name"`
	Description     *string         `json:"description,omitempty"`
	ProjectKey      string          `json:"projectKey"`
	GenerateSdkKeys *bool           `json:"generateSdkKeys,omitempty"`
	Archived        *bool           `json:"archived,omitempty"`
	Tags            []string        `json:"tags,omitempty"`
	Maintainer      *ViewMaintainer `json:"maintainer,omitempty"`
}

type ViewMaintainer struct {
	Kind             string                `json:"kind"`
	MaintainerMember *ViewMaintainerMember `json:"maintainerMember,omitempty"`
	MaintainerTeam   *ViewMaintainerTeam   `json:"maintainerTeam,omitempty"`
}

type ViewMaintainerMember struct {
	Id string `json:"id"`
}

type ViewMaintainerTeam struct {
	Key string `json:"key"`
}

func getView(client *Client, projectKey, viewKey string) (*View, *http.Response, error) {
	return getViewRaw(client, projectKey, viewKey)
}

func getViewRaw(client *Client, projectKey, viewKey string) (*View, *http.Response, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/views/%s", client.apiHost, projectKey, viewKey)
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}
	req, err := http.NewRequestWithContext(client.ctx, "GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.fallbackClient.Do(req)
	if err != nil {
		return nil, resp, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, resp, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var view View
	err = json.NewDecoder(resp.Body).Decode(&view)
	if err != nil {
		return nil, resp, err
	}

	return &view, resp, nil
}

func createView(client *Client, projectKey string, viewPost map[string]interface{}) (*View, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/views", client.apiHost, projectKey)
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}
	
	jsonData, err := json.Marshal(viewPost)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(client.ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.fallbackClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var view View
	err = json.NewDecoder(resp.Body).Decode(&view)
	if err != nil {
		return nil, err
	}

	return &view, nil
}

func patchView(client *Client, projectKey, viewKey string, patch []ldapi.PatchOperation) error {
	url := fmt.Sprintf("%s/api/v2/projects/%s/views/%s", client.apiHost, projectKey, viewKey)
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}
	
	jsonData, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(client.ctx, "PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.fallbackClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

func deleteView(client *Client, projectKey, viewKey string) error {
	url := fmt.Sprintf("%s/api/v2/projects/%s/views/%s", client.apiHost, projectKey, viewKey)
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}
	
	req, err := http.NewRequestWithContext(client.ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.fallbackClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

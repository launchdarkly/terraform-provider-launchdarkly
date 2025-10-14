package launchdarkly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func viewRead(ctx context.Context, d *schema.ResourceData, meta interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return diag.FromErr(err)
	}

	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(KEY).(string)

	view, res, err := getView(betaClient, projectKey, viewKey)

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

	// Set maintainer fields in state based on API response
	// Since ExactlyOneOf validation ensures one maintainer field is always set,
	// we can safely set the appropriate field and clear the other
	if view.Maintainer != nil {
		if view.Maintainer.Kind == "member" && view.Maintainer.MaintainerMember != nil {
			_ = d.Set(MAINTAINER_ID, view.Maintainer.MaintainerMember.Id)
			_ = d.Set(MAINTAINER_TEAM_KEY, "")
		} else if view.Maintainer.Kind == "team" && view.Maintainer.MaintainerTeam != nil {
			_ = d.Set(MAINTAINER_TEAM_KEY, view.Maintainer.MaintainerTeam.Key)
			_ = d.Set(MAINTAINER_ID, "")
		}
	}

	err = d.Set(TAGS, view.Tags)
	if err != nil {
		return diag.Errorf("could not set tags on view with key %q: %v", view.Key, err)
	}

	// For data sources, also fetch and set linked flags for discovery
	if isDataSource {
		linkedFlags, err := getLinkedResources(betaClient, projectKey, viewKey, FLAGS)
		if err != nil {
			// Log warning but don't fail the read for discovery data
			log.Printf("[WARN] failed to get linked flags for view %q in project %q: %v", viewKey, projectKey, err)
		} else {
			flagKeys := make([]string, len(linkedFlags))
			for i, flag := range linkedFlags {
				flagKeys[i] = flag.ResourceKey
			}
			err = d.Set(LINKED_FLAGS, flagKeys)
			if err != nil {
				return diag.Errorf("could not set linked_flags on view with key %q: %v", view.Key, err)
			}
		}

		// Also fetch and set linked segments for discovery
		linkedSegments, err := getLinkedResources(betaClient, projectKey, viewKey, SEGMENTS)
		if err != nil {
			// Log warning but don't fail the read for discovery data
			log.Printf("[WARN] failed to get linked segments for view %q in project %q: %v", viewKey, projectKey, err)
		} else {
			segments := make([]map[string]interface{}, len(linkedSegments))
			for i, segment := range linkedSegments {
				segments[i] = map[string]interface{}{
					SEGMENT_ENVIRONMENT_ID: segment.EnvironmentId,
					SEGMENT_KEY:            segment.ResourceKey,
				}
			}
			err = d.Set(LINKED_SEGMENTS, segments)
			if err != nil {
				return diag.Errorf("could not set linked_segments on view with key %q: %v", view.Key, err)
			}
		}
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

func buildViewURL(client *Client, projectKey, viewKey string) string {
	host := client.apiHost
	if host == "" {
		host = "app.launchdarkly.com"
	}
	var url string
	if viewKey == "" {
		url = fmt.Sprintf("https://%s/api/v2/projects/%s/views", host, projectKey)
	} else {
		url = fmt.Sprintf("https://%s/api/v2/projects/%s/views/%s", host, projectKey, viewKey)
	}
	return url
}

func getViewRaw(client *Client, projectKey, viewKey string) (*View, *http.Response, error) {
	url := buildViewURL(client, projectKey, viewKey)

	req, err := http.NewRequestWithContext(client.ctx, "GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", "beta")

	resp, err := client.ld.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return nil, resp, err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp, readErr
	}

	if resp.StatusCode >= 400 {
		return nil, resp, fmt.Errorf("LaunchDarkly API error - Status: %d, Response: %s", resp.StatusCode, string(body))
	}

	var view View
	if err := json.Unmarshal(body, &view); err != nil {
		return nil, resp, err
	}

	return &view, resp, nil
}

func createView(client *Client, projectKey string, viewPost map[string]interface{}) (*View, error) {
	url := buildViewURL(client, projectKey, "")
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
	req.Header.Set("LD-API-Version", "beta")

	resp, err := client.ld.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("LaunchDarkly API error - Status: %d, Response: %s", resp.StatusCode, string(body))
	}

	var view View
	if err := json.Unmarshal(body, &view); err != nil {
		return nil, err
	}

	return &view, nil
}

func patchView(client *Client, projectKey, viewKey string, patch map[string]interface{}) error {
	url := buildViewURL(client, projectKey, viewKey)
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
	req.Header.Set("LD-API-Version", "beta")

	resp, err := client.ld.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return readErr
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("LaunchDarkly API error - Status: %d, Response: %s", resp.StatusCode, string(body))
	}

	return nil
}

func deleteView(client *Client, projectKey, viewKey string) error {
	url := buildViewURL(client, projectKey, viewKey)

	req, err := http.NewRequestWithContext(client.ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", "beta")

	resp, err := client.ld.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return readErr
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("LaunchDarkly API error - Status: %d, Response: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ViewLinkedResource represents a linked resource in a view
type ViewLinkedResource struct {
	ResourceKey   string `json:"resourceKey"`
	ResourceType  string `json:"resourceType"`
	LinkedAt      int64  `json:"linkedAt"`
	EnvironmentId string `json:"environmentId,omitempty"`
}

// ViewLinkedResources represents the response from getting linked resources
type ViewLinkedResources struct {
	Items []ViewLinkedResource `json:"items"`
}

// ViewLinkRequest represents the request body for linking resources
type ViewLinkRequest struct {
	Keys               []string                `json:"keys,omitempty"`
	SegmentIdentifiers []ViewSegmentIdentifier `json:"segmentIdentifiers,omitempty"`
	Comment            string                  `json:"comment,omitempty"`
}

// ViewSegmentIdentifier represents a segment identifier for linking to a view
type ViewSegmentIdentifier struct {
	EnvironmentId string `json:"environmentId"`
	SegmentKey    string `json:"segmentKey"`
}

// chunkStringSlice splits a string slice into chunks of the specified size
func chunkStringSlice(slice []string, chunkSize int) [][]string {
	var chunks [][]string
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

// chunkSegmentIdentifiers splits a segment identifier slice into chunks of the specified size
func chunkSegmentIdentifiers(slice []ViewSegmentIdentifier, chunkSize int) [][]ViewSegmentIdentifier {
	var chunks [][]ViewSegmentIdentifier
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

// linkResourcesToView links resources to a view
// The API supports a maximum of 20 keys per request, so we chunk the keys accordingly
func linkResourcesToView(client *Client, projectKey, viewKey, resourceType string, resourceKeys []string) error {
	// Flagged on the BE, can't read flag here
	const maxKeysPerRequest = 10

	// Handle empty slice
	if len(resourceKeys) == 0 {
		return nil
	}

	// Chunk the keys into groups of maxKeysPerRequest
	keyChunks := chunkStringSlice(resourceKeys, maxKeysPerRequest)

	var errors []string

	for i, chunk := range keyChunks {
		err := linkResourceChunkToView(client, projectKey, viewKey, resourceType, chunk)
		if err != nil {
			errors = append(errors, fmt.Sprintf("chunk %d/%d: %v", i+1, len(keyChunks), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to link some resource chunks: %s", strings.Join(errors, "; "))
	}

	return nil
}

// performViewLinkOperation performs the actual HTTP request for linking/unlinking resources
func performViewLinkOperation(client *Client, projectKey, viewKey, resourceType string, resourceKeys []string, method string) error {
	url := buildViewLinkURL(client, projectKey, viewKey, resourceType)

	linkRequest := ViewLinkRequest{
		Keys: resourceKeys,
	}

	jsonData, err := json.Marshal(linkRequest)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(client.ctx, method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", "beta")

	resp, err := client.ld.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// performViewSegmentLinkOperation performs the actual HTTP request for linking/unlinking segments
func performViewSegmentLinkOperation(client *Client, projectKey, viewKey string, segmentIdentifiers []ViewSegmentIdentifier, method string) error {
	url := buildViewLinkURL(client, projectKey, viewKey, SEGMENTS)

	linkRequest := ViewLinkRequest{
		SegmentIdentifiers: segmentIdentifiers,
	}

	jsonData, err := json.Marshal(linkRequest)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(client.ctx, method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", "beta")

	resp, err := client.ld.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// linkResourceChunkToView links a single chunk of resources to a view
func linkResourceChunkToView(client *Client, projectKey, viewKey, resourceType string, resourceKeys []string) error {
	return performViewLinkOperation(client, projectKey, viewKey, resourceType, resourceKeys, "POST")
}

// unlinkResourcesFromView unlinks resources from a view
func unlinkResourcesFromView(client *Client, projectKey, viewKey, resourceType string, resourceKeys []string) error {
	// Flagged on the BE, can't read flag here
	const maxKeysPerRequest = 10

	// Handle empty slice
	if len(resourceKeys) == 0 {
		return nil
	}

	// Chunk the keys into groups of maxKeysPerRequest
	keyChunks := chunkStringSlice(resourceKeys, maxKeysPerRequest)

	var errors []string

	for i, chunk := range keyChunks {
		err := unlinkResourceChunkFromView(client, projectKey, viewKey, resourceType, chunk)
		if err != nil {
			errors = append(errors, fmt.Sprintf("chunk %d/%d: %v", i+1, len(keyChunks), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to unlink some resource chunks: %s", strings.Join(errors, "; "))
	}

	return nil
}

// unlinkResourceChunkFromView unlinks a single chunk of resources from a view
func unlinkResourceChunkFromView(client *Client, projectKey, viewKey, resourceType string, resourceKeys []string) error {
	return performViewLinkOperation(client, projectKey, viewKey, resourceType, resourceKeys, "DELETE")
}

// linkSegmentsToView links segments to a view using segment identifiers
func linkSegmentsToView(client *Client, projectKey, viewKey string, segmentIdentifiers []ViewSegmentIdentifier) error {
	const maxSegmentsPerRequest = 10

	// Handle empty slice
	if len(segmentIdentifiers) == 0 {
		return nil
	}

	// Chunk the segment identifiers into groups of maxSegmentsPerRequest
	segmentChunks := chunkSegmentIdentifiers(segmentIdentifiers, maxSegmentsPerRequest)

	var errors []string

	for i, chunk := range segmentChunks {
		err := linkSegmentChunkToView(client, projectKey, viewKey, chunk)
		if err != nil {
			errors = append(errors, fmt.Sprintf("chunk %d/%d: %v", i+1, len(segmentChunks), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to link some segment chunks: %s", strings.Join(errors, "; "))
	}

	return nil
}

// linkSegmentChunkToView links a single chunk of segments to a view
func linkSegmentChunkToView(client *Client, projectKey, viewKey string, segmentIdentifiers []ViewSegmentIdentifier) error {
	return performViewSegmentLinkOperation(client, projectKey, viewKey, segmentIdentifiers, "POST")
}

// unlinkSegmentsFromView unlinks segments from a view
func unlinkSegmentsFromView(client *Client, projectKey, viewKey string, segmentIdentifiers []ViewSegmentIdentifier) error {
	const maxSegmentsPerRequest = 10

	// Handle empty slice
	if len(segmentIdentifiers) == 0 {
		return nil
	}

	// Chunk the segment identifiers into groups of maxSegmentsPerRequest
	segmentChunks := chunkSegmentIdentifiers(segmentIdentifiers, maxSegmentsPerRequest)

	var errors []string

	for i, chunk := range segmentChunks {
		err := unlinkSegmentChunkFromView(client, projectKey, viewKey, chunk)
		if err != nil {
			errors = append(errors, fmt.Sprintf("chunk %d/%d: %v", i+1, len(segmentChunks), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to unlink some segment chunks: %s", strings.Join(errors, "; "))
	}

	return nil
}

// unlinkSegmentChunkFromView unlinks a single chunk of segments from a view
func unlinkSegmentChunkFromView(client *Client, projectKey, viewKey string, segmentIdentifiers []ViewSegmentIdentifier) error {
	return performViewSegmentLinkOperation(client, projectKey, viewKey, segmentIdentifiers, "DELETE")
}

// getLinkedResources gets all linked resources of a specific type for a view
func getLinkedResources(client *Client, projectKey, viewKey, resourceType string) ([]ViewLinkedResource, error) {
	url := buildViewLinkedResourcesURL(client, projectKey, viewKey, resourceType)

	req, err := http.NewRequestWithContext(client.ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", "beta")

	resp, err := client.ld.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var linkedResources ViewLinkedResources
	err = json.NewDecoder(resp.Body).Decode(&linkedResources)
	if err != nil {
		return nil, err
	}

	return linkedResources.Items, nil
}

// buildViewLinkURL builds the URL for linking/unlinking resources to/from a view
func buildViewLinkURL(client *Client, projectKey, viewKey, resourceType string) string {
	host := client.apiHost
	if host == "" {
		host = "app.launchdarkly.com"
	}
	return fmt.Sprintf("https://%s/api/v2/projects/%s/views/%s/link/%s", host, projectKey, viewKey, resourceType)
}

// buildViewLinkedResourcesURL builds the URL for getting linked resources from a view
func buildViewLinkedResourcesURL(client *Client, projectKey, viewKey, resourceType string) string {
	host := client.apiHost
	if host == "" {
		host = "app.launchdarkly.com"
	}
	return fmt.Sprintf("https://%s/api/v2/projects/%s/views/%s/linked/%s", host, projectKey, viewKey, resourceType)
}

// viewIdToKeys splits a view ID into project key and view key
func viewIdToKeys(id string) (projectKey string, viewKey string, err error) {
	if strings.Count(id, "/") != 1 {
		return "", "", fmt.Errorf("found unexpected view ID format: %s. expected format: 'project_key/view_key'", id)
	}
	parts := strings.Split(id, "/")
	return parts[0], parts[1], nil
}

// ViewsResponse represents the response from getting all views
type ViewsResponse struct {
	Items []View `json:"items"`
}

// getViewsContainingFlag finds all views that contain a specific flag using the view-associations endpoint
func getViewsContainingFlag(client *Client, projectKey, flagKey string) ([]string, error) {
	url := buildViewAssociationsURL(client, projectKey, FLAGS, flagKey)
	req, err := http.NewRequestWithContext(client.ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", "beta")

	resp, err := client.ld.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var viewsResponse ViewsResponse
	err = json.NewDecoder(resp.Body).Decode(&viewsResponse)
	if err != nil {
		return nil, err
	}

	// Extract view keys from the response
	viewKeys := make([]string, len(viewsResponse.Items))
	for i, view := range viewsResponse.Items {
		viewKeys[i] = view.Key
	}

	return viewKeys, nil
}

// getViewsContainingSegment finds all views that contain a specific segment using the view-associations endpoint
func getViewsContainingSegment(client *Client, projectKey, environmentId, segmentKey string) ([]string, error) {
	url := buildViewAssociationsURLWithEnv(client, projectKey, SEGMENTS, segmentKey, environmentId)
	req, err := http.NewRequestWithContext(client.ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", "beta")

	resp, err := client.ld.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var viewsResponse ViewsResponse
	err = json.NewDecoder(resp.Body).Decode(&viewsResponse)
	if err != nil {
		return nil, err
	}

	// Extract view keys from the response
	viewKeys := make([]string, len(viewsResponse.Items))
	for i, view := range viewsResponse.Items {
		viewKeys[i] = view.Key
	}

	return viewKeys, nil
}

// buildViewAssociationsURL builds the URL for getting views associated with a specific resource
func buildViewAssociationsURL(client *Client, projectKey, resourceType, resourceKey string) string {
	host := client.apiHost
	if host == "" {
		host = "app.launchdarkly.com"
	}
	return fmt.Sprintf("https://%s/api/v2/projects/%s/view-associations/%s/%s", host, projectKey, resourceType, resourceKey)
}

// buildViewAssociationsURLWithEnv builds the URL for getting views associated with a specific resource (with environment ID for segments)
func buildViewAssociationsURLWithEnv(client *Client, projectKey, resourceType, resourceKey, environmentId string) string {
	host := client.apiHost
	if host == "" {
		host = "app.launchdarkly.com"
	}
	url := fmt.Sprintf("https://%s/api/v2/projects/%s/view-associations/%s/%s", host, projectKey, resourceType, resourceKey)
	if environmentId != "" {
		url = fmt.Sprintf("%s?environmentId=%s", url, environmentId)
	}
	return url
}

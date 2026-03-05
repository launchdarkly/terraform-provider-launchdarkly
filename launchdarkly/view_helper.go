package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// setViewRequestHeaders sets the common headers for View API requests
func setViewRequestHeaders(req *http.Request, apiKey string) {
	req.Header.Set("Authorization", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", "beta")
	req.Header.Set("User-Agent", fmt.Sprintf("launchdarkly-terraform-provider/%s", version))
}

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
	description := ""
	if view.Description != nil {
		description = *view.Description
	}
	_ = d.Set(DESCRIPTION, description)
	generateSDKKeys := false
	if view.GenerateSdkKeys != nil {
		generateSDKKeys = *view.GenerateSdkKeys
	}
	_ = d.Set(GENERATE_SDK_KEYS, generateSDKKeys)
	archived := false
	if view.Archived != nil {
		archived = *view.Archived
	}
	_ = d.Set(ARCHIVED, archived)

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

func viewFromAPI(apiView *ldapi.View) *View {
	if apiView == nil {
		return nil
	}

	description := apiView.Description
	generateSDKKeys := apiView.GenerateSdkKeys
	archived := apiView.Archived

	view := &View{
		Id:              apiView.Id,
		Key:             apiView.Key,
		Name:            apiView.Name,
		Description:     &description,
		ProjectKey:      apiView.ProjectKey,
		GenerateSdkKeys: &generateSDKKeys,
		Archived:        &archived,
		Tags:            apiView.Tags,
	}

	if apiView.Maintainer != nil {
		maintainer := &ViewMaintainer{
			Kind: apiView.Maintainer.Kind,
		}
		if apiView.Maintainer.MaintainerMember != nil {
			maintainer.MaintainerMember = &ViewMaintainerMember{
				Id: apiView.Maintainer.MaintainerMember.Id,
			}
		}
		if apiView.Maintainer.MaintainerTeam != nil {
			maintainer.MaintainerTeam = &ViewMaintainerTeam{
				Key: apiView.Maintainer.MaintainerTeam.Key,
			}
		}
		view.Maintainer = maintainer
	}

	return view
}

func getView(client *Client, projectKey, viewKey string) (*View, *http.Response, error) {
	return getViewRaw(client, projectKey, viewKey)
}

func getViewRaw(client *Client, projectKey, viewKey string) (*View, *http.Response, error) {
	var (
		apiView *ldapi.View
		resp    *http.Response
		err     error
	)

	err = client.withConcurrency(client.ctx, func() error {
		apiView, resp, err = client.ld.ViewsBetaApi.GetView(client.ctx, projectKey, viewKey).
			LDAPIVersion("beta").
			Execute()
		return err
	})
	if err != nil {
		return nil, resp, err
	}

	return viewFromAPI(apiView), resp, nil
}

func createView(client *Client, projectKey string, viewPost map[string]interface{}) (*View, error) {
	viewRequest := ldapi.NewViewPost(viewPost["key"].(string), viewPost["name"].(string))

	if description, ok := viewPost["description"].(string); ok {
		viewRequest.SetDescription(description)
	}
	if generateSDKKeys, ok := viewPost["generateSdkKeys"].(bool); ok {
		viewRequest.SetGenerateSdkKeys(generateSDKKeys)
	}
	if maintainerID, ok := viewPost["maintainerId"].(string); ok {
		viewRequest.SetMaintainerId(maintainerID)
	}
	if maintainerTeamKey, ok := viewPost["maintainerTeamKey"].(string); ok {
		viewRequest.SetMaintainerTeamKey(maintainerTeamKey)
	}
	if tagsRaw, ok := viewPost["tags"]; ok {
		switch tags := tagsRaw.(type) {
		case []string:
			viewRequest.SetTags(tags)
		case []interface{}:
			viewRequest.SetTags(interfaceSliceToStringSlice(tags))
		}
	}

	var (
		apiView *ldapi.View
		err     error
	)
	err = client.withConcurrency(client.ctx, func() error {
		apiView, _, err = client.ld.ViewsBetaApi.CreateView(client.ctx, projectKey).
			LDAPIVersion("beta").
			ViewPost(*viewRequest).
			Execute()
		return err
	})
	if err != nil {
		return nil, err
	}

	return viewFromAPI(apiView), nil
}

func patchView(client *Client, projectKey, viewKey string, patch map[string]interface{}) error {
	viewPatch := ldapi.NewViewPatch()

	if name, ok := patch["name"].(string); ok {
		viewPatch.SetName(name)
	}
	if description, ok := patch["description"].(string); ok {
		viewPatch.SetDescription(description)
	}
	if generateSDKKeys, ok := patch["generateSdkKeys"].(bool); ok {
		viewPatch.SetGenerateSdkKeys(generateSDKKeys)
	}
	if maintainerID, ok := patch["maintainerId"].(string); ok {
		viewPatch.SetMaintainerId(maintainerID)
	}
	if maintainerTeamKey, ok := patch["maintainerTeamKey"].(string); ok {
		viewPatch.SetMaintainerTeamKey(maintainerTeamKey)
	}
	if tagsRaw, ok := patch["tags"]; ok {
		switch tags := tagsRaw.(type) {
		case []string:
			viewPatch.SetTags(tags)
		case []interface{}:
			viewPatch.SetTags(interfaceSliceToStringSlice(tags))
		}
	}
	if archived, ok := patch["archived"].(bool); ok {
		viewPatch.SetArchived(archived)
	}

	return client.withConcurrency(client.ctx, func() error {
		_, _, err := client.ld.ViewsBetaApi.UpdateView(client.ctx, projectKey, viewKey).
			LDAPIVersion("beta").
			ViewPatch(*viewPatch).
			Execute()
		return err
	})
}

func deleteView(client *Client, projectKey, viewKey string) error {
	return client.withConcurrency(client.ctx, func() error {
		_, err := client.ld.ViewsBetaApi.DeleteView(client.ctx, projectKey, viewKey).
			LDAPIVersion("beta").
			Execute()
		return err
	})
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
	Filter             string                  `json:"filter,omitempty"`
	EnvironmentId      string                  `json:"environmentId,omitempty"`
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

// performViewLinkOperation performs the typed API request for linking/unlinking resources.
func performViewLinkOperation(client *Client, projectKey, viewKey, resourceType string, resourceKeys []string, method string) error {
	requestPayload := ldapi.ViewLinkRequestKeysAsViewLinkRequest(ldapi.NewViewLinkRequestKeys(resourceKeys))

	return client.withConcurrency(client.ctx, func() error {
		switch method {
		case http.MethodPost:
			_, _, err := client.ld.ViewsBetaApi.LinkResource(client.ctx, projectKey, viewKey, resourceType).
				LDAPIVersion("beta").
				ViewLinkRequest(requestPayload).
				Execute()
			return err
		case http.MethodDelete:
			_, _, err := client.ld.ViewsBetaApi.UnlinkResource(client.ctx, projectKey, viewKey, resourceType).
				LDAPIVersion("beta").
				ViewLinkRequest(requestPayload).
				Execute()
			return err
		default:
			return fmt.Errorf("unsupported view link method %q", method)
		}
	})
}

// performViewSegmentLinkOperation performs the typed API request for linking/unlinking segments.
func performViewSegmentLinkOperation(client *Client, projectKey, viewKey string, segmentIdentifiers []ViewSegmentIdentifier, method string) error {
	apiSegmentIdentifiers := make([]ldapi.ViewLinkRequestSegmentIdentifier, len(segmentIdentifiers))
	for i, segmentIdentifier := range segmentIdentifiers {
		apiSegmentIdentifiers[i] = *ldapi.NewViewLinkRequestSegmentIdentifier(segmentIdentifier.EnvironmentId, segmentIdentifier.SegmentKey)
	}
	requestPayload := ldapi.ViewLinkRequestSegmentIdentifiersAsViewLinkRequest(
		ldapi.NewViewLinkRequestSegmentIdentifiers(apiSegmentIdentifiers),
	)

	return client.withConcurrency(client.ctx, func() error {
		switch method {
		case http.MethodPost:
			_, _, err := client.ld.ViewsBetaApi.LinkResource(client.ctx, projectKey, viewKey, SEGMENTS).
				LDAPIVersion("beta").
				ViewLinkRequest(requestPayload).
				Execute()
			return err
		case http.MethodDelete:
			_, _, err := client.ld.ViewsBetaApi.UnlinkResource(client.ctx, projectKey, viewKey, SEGMENTS).
				LDAPIVersion("beta").
				ViewLinkRequest(requestPayload).
				Execute()
			return err
		default:
			return fmt.Errorf("unsupported view segment link method %q", method)
		}
	})
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
	var (
		apiLinkedResources *ldapi.ViewLinkedResources
		err                error
	)

	err = client.withConcurrency(client.ctx, func() error {
		apiLinkedResources, _, err = client.ld.ViewsBetaApi.GetLinkedResources(client.ctx, projectKey, viewKey, resourceType).
			LDAPIVersion("beta").
			Execute()
		return err
	})
	if err != nil {
		return nil, err
	}

	linkedResources := make([]ViewLinkedResource, len(apiLinkedResources.Items))
	for i, resource := range apiLinkedResources.Items {
		environmentID := ""
		if resource.EnvironmentId != nil {
			environmentID = *resource.EnvironmentId
		}
		linkedResources[i] = ViewLinkedResource{
			ResourceKey:   resource.ResourceKey,
			ResourceType:  resource.ResourceType,
			LinkedAt:      resource.LinkedAt,
			EnvironmentId: environmentID,
		}
	}

	return linkedResources, nil
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
	var (
		viewsResponse *ldapi.Views
		err           error
	)

	err = client.withConcurrency(client.ctx, func() error {
		viewsResponse, _, err = client.ld.ViewsBetaApi.GetLinkedViews(client.ctx, projectKey, FLAGS, flagKey).
			LDAPIVersion("beta").
			Execute()
		return err
	})
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
	var (
		viewsResponse *ldapi.Views
		err           error
	)

	err = client.withConcurrency(client.ctx, func() error {
		request := client.ld.ViewsBetaApi.GetLinkedViews(client.ctx, projectKey, SEGMENTS, segmentKey).
			LDAPIVersion("beta")
		if environmentId != "" {
			request = request.EnvironmentId(environmentId)
		}
		viewsResponse, _, err = request.Execute()
		return err
	})
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

// performViewFilterLinkOperation performs an HTTP request for linking/unlinking resources using a filter string.
// environmentId is optional — only required for segment filter operations.
func performViewFilterLinkOperation(client *Client, projectKey, viewKey, resourceType, filter, environmentId, method string) error {
	filterRequest := ldapi.NewViewLinkRequestFilter(filter)
	if environmentId != "" {
		filterRequest.SetEnvironmentId(environmentId)
	}
	requestPayload := ldapi.ViewLinkRequestFilterAsViewLinkRequest(filterRequest)

	return client.withConcurrency(client.ctx, func() error {
		switch method {
		case http.MethodPost:
			_, _, err := client.ld.ViewsBetaApi.LinkResource(client.ctx, projectKey, viewKey, resourceType).
				LDAPIVersion("beta").
				ViewLinkRequest(requestPayload).
				Execute()
			return err
		case http.MethodDelete:
			_, _, err := client.ld.ViewsBetaApi.UnlinkResource(client.ctx, projectKey, viewKey, resourceType).
				LDAPIVersion("beta").
				ViewLinkRequest(requestPayload).
				Execute()
			return err
		default:
			return fmt.Errorf("unsupported view filter link method %q", method)
		}
	})
}

// linkResourcesByFilterToView links resources matching a filter to a view.
// environmentId is optional — only required for segment filter operations.
func linkResourcesByFilterToView(client *Client, projectKey, viewKey, resourceType, filter, environmentId string) error {
	return performViewFilterLinkOperation(client, projectKey, viewKey, resourceType, filter, environmentId, "POST")
}

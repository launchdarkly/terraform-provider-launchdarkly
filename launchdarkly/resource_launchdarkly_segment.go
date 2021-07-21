package launchdarkly

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func resourceSegment() *schema.Resource {
	schemaMap := baseSegmentSchema()
	schemaMap[PROJECT_KEY] = &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validateKey(),
		Description:  "The segment's project key.",
	}
	schemaMap[ENV_KEY] = &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validateKey(),
		Description:  "The segment's environment key.",
	}
	schemaMap[KEY] = &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validateKey(),
		Description:  "The unique key that references the segment.",
	}
	schemaMap[NAME] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The human-friendly name for the segment.",
	}
	return &schema.Resource{
		Create: resourceSegmentCreate,
		Read:   resourceSegmentRead,
		Update: resourceSegmentUpdate,
		Delete: resourceSegmentDelete,
		Exists: resourceSegmentExists,

		Importer: &schema.ResourceImporter{
			State: resourceSegmentImport,
		},

		Schema: schemaMap,
	}
}

func resourceSegmentCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)

	key := d.Get(KEY).(string)
	description := d.Get(DESCRIPTION).(string)
	segmentName := d.Get(NAME).(string)
	tags := stringsFromResourceData(d, TAGS)

	segment := ldapi.UserSegmentBody{
		Name:        segmentName,
		Key:         key,
		Description: description,
		Tags:        tags,
	}

	_, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.UserSegmentsApi.PostUserSegment(client.ctx, projectKey, envKey, segment)
	})

	if err != nil {
		return fmt.Errorf("failed to create segment %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	// ld's api does not allow some fields to be passed in during segment creation so we do an update:
	// https://apidocs.launchdarkly.com/reference#create-segment
	err = resourceSegmentUpdate(d, metaRaw)
	if err != nil {
		return fmt.Errorf("failed to update segment with name %q key %q for projectKey %q: %s",
			segmentName, key, projectKey, handleLdapiErr(err))
	}

	d.SetId(projectKey + "/" + envKey + "/" + key)
	return resourceSegmentRead(d, metaRaw)
}

func resourceSegmentRead(d *schema.ResourceData, metaRaw interface{}) error {
	return segmentRead(d, metaRaw, false)
}

func resourceSegmentUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	key := d.Get(KEY).(string)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)
	description := d.Get(DESCRIPTION).(string)
	name := d.Get(NAME).(string)
	tags := stringsFromResourceData(d, TAGS)
	included := d.Get(INCLUDED).([]interface{})
	excluded := d.Get(EXCLUDED).([]interface{})
	rules, err := segmentRulesFromResourceData(d, RULES)
	if err != nil {
		return err
	}
	patch := []ldapi.PatchOperation{
		patchReplace("/name", name),
		patchReplace("/description", description),
		patchReplace("/tags", tags),
		patchReplace("/temporary", TEMPORARY),
		patchReplace("/included", included),
		patchReplace("/excluded", excluded),
		patchReplace("/rules", rules),
	}

	_, _, err = handleRateLimit(func() (interface{}, *http.Response, error) {
		return handleNoConflict(func() (interface{}, *http.Response, error) {
			return client.ld.UserSegmentsApi.PatchUserSegment(client.ctx, projectKey, envKey, key, patch)
		})
	})
	if err != nil {
		return fmt.Errorf("failed to update segment %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return resourceSegmentRead(d, metaRaw)
}

func resourceSegmentDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)
	key := d.Get(KEY).(string)

	_, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		res, err := client.ld.UserSegmentsApi.DeleteUserSegment(client.ctx, projectKey, envKey, key)
		return nil, res, err
	})

	if err != nil {
		return fmt.Errorf("failed to delete segment %q from project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return nil
}

func resourceSegmentExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)
	key := d.Get(KEY).(string)

	_, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.UserSegmentsApi.GetUserSegment(client.ctx, projectKey, envKey, key)
	})
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check if segment %q exists in project %q: %s",
			key, projectKey, handleLdapiErr(err))
	}
	return true, nil
}

func resourceSegmentImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()

	if strings.Count(id, "/") != 2 {
		return nil, fmt.Errorf("found unexpected segment id format: %q expected format: 'project_key/env_key/segment_key'", id)
	}

	parts := strings.SplitN(d.Id(), "/", 3)

	projectKey, envKey, segmentKey := parts[0], parts[1], parts[2]

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(ENV_KEY, envKey)
	_ = d.Set(KEY, segmentKey)

	return []*schema.ResourceData{d}, nil
}

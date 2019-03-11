package launchdarkly

import (
	"fmt"
	"strings"

	"github.com/launchdarkly/api-client-go"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceSegment() *schema.Resource {
	return &schema.Resource{
		Create: resourceSegmentCreate,
		Read:   resourceSegmentRead,
		Update: resourceSegmentUpdate,
		Delete: resourceSegmentDelete,
		Exists: resourceSegmentExists,

		Importer: &schema.ResourceImporter{
			State: resourceSegmentImport,
		},

		Schema: map[string]*schema.Schema{
			project_key: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			env_key: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			key: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			name: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			description: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			tags: tagsSchema(),
			included: &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			excluded: &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
		},
	}
}

func resourceSegmentCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(project_key).(string)
	envKey := d.Get(env_key).(string)

	if exists, err := projectExists(projectKey, client); !exists {
		if err != nil {
			return err
		}
		return fmt.Errorf("Cannot find project with key %q", projectKey)
	}

	if exists, err := environmentExists(projectKey, envKey, client); !exists {
		if err != nil {
			return err
		}
		return fmt.Errorf("Cannot find environment with key %q", envKey)
	}

	key := d.Get(key).(string)
	description := d.Get(description).(string)
	segmentName := d.Get(name).(string)
	tags := stringsFromResourceData(d, tags)

	segment := ldapi.UserSegmentBody{
		Name:        segmentName,
		Key:         key,
		Description: description,
		Tags:        tags,
	}

	_, _, err := client.LaunchDarkly.UserSegmentsApi.PostUserSegment(client.Ctx, projectKey, envKey, segment)

	if err != nil {
		return fmt.Errorf("Failed to create segment %q in project %q: %s", key, projectKey, err)
	}

	// LaunchDarkly's api does not allow some fields to be passed in during segment creation so we do an update:
	// https://apidocs.launchdarkly.com/reference#create-segment
	err = resourceSegmentUpdate(d, metaRaw)
	if err != nil {
		return fmt.Errorf("failed to update segment with name %q key %q for projectKey %q: %v", segmentName, key, projectKey, err)
	}

	d.SetId(projectKey + "/" + envKey + "/" + key)
	return resourceSegmentRead(d, metaRaw)
}

func resourceSegmentRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(project_key).(string)
	envKey := d.Get(env_key).(string)
	key := d.Get(key).(string)

	segment, _, err := client.LaunchDarkly.UserSegmentsApi.GetUserSegment(client.Ctx, projectKey, envKey, key)

	if err != nil {
		return fmt.Errorf("Failed to get segment %q of project %q: %s", key, projectKey, err)
	}

	d.Set(name, segment.Name)
	d.Set(description, segment.Description)
	d.Set(tags, segment.Tags)
	d.Set(included, segment.Included)
	d.Set(excluded, segment.Excluded)
	return nil
}

func resourceSegmentUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	key := d.Get(key).(string)
	projectKey := d.Get(project_key).(string)
	envKey := d.Get(env_key).(string)
	description := d.Get(description).(string)
	name := d.Get(name).(string)
	tags := stringsFromResourceData(d, tags)
	included := d.Get(included).([]interface{})
	excluded := d.Get(excluded).([]interface{})

	patch := []ldapi.PatchOperation{
		patchReplace("/name", name),
		patchReplace("/description", description),
		patchReplace("/tags", tags),
		patchReplace("/temporary", temporary),
		patchReplace("/included", included),
		patchReplace("/excluded", excluded),
	}

	_, _, err := client.LaunchDarkly.UserSegmentsApi.PatchUserSegment(client.Ctx, projectKey, envKey, key, patch)
	if err != nil {
		return fmt.Errorf("Failed to update segment %q in project %q: %s", key, projectKey, err)
	}

	return resourceSegmentRead(d, metaRaw)
}

func resourceSegmentDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(project_key).(string)
	envKey := d.Get(env_key).(string)
	key := d.Get(key).(string)

	_, err := client.LaunchDarkly.UserSegmentsApi.DeleteUserSegment(client.Ctx, projectKey, envKey, key)
	if err != nil {
		return fmt.Errorf("Failed to delete segment %q from project %q: %s", key, projectKey, err)
	}

	return nil
}

func resourceSegmentExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	projectKey := d.Get(project_key).(string)
	envKey := d.Get(env_key).(string)
	key := d.Get(key).(string)

	_, httpResponse, err := client.LaunchDarkly.UserSegmentsApi.GetUserSegment(client.Ctx, projectKey, envKey, key)
	if httpResponse != nil && httpResponse.StatusCode == 404 {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("Failed to check if segment %q exists in project %q: %s", key, projectKey, err)
	}
	return true, nil
}

func resourceSegmentImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()

	if strings.Count(id, "/") != 2 {
		return nil, fmt.Errorf("found unexpected segment id format: %q expected format: 'project_key/env_key/segment_key'", id)
	}

	parts := strings.SplitN(d.Id(), "/", 3)

	projectKey, envKey, key := parts[0], parts[1], parts[2]

	d.Set(project_key, projectKey)
	d.Set(env_key, envKey)
	d.Set(key, key)
	d.SetId(key)

	if err := resourceSegmentRead(d, meta); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

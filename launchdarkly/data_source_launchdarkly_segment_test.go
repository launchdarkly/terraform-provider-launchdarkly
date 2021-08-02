package launchdarkly

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/require"
)

const (
	testAccDataSourceSegment = `
data "launchdarkly_segment" "test" {
	key = "%s"
	project_key = "%s"
	env_key = "test"
}
`
)

type testSegmentUpdate struct {
	Included []interface{}
	Excluded []interface{}
	Rules    []ldapi.UserSegmentRule
}

func testAccDataSourceSegmentCreate(client *Client, projectKey, segmentKey string, properties testSegmentUpdate) (*ldapi.UserSegment, error) {
	envKey := "test"
	projectBody := ldapi.ProjectBody{
		Name: "Terraform Segment DS Test",
		Key:  projectKey,
	}
	project, err := testAccDataSourceProjectCreate(client, projectBody)
	if err != nil {
		return nil, err
	}

	segmentBody := ldapi.UserSegmentBody{
		Name:        "Data Source Test Segment",
		Key:         segmentKey,
		Description: "test description",
		Tags:        []string{"terraform"},
	}
	_, _, err = handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.UserSegmentsApi.PostUserSegment(client.ctx, project.Key, envKey, segmentBody)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create segment %q in project %q: %s", segmentKey, projectKey, handleLdapiErr(err))
	}

	patch := []ldapi.PatchOperation{
		patchReplace("/included", properties.Included),
		patchReplace("/excluded", properties.Excluded),
		patchReplace("/rules", properties.Rules),
	}
	rawSegment, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return handleNoConflict(func() (interface{}, *http.Response, error) {
			return client.ld.UserSegmentsApi.PatchUserSegment(client.ctx, projectKey, envKey, segmentKey, patch)
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update segment %q in project %q: %s", segmentKey, projectKey, handleLdapiErr(err))
	}

	if segment, ok := rawSegment.(ldapi.UserSegment); ok {
		return &segment, nil
	}
	return nil, fmt.Errorf("failed to create segment %q in project %q: %s", segmentKey, projectKey, handleLdapiErr(err))
}

func TestAccDataSourceSegment_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	segmentKey := "bad-segment-key"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false)
	require.NoError(t, err)
	_, err = testAccDataSourceProjectCreate(client, ldapi.ProjectBody{Name: "Segment DS No Match Test", Key: projectKey})
	require.NoError(t, err)

	defer func() {
		err := testAccDataSourceProjectDelete(client, projectKey)
		require.NoError(t, err)
	}()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceSegment, segmentKey, projectKey),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`errors during refresh: failed to get segment "bad-segment-key" of project "%s": 404 Not Found:`, projectKey)),
			},
		},
	})
}

func TestAccDataSourceSegment_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	segmentKey := "data-source-test"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false)
	require.NoError(t, err)

	properties := testSegmentUpdate{
		Included: []interface{}{"some@email.com", "some_other@email.com"},
		Excluded: []interface{}{"some_bad@email.com"},
		Rules: []ldapi.UserSegmentRule{
			{
				Clauses: []ldapi.Clause{
					{
						Attribute: "name",
						Op:        "startsWith",
						Values:    []interface{}{"a"},
					},
				},
			},
		},
	}
	segment, err := testAccDataSourceSegmentCreate(client, projectKey, segmentKey, properties)
	require.NoError(t, err)

	defer func() {
		err := testAccDataSourceProjectDelete(client, projectKey)
		require.NoError(t, err)
	}()

	resourceName := "data.launchdarkly_segment.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceSegment, segmentKey, projectKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "key"),
					resource.TestCheckResourceAttr(resourceName, "name", segment.Name),
					resource.TestCheckResourceAttr(resourceName, "key", segment.Key),
					resource.TestCheckResourceAttr(resourceName, "id", projectKey+"/test/"+segmentKey),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", "name"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", "startsWith"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "included.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included.0", "some@email.com"),
					resource.TestCheckResourceAttr(resourceName, "excluded.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "excluded.0", "some_bad@email.com"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_date"),
				),
			},
		},
	})
}

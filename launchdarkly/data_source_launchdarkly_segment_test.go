package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v12"
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
	Included         []interface{}
	Excluded         []interface{}
	IncludedContexts []ldapi.SegmentTarget
	ExcludedContexts []ldapi.SegmentTarget
	Rules            []ldapi.UserSegmentRule
}

func testAccDataSourceSegmentCreate(client *Client, projectKey, segmentKey string, properties testSegmentUpdate) (*ldapi.UserSegment, error) {
	envKey := "test"
	projectBody := ldapi.ProjectPost{
		Name: "Terraform Segment DS Test",
		Key:  projectKey,
	}
	project, err := testAccProjectScaffoldCreate(client, projectBody)
	if err != nil {
		return nil, err
	}

	segmentBody := ldapi.SegmentBody{
		Name:        "Data Source Test Segment",
		Key:         segmentKey,
		Description: ldapi.PtrString("test description"),
		Tags:        []string{"terraform"},
	}
	_, _, err = client.ld.SegmentsApi.PostSegment(client.ctx, project.Key, envKey).SegmentBody(segmentBody).Execute()

	if err != nil {
		return nil, fmt.Errorf("failed to create segment %q in project %q: %s", segmentKey, projectKey, handleLdapiErr(err))
	}

	patch := ldapi.PatchWithComment{
		Patch: []ldapi.PatchOperation{
			patchReplace("/included", properties.Included),
			patchReplace("/excluded", properties.Excluded),
			patchReplace("/includedContexts", properties.IncludedContexts),
			patchReplace("/excludedContexts", properties.ExcludedContexts),
			patchReplace("/rules", properties.Rules),
		},
	}
	segment, _, err := client.ld.SegmentsApi.PatchSegment(client.ctx, projectKey, envKey, segmentKey).PatchWithComment(patch).Execute()

	if err != nil {
		return nil, fmt.Errorf("failed to update segment %q in project %q: %s", segmentKey, projectKey, handleLdapiErr(err))
	}

	return segment, nil
}

func TestAccDataSourceSegment_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	segmentKey := "bad-segment-key"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)
	_, err = testAccProjectScaffoldCreate(client, ldapi.ProjectPost{Name: "Segment DS No Match Test", Key: projectKey})
	require.NoError(t, err)

	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
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
				ExpectError: regexp.MustCompile(fmt.Sprintf(`Error: failed to get segment "bad-segment-key" of project "%s": 404 Not Found:`, projectKey)),
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
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)

	weight := int32(30000)
	testContextKind := "test-kind"
	accountContextKind := "account-kind"
	properties := testSegmentUpdate{
		Included: []interface{}{"some@email.com", "some_other@email.com"},
		Excluded: []interface{}{"some_bad@email.com"},
		IncludedContexts: []ldapi.SegmentTarget{
			{
				Values:      []string{"account1", "account2"},
				ContextKind: &accountContextKind,
			},
		},
		ExcludedContexts: []ldapi.SegmentTarget{
			{
				Values:      []string{"test1", "test2"},
				ContextKind: &testContextKind,
			},
		},
		Rules: []ldapi.UserSegmentRule{
			{
				Clauses: []ldapi.Clause{
					{
						Attribute: "name",
						Op:        "startsWith",
						Values:    []interface{}{"a"},
					},
				},
				Weight:             &weight,
				RolloutContextKind: &testContextKind,
			},
		},
	}
	segment, err := testAccDataSourceSegmentCreate(client, projectKey, segmentKey, properties)
	require.NoError(t, err)

	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
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
					resource.TestCheckResourceAttrSet(resourceName, KEY),
					resource.TestCheckResourceAttr(resourceName, NAME, segment.Name),
					resource.TestCheckResourceAttr(resourceName, KEY, segment.Key),
					resource.TestCheckResourceAttr(resourceName, ID, projectKey+"/test/"+segmentKey),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", "name"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", "startsWith"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.weight", "30000"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.rollout_context_kind", "test-kind"),
					resource.TestCheckResourceAttr(resourceName, "included.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included.0", "some@email.com"),
					resource.TestCheckResourceAttr(resourceName, "excluded.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "excluded.0", "some_bad@email.com"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.values.0", "account1"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.values.1", "account2"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.context_kind", "account-kind"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.values.0", "test1"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.values.1", "test2"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.context_kind", "test-kind"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
				),
			},
		},
	})
}

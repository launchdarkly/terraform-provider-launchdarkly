package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const (
	testAccSegmentCreate = `
resource "launchdarkly_project" "testProject" {
	name = "testProject"
	key = "test-project"
}

resource "launchdarkly_segment" "segment3" {
    key = "segmentKey1"
	project_key = "${launchdarkly_project.testProject.key}"
	env_key = "test"
  	name = "segment name"
	description = "segment description"
	tags = ["segmentTag1", "segmentTag2"]
	included = ["user1", "user2"]
	excluded = ["user3", "user4"]
}`

	testAccSegmentUpdate = `
resource "launchdarkly_project" "testProject" {
	name = "testProject"
	key = "test-project"
}
resource "launchdarkly_segment" "segment3" {
    key = "segmentKey1"
	project_key = "${launchdarkly_project.testProject.key}"
	env_key = "test"
  	name = "segment name"
	description = "segment description"
	tags = ["segmentTag1", "segmentTag2"]
	included = ["user1", "user2", "user3", "user4"]
	excluded = []
	rules = [
        {
        clauses = [{
            attribute = "test_att",
            op = "in",
            values = ["test"],
            negate = false,
            },
            {
            attribute = "test_att_1",
            op = "endsWith",
            values = ["test2"],
            negate = true,
            }],
        weight = 50000,
        bucket_by = "bucket"
		}
	]
}`
)

func TestAccSegment_Create(t *testing.T) {
	resourceName := "launchdarkly_segment.segment3"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSegmentCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.testProject"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, "project_key", "test-project"),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "segment name"),
					resource.TestCheckResourceAttr(resourceName, "description", "segment description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("segmentTag1"), "segmentTag1"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("segmentTag2"), "segmentTag2"),
					resource.TestCheckResourceAttr(resourceName, "included.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "included.1", "user2"),
					resource.TestCheckResourceAttr(resourceName, "excluded.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "excluded.0", "user3"),
					resource.TestCheckResourceAttr(resourceName, "excluded.1", "user4"),
				),
			},
		},
	})
}

func TestAccSegment_Update(t *testing.T) {
	resourceName := "launchdarkly_segment.segment3"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSegmentCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.testProject"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, "project_key", "test-project"),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "segment name"),
					resource.TestCheckResourceAttr(resourceName, "description", "segment description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("segmentTag1"), "segmentTag1"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("segmentTag2"), "segmentTag2"),
					resource.TestCheckResourceAttr(resourceName, "included.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "included.1", "user2"),
					resource.TestCheckResourceAttr(resourceName, "excluded.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "excluded.0", "user3"),
					resource.TestCheckResourceAttr(resourceName, "excluded.1", "user4"),
				),
			},
			{
				Config: testAccSegmentUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.testProject"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, "project_key", "test-project"),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "segment name"),
					resource.TestCheckResourceAttr(resourceName, "description", "segment description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("segmentTag1"), "segmentTag1"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("segmentTag2"), "segmentTag2"),
					resource.TestCheckResourceAttr(resourceName, "included.#", "4"),
					resource.TestCheckResourceAttr(resourceName, "included.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "included.1", "user2"),
					resource.TestCheckResourceAttr(resourceName, "included.2", "user3"),
					resource.TestCheckResourceAttr(resourceName, "included.3", "user4"),
					resource.TestCheckResourceAttr(resourceName, "excluded.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.weight", "50000"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.bucket_by", "bucket"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", "test_att"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", "in"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "test"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.attribute", "test_att_1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.op", "endsWith"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.values.0", "test2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.negate", "true"),
				),
			},
		},
	})
}

func testAccCheckSegmentExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		segmentKey, ok := rs.Primary.Attributes[key]
		if !ok {
			return fmt.Errorf("environment key not found: %s", resourceName)
		}
		envKey, ok := rs.Primary.Attributes[env_key]
		if !ok {
			return fmt.Errorf("project key not found: %s", resourceName)
		}
		projKey, ok := rs.Primary.Attributes[project_key]
		if !ok {
			return fmt.Errorf("project key not found: %s", resourceName)
		}
		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.UserSegmentsApi.GetUserSegment(client.ctx, projKey, envKey, segmentKey)
		if err != nil {
			return fmt.Errorf("received an error getting environment. %s", err)
		}
		return nil
	}
}

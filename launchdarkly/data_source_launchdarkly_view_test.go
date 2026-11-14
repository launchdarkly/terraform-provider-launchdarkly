package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	testAccDataSourceViewBasic = `
data "launchdarkly_view" "test" {
	project_key = "%s"
	key         = "%s"
}
`
)

func TestAccDataSourceView_noMatchReturnsError(t *testing.T) {
	projectKey := "nonexistent-project-key"
	viewKey := "nonexistent-view-key"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceViewBasic, projectKey, viewKey),
				ExpectError: regexp.MustCompile(`failed to get view with key "nonexistent-view-key" in project "nonexistent-project-key"`),
			},
		},
	})
}

func TestAccDataSourceView_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "view-data-source-test-" + projectKey
	viewKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	viewName := "Terraform Test View"
	viewDescription := "Test view description"
	tag := "test-tag"

	maintainerId := firstMemberIDForTest(t)

	resourceName := "data.launchdarkly_view.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
	environments = {
		"test-env" = {
			name  = "Test Environment"
			color = "000000"
		}
	}
}

resource "launchdarkly_view" "test" {
	project_key   = launchdarkly_project.test.key
	key           = "%s"
	name          = "%s"
	description   = "%s"
	maintainer_id = "%s"
	tags          = ["%s"]
}

data "launchdarkly_view" "test" {
	project_key = launchdarkly_project.test.key
	key         = launchdarkly_view.test.key
}
`, projectName, projectKey, viewKey, viewName, viewDescription, maintainerId, tag),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, PROJECT_KEY),
					resource.TestCheckResourceAttrSet(resourceName, KEY),
					resource.TestCheckResourceAttrSet(resourceName, NAME),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KEY, viewKey),
					resource.TestCheckResourceAttr(resourceName, NAME, viewName),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, viewDescription),
					resource.TestCheckResourceAttrSet(resourceName, ID),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
				),
			},
		},
	})
}

func TestAccDataSourceView_withLinkedFlags(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "view-discovery-test-" + projectKey
	resourceName := "data.launchdarkly_view.test"

	maintainerId := firstMemberIDForTest(t)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
	environments = {
		"test-env" = {
			name  = "Test Environment"
			color = "000000"
		}
	}
}

resource "launchdarkly_view" "test" {
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	description   = "Test view for discovery testing"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "test1" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-1"
	name        = "Test Flag 1"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
}

resource "launchdarkly_feature_flag" "test2" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-2"
	name        = "Test Flag 2"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
}

resource "launchdarkly_view_links" "test" {
	project_key = launchdarkly_project.test.key
	view_key    = launchdarkly_view.test.key
	
	flags = [
		launchdarkly_feature_flag.test1.key,
		launchdarkly_feature_flag.test2.key
	]
}

data "launchdarkly_view" "test" {
	project_key = launchdarkly_project.test.key
	key         = launchdarkly_view.test.key
	depends_on  = [launchdarkly_view_links.test]
}
`, projectName, projectKey, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "key", "test-view"),
					resource.TestCheckResourceAttr(resourceName, "name", "Test View"),
					resource.TestCheckResourceAttr(resourceName, "linked_flags.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "linked_flags.*", "test-flag-1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "linked_flags.*", "test-flag-2"),
				),
			},
		},
	})
}

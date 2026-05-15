package launchdarkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testAccContextKindResProject scaffolds the project a context kind needs.
func testAccContextKindResProject(projectKey string) string {
	return fmt.Sprintf(`
resource "launchdarkly_project" "test" {
  key  = "%s"
  name = "Context kind acceptance test"
  tags = ["terraform", "context-kind-test"]
  environments = [{
    name  = "Test Environment"
    key   = "test-env"
    color = "010101"
  }]
}
`, projectKey)
}

func testAccContextKindResBasic(projectKey, kindKey string) string {
	return fmt.Sprintf(`
%s

resource "launchdarkly_context_kind" "basic" {
  project_key = launchdarkly_project.test.key
  key         = "%s"
  name        = "Organization"
  description = "An organization that owns one or more accounts"
}
`, testAccContextKindResProject(projectKey), kindKey)
}

func testAccContextKindResUpdate(projectKey, kindKey string) string {
	return fmt.Sprintf(`
%s

resource "launchdarkly_context_kind" "basic" {
  project_key = launchdarkly_project.test.key
  key         = "%s"
  name        = "Organization v2"
  description = "Updated description"
}
`, testAccContextKindResProject(projectKey), kindKey)
}

func testAccContextKindResArchived(projectKey, kindKey string) string {
	return fmt.Sprintf(`
%s

resource "launchdarkly_context_kind" "basic" {
  project_key = launchdarkly_project.test.key
  key         = "%s"
  name        = "Organization"
  description = "An organization that owns one or more accounts"
  archived    = true
}
`, testAccContextKindResProject(projectKey), kindKey)
}

func testAccContextKindResRejectUser(projectKey string) string {
	return fmt.Sprintf(`
%s

resource "launchdarkly_context_kind" "blocked" {
  project_key = launchdarkly_project.test.key
  key         = "user"
  name        = "User"
}
`, testAccContextKindResProject(projectKey))
}

func testAccContextKindResWithDataSource(projectKey, kindKey string) string {
	return fmt.Sprintf(`
%s

resource "launchdarkly_context_kind" "basic" {
  project_key = launchdarkly_project.test.key
  key         = "%s"
  name        = "Organization"
  description = "Org via data source test"
}

data "launchdarkly_context_kind" "lookup" {
  project_key = launchdarkly_project.test.key
  key         = launchdarkly_context_kind.basic.key
}
`, testAccContextKindResProject(projectKey), kindKey)
}

func TestAccContextKind_basic(t *testing.T) {
	t.Parallel()
	resourceName := "launchdarkly_context_kind.basic"
	projectKey := "ck-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	kindKey := "organization-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccContextKindResBasic(projectKey, kindKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "key", kindKey),
					resource.TestCheckResourceAttr(resourceName, "name", "Organization"),
					resource.TestCheckResourceAttr(resourceName, "description", "An organization that owns one or more accounts"),
					resource.TestCheckResourceAttr(resourceName, "archived", "false"),
					resource.TestCheckResourceAttr(resourceName, "id", fmt.Sprintf("%s/%s", projectKey, kindKey)),
					resource.TestCheckResourceAttrSet(resourceName, "version"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_modified"),
					resource.TestCheckResourceAttrSet(resourceName, "created_from"),
				),
			},
		},
	})
}

func TestAccContextKind_update(t *testing.T) {
	t.Parallel()
	resourceName := "launchdarkly_context_kind.basic"
	projectKey := "ck-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	kindKey := "organization-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccContextKindResBasic(projectKey, kindKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Organization"),
				),
			},
			{
				Config: testAccContextKindResUpdate(projectKey, kindKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Organization v2"),
					resource.TestCheckResourceAttr(resourceName, "description", "Updated description"),
				),
			},
		},
	})
}

func TestAccContextKind_archive(t *testing.T) {
	t.Parallel()
	resourceName := "launchdarkly_context_kind.basic"
	projectKey := "ck-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	kindKey := "organization-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccContextKindResBasic(projectKey, kindKey),
				Check:  resource.TestCheckResourceAttr(resourceName, "archived", "false"),
			},
			{
				Config: testAccContextKindResArchived(projectKey, kindKey),
				Check:  resource.TestCheckResourceAttr(resourceName, "archived", "true"),
			},
			{
				Config: testAccContextKindResBasic(projectKey, kindKey),
				Check:  resource.TestCheckResourceAttr(resourceName, "archived", "false"),
			},
		},
	})
}

func TestAccContextKind_import(t *testing.T) {
	t.Parallel()
	resourceName := "launchdarkly_context_kind.basic"
	projectKey := "ck-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	kindKey := "organization-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccContextKindResBasic(projectKey, kindKey),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     fmt.Sprintf("%s/%s", projectKey, kindKey),
			},
		},
	})
}

func TestAccContextKind_rejectsUserKey(t *testing.T) {
	t.Parallel()
	projectKey := "ck-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config:      testAccContextKindResRejectUser(projectKey),
				ExpectError: regexp.MustCompile(`Cannot manage the built-in .user. context kind`),
			},
		},
	})
}

func TestAccContextKindDataSource_basic(t *testing.T) {
	t.Parallel()
	dataSourceName := "data.launchdarkly_context_kind.lookup"
	projectKey := "ck-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	kindKey := "organization-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccContextKindResWithDataSource(projectKey, kindKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(dataSourceName, "key", kindKey),
					resource.TestCheckResourceAttr(dataSourceName, "name", "Organization"),
					resource.TestCheckResourceAttr(dataSourceName, "description", "Org via data source test"),
					resource.TestCheckResourceAttr(dataSourceName, "id", fmt.Sprintf("%s/%s", projectKey, kindKey)),
				),
			},
		},
	})
}

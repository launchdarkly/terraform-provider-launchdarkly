package tests

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testAccContextKindProject scaffolds the project a context kind needs.
func testAccContextKindProject(projectKey string) string {
	return fmt.Sprintf(`
resource "launchdarkly_project" "test" {
  key  = "%s"
  name = "Context kind acceptance test"
  tags = ["terraform", "context-kind-test"]
  environments {
    name  = "Test Environment"
    key   = "test-env"
    color = "010101"
  }
}
`, projectKey)
}

func testAccContextKindBasic(projectKey, kindKey string) string {
	return fmt.Sprintf(`
%s

resource "launchdarkly_context_kind" "basic" {
  project_key = launchdarkly_project.test.key
  key         = "%s"
  name        = "Organization"
  description = "An organization that owns one or more accounts"
}
`, testAccContextKindProject(projectKey), kindKey)
}

func testAccContextKindUpdate(projectKey, kindKey string) string {
	return fmt.Sprintf(`
%s

resource "launchdarkly_context_kind" "basic" {
  project_key = launchdarkly_project.test.key
  key         = "%s"
  name        = "Organization v2"
  description = "Updated description"
}
`, testAccContextKindProject(projectKey), kindKey)
}

func testAccContextKindArchived(projectKey, kindKey string) string {
	return fmt.Sprintf(`
%s

resource "launchdarkly_context_kind" "basic" {
  project_key = launchdarkly_project.test.key
  key         = "%s"
  name        = "Organization"
  description = "An organization that owns one or more accounts"
  archived    = true
}
`, testAccContextKindProject(projectKey), kindKey)
}

func testAccContextKindRejectUser(projectKey string) string {
	return fmt.Sprintf(`
%s

resource "launchdarkly_context_kind" "blocked" {
  project_key = launchdarkly_project.test.key
  key         = "user"
  name        = "User"
}
`, testAccContextKindProject(projectKey))
}

func testAccContextKindWithDataSource(projectKey, kindKey string) string {
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
`, testAccContextKindProject(projectKey), kindKey)
}

func TestAccContextKind_basic(t *testing.T) {
	t.Parallel()
	resourceName := "launchdarkly_context_kind.basic"
	projectKey := "ck-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	kindKey := "organization-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccFrameworkMuxProviders(context.Background(), t),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccContextKindBasic(projectKey, kindKey),
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
		ProtoV5ProviderFactories: testAccFrameworkMuxProviders(context.Background(), t),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccContextKindBasic(projectKey, kindKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Organization"),
				),
			},
			{
				Config: testAccContextKindUpdate(projectKey, kindKey),
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
		ProtoV5ProviderFactories: testAccFrameworkMuxProviders(context.Background(), t),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccContextKindBasic(projectKey, kindKey),
				Check:  resource.TestCheckResourceAttr(resourceName, "archived", "false"),
			},
			{
				Config: testAccContextKindArchived(projectKey, kindKey),
				Check:  resource.TestCheckResourceAttr(resourceName, "archived", "true"),
			},
			{
				Config: testAccContextKindBasic(projectKey, kindKey),
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
		ProtoV5ProviderFactories: testAccFrameworkMuxProviders(context.Background(), t),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccContextKindBasic(projectKey, kindKey),
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
		ProtoV5ProviderFactories: testAccFrameworkMuxProviders(context.Background(), t),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config:      testAccContextKindRejectUser(projectKey),
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
		ProtoV5ProviderFactories: testAccFrameworkMuxProviders(context.Background(), t),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccContextKindWithDataSource(projectKey, kindKey),
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

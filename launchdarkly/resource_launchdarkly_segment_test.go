package launchdarkly

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	testAccSegmentCreate = `
resource "launchdarkly_segment" "test" {
    key         = "segmentKey1"
	project_key = launchdarkly_project.test.key
	env_key     = "test"
  	name        = "segment name"
	description = "segment description"
	tags        = ["segmentTag1", "segmentTag2"]
	included    = ["user1", "user2"]
	excluded    = ["user3", "user4"]
}`

	testAccSegmentUpdate = `
resource "launchdarkly_segment" "test" {
    key         = "segmentKey1"
	project_key = launchdarkly_project.test.key
	env_key     = "test"
  	name        = "segment name"
	description = "segment description"
	tags        = ["segmentTag1", ".segmentTag2"]
	included    = ["user1", "user2", "user3", "user4"]
	excluded    = []
	rules = [{
		clauses = [{
			attribute = "test_att"
			op = "in"
			values = ["test"]
		}, {
			attribute = "test_att_1"
			op = "endsWith"
			values = ["test2"]
			negate = true
			context_kind = "user"
		}]
		weight = 50000
		bucket_by = "bucket"
		rollout_context_kind = "other"
	}]
}`

	testAccSegmentUpdateWithContextTargets = `
	resource "launchdarkly_segment" "test" {
		key         = "segmentKey1"
		project_key = launchdarkly_project.test.key
		env_key     = "test"
		name        = "segment name"
		description = "segment description"
		included = ["user1", "user2"]
		included_contexts = [{
			values = ["account1", "account2"]
			context_kind = "account"
		}, {
			values = ["other_value"]
			context_kind = "other"
		}]
		excluded_contexts = [{
			values = ["bad_account"]
			context_kind = "account"
		}]
		rules = [{
			clauses = [{
				attribute = "test_att"
				op = "in"
				values = ["test"]
			}, {
				attribute = "test_att_1"
				op = "endsWith"
				values = ["test2"]
				negate = true
				context_kind = "user"
			}]
			weight = 50000
			bucket_by = "bucket"
		}]
	}`

	testAccSegmentCreateWithRules = `
resource "launchdarkly_segment" "test" {
    key         = "segmentKey1"
	project_key = launchdarkly_project.test.key
	env_key     = "test"
  	name        = "segment name"
	description = "segment description"
	tags        = ["segmentTag1", "segmentTag2"]
	included    = ["user1", "user2"]
	excluded    = ["user3", "user4"]
	unbounded = false
	rules = [{
		clauses = [{
			attribute = "test_att"
			op        = "endsWith"
			values    = ["test"]
			negate    = false
		}]
	}, {
		clauses = [{
			attribute  = "is_vip"
			op         = "in"
			values     = [true]
			value_type = "boolean"
			negate     = false
			context_kind = "account"
		}, {
			attribute  = "answer"
			op         = "in"
			values     = [42, 84.68]
			value_type = "number"
			negate     = true
			context_kind = "survey"
		}]
	}]
}`

	testAccSegmentCreateWithContextTargets = `
resource "launchdarkly_segment" "test" {
	key         = "segmentKey1"
	project_key = launchdarkly_project.test.key
	env_key     = "test"
	name        = "segment name"
	excluded = ["user1", "user2"]
	included_contexts = [{
		values = ["account1"]
		context_kind = "account"
	}]
	excluded_contexts = [{
		values = ["bad_account"]
		context_kind = "account"
	}, {
		values = ["meanie", "beanie"]
		context_kind = "eanies"
	}]
}`

	testAccSegmentCreateWithUnbounded = `
resource "launchdarkly_segment" "test" {
	key                    = "segmentKey1"
	project_key            = launchdarkly_project.test.key
	env_key                = "test"
	name                   = "segment name"
	description            = "segment description"
	tags                   = ["segmentTag1", "segmentTag2"]
	unbounded              = true
	unbounded_context_kind = "device"
}`

	testAccSegmentCreateWithUnboundedUpdate = `
resource "launchdarkly_segment" "test" {
	key                    = "segmentKey1"
	project_key            = launchdarkly_project.test.key
	env_key                = "test"
	name                   = "segment name"
	description            = "segment description"
	tags                   = ["segmentTag1", "segmentTag2"]
	unbounded              = true
	unbounded_context_kind = "account"
}`

	testAccSegmentWithAnonymousUser = `
resource "launchdarkly_segment" "anon" {
	key 					= "anonymousSegment"
	project_key            	= launchdarkly_project.test.key
	env_key                	= "test"
	name 					= "anonymous segment"
	rules = [{
		clauses = [{
			attribute  = "anonymous"
			op         = "in"
			negate     = false
			values = [
				true
			]
		}]
	}]
}
`
)

func TestAccSegment_CreateAndUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_segment.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccSegmentCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "segment description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "segmentTag1"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "segmentTag2"),
					resource.TestCheckResourceAttr(resourceName, "included.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "included.1", "user2"),
					resource.TestCheckResourceAttr(resourceName, "excluded.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "excluded.0", "user3"),
					resource.TestCheckResourceAttr(resourceName, "excluded.1", "user4"),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccSegmentUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "segment description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", ".segmentTag2"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "segmentTag1"),
					resource.TestCheckResourceAttr(resourceName, "included.#", "4"),
					resource.TestCheckResourceAttr(resourceName, "included.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "included.1", "user2"),
					resource.TestCheckResourceAttr(resourceName, "included.2", "user3"),
					resource.TestCheckResourceAttr(resourceName, "included.3", "user4"),
					resource.TestCheckResourceAttr(resourceName, "excluded.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.weight", "50000"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.bucket_by", "bucket"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.rollout_context_kind", "other"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", "test_att"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", "in"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "test"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.context_kind", "user"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.attribute", "test_att_1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.op", "endsWith"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.values.0", "test2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.negate", "true"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.context_kind", "user"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccSegmentUpdateWithContextTargets),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "segment description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "included.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "included.1", "user2"),
					resource.TestCheckNoResourceAttr(resourceName, "excluded.#"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.values.0", "account1"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.values.1", "account2"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.1.values.0", "other_value"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.1.context_kind", "other"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.values.0", "bad_account"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.weight", "50000"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.bucket_by", "bucket"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.rollout_context_kind", "user"), // should default when missing
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", "test_att"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", "in"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "test"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.context_kind", "user"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.attribute", "test_att_1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.op", "endsWith"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.values.0", "test2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.negate", "true"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.context_kind", "user"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSegment_Unbounded(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_segment.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccSegmentCreateWithUnbounded),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "segment description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "segmentTag1"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "segmentTag2"),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
					resource.TestCheckResourceAttr(resourceName, UNBOUNDED, "true"),
					resource.TestCheckResourceAttr(resourceName, UNBOUNDED_CONTEXT_KIND, "device"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccSegmentCreateWithUnboundedUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "segment description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "segmentTag1"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "segmentTag2"),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
					resource.TestCheckResourceAttr(resourceName, UNBOUNDED, "true"),
					resource.TestCheckResourceAttr(resourceName, UNBOUNDED_CONTEXT_KIND, "account"),
				),
			},
		},
	})
}

func TestAccSegment_WithAnonymousClause(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_segment.anon"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccSegmentWithAnonymousUser),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "anonymousSegment"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "anonymous segment"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", "anonymous"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", "in"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSegment_WithRules(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_segment.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccSegmentCreateWithRules),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "segment description"),
					resource.TestCheckResourceAttr(resourceName, UNBOUNDED, "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", "test_att"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", "endsWith"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "test"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.value_type", "string"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.context_kind", "user"), // this should automatically populate even if not set
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.attribute", "is_vip"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.op", "in"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.values.0", "true"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.value_type", "boolean"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.1.attribute", "answer"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.1.op", "in"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.1.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.1.values.0", "42"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.1.values.1", "84.68"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.1.value_type", "number"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.1.negate", "true"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.1.context_kind", "survey"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Remove the rules block and confirm the rules were actually deleted
			{
				Config: withRandomProject(projectKey, testAccSegmentCreate), // this is the same but without the rules
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "segment description"),
					resource.TestCheckResourceAttr(resourceName, UNBOUNDED, "false"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "segmentTag1"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "segmentTag2"),
					resource.TestCheckResourceAttr(resourceName, "included.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "included.1", "user2"),
					resource.TestCheckResourceAttr(resourceName, "excluded.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "excluded.0", "user3"),
					resource.TestCheckResourceAttr(resourceName, "excluded.1", "user4"),
					resource.TestCheckNoResourceAttr(resourceName, "included_contexts.#"),
					resource.TestCheckNoResourceAttr(resourceName, "excluded_contexts.#"),
					resource.TestCheckNoResourceAttr(resourceName, fmt.Sprintf("%s.#", RULES)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSegment_WithTargetingByContext(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_segment.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccSegmentCreateWithContextTargets),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, "excluded.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "excluded.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "excluded.1", "user2"),
					resource.TestCheckNoResourceAttr(resourceName, "included.#"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.values.0", "account1"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.values.0", "bad_account"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.1.values.0", "meanie"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.1.values.1", "beanie"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.1.context_kind", "eanies"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccSegmentUpdateWithContextTargets),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "segment description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "included.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "included.1", "user2"),
					resource.TestCheckNoResourceAttr(resourceName, "excluded.#"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.values.0", "account1"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.values.1", "account2"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.1.values.0", "other_value"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.1.context_kind", "other"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.values.0", "bad_account"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.weight", "50000"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.bucket_by", "bucket"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.rollout_context_kind", "user"), // should default from the API if not set
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", "test_att"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", "in"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "test"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.context_kind", "user"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.attribute", "test_att_1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.op", "endsWith"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.values.0", "test2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.negate", "true"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.context_kind", "user"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{ // check that it reverts well as well
				Config: withRandomProject(projectKey, testAccSegmentCreateWithContextTargets),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, "excluded.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "excluded.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "excluded.1", "user2"),
					resource.TestCheckNoResourceAttr(resourceName, "included.#"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.values.0", "account1"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.values.0", "bad_account"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.1.values.0", "meanie"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.1.values.1", "beanie"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.1.context_kind", "eanies"),
					resource.TestCheckNoResourceAttr(resourceName, "rules.#"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
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
		segmentKey, ok := rs.Primary.Attributes[KEY]
		if !ok {
			return fmt.Errorf("environment key not found: %s", resourceName)
		}
		envKey, ok := rs.Primary.Attributes[ENV_KEY]
		if !ok {
			return fmt.Errorf("project key not found: %s", resourceName)
		}
		projKey, ok := rs.Primary.Attributes[PROJECT_KEY]
		if !ok {
			return fmt.Errorf("project key not found: %s", resourceName)
		}
		client := mustTestAccClient()
		_, _, err := client.ld.SegmentsApi.GetSegment(client.ctx, projKey, envKey, segmentKey).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting environment. %s", err)
		}
		return nil
	}
}

// TestAccSegment_ViewAssociationRequired tests that creating a segment without view_keys
// fails when the project requires view association for new segments
func TestAccSegment_ViewAssociationRequired(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_segment.test"
	testAccSegmentWithViewKeys := ""

	// Config with project requiring view association but segment without view_keys (should fail)
	testAccSegmentWithoutViewKeys := fmt.Sprintf(`
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "View Requirement Test"
	require_view_association_for_new_segments = true
	environments = [{
		key   = "test-env"
		name  = "Test Environment"
		color = "010101"
	}]
}

resource "launchdarkly_segment" "test" {
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	key         = "test-segment-no-views"
	name        = "Test Segment Without Views"
}
`, projectKey)

	// Config with project requiring view association and segment with view_keys (should succeed)
	testAccSegmentWithViewKeysTemplate := `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "View Requirement Test"
	require_view_association_for_new_segments = true
	environments = [{
		key   = "test-env"
		name  = "Test Environment"
		color = "010101"
	}]
}

resource "launchdarkly_view" "test" {
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	maintainer_id = "%s"
}

resource "launchdarkly_segment" "test" {
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	key         = "test-segment-with-views"
	name        = "Test Segment With Views"
	view_keys   = [launchdarkly_view.test.key]
}
`
	maintainerID := "507f1f77bcf86cd799439011"
	if os.Getenv("TF_ACC") != "" {
		testAccPreCheck(t)
		maintainerID = firstMemberIDForTest(t)
	}
	testAccSegmentWithViewKeys = fmt.Sprintf(testAccSegmentWithViewKeysTemplate, projectKey, maintainerID)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			// Step 1: Verify segment without view_keys fails when project requires it
			{
				Config:      testAccSegmentWithoutViewKeys,
				ExpectError: regexp.MustCompile(`requires new segments to be associated with at least one view`),
			},
			// Step 2: Verify segment with view_keys succeeds when project requires it
			{
				Config: testAccSegmentWithViewKeys,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "view_keys.#", "1"),
				),
			},
		},
	})
}

// TestAccSegment_ApprovalRequired covers issue #370: when segment approvals are
// enabled for an environment, a segment with no targeting must still create
// (the gated, no-op targeting PATCH is skipped), while a segment that does
// configure targeting must fail with the actionable "approval is required"
// error rather than the old opaque failure. Segment approvals are not exposed
// through the provider schema, so they are toggled out-of-band via the beta
// approval-settings API in a PreConfig step. Runs serially because it mutates
// environment-scoped approval settings.
func TestAccSegment_ApprovalRequired(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	envKey := "test-env"
	bareResourceName := "launchdarkly_segment.bare"

	projectOnly := fmt.Sprintf(`
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Segment Approvals Test"
	environments = [{
		key   = "%s"
		name  = "Test Environment"
		color = "010101"
	}]
}
`, projectKey, envKey)

	bareSegment := projectOnly + `
resource "launchdarkly_segment" "bare" {
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	key         = "approval-bare"
	name        = "bare under approvals"
	description = "no targeting; must create under approvals"
}
`

	withRules := bareSegment + `
resource "launchdarkly_segment" "rules" {
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	key         = "approval-rules"
	name        = "rules under approvals"
	rules = [{
		clauses = [{
			attribute    = "email"
			op           = "in"
			values       = ["a@b.com"]
			context_kind = "user"
		}]
	}]
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			// Step 1: create the project + environment (no segment yet).
			{
				Config: projectOnly,
			},
			// Step 2: enable segment approvals, then a targeting-free segment
			// must still create successfully.
			{
				PreConfig: func() { enableSegmentApprovalsForTest(t, projectKey, envKey) },
				Config:    bareSegment,
				Check:     resource.ComposeTestCheckFunc(testAccCheckSegmentExists(bareResourceName)),
			},
			// Step 3: a segment that configures targeting must fail with the
			// actionable approval error; the partial shell is rolled back.
			{
				// Terraform word-wraps diagnostics, so tolerate whitespace
				// (including newlines) between words.
				Config:      withRules,
				ExpectError: regexp.MustCompile(`approval\s+is\s+required`),
			},
		},
	})
}

// enableSegmentApprovalsForTest turns on segment approvals for the given
// environment via the beta approval-settings API. Segment approvals are not
// configurable through the provider schema, so the test toggles them directly.
func enableSegmentApprovalsForTest(t *testing.T, projectKey, envKey string) {
	t.Helper()
	host := os.Getenv(LAUNCHDARKLY_API_HOST)
	if host == "" {
		host = DEFAULT_LAUNCHDARKLY_HOST
	}
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}
	body := fmt.Sprintf(`{"environmentKey":%q,"resourceKind":"segment","required":true,"serviceKind":"launchdarkly","minNumApprovals":1}`, envKey)
	req, err := http.NewRequest(http.MethodPatch, host+"/api/v2/approval-requests/projects/"+projectKey+"/settings", strings.NewReader(body))
	if err != nil {
		t.Fatalf("build approval-settings request: %s", err)
	}
	req.Header.Set("Authorization", os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN))
	req.Header.Set("LD-API-Version", "beta")
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("enable segment approvals: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("enable segment approvals for %s/%s returned %d: %s", projectKey, envKey, resp.StatusCode, string(b))
	}
}

package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	ldapi "github.com/launchdarkly/api-client-go/v17"
	"github.com/stretchr/testify/require"
)

const (
	testAccFeatureFlagEnvironmentBasic = `
resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Basic feature flag"
	variation_type = "number"
	variations {
		value = 10
	}
	variations {
		value = 20
	}
	variations {
		value = 30
	}
}

resource "launchdarkly_feature_flag_environment" "basic" {
	flag_id 		  = launchdarkly_feature_flag.basic.id
	env_key 		  = "test"
	on = false
  	fallthrough {
    	variation = 1
  	}
	off_variation = 2
	targets {
		values    = ["user1"]
		variation = 0
	}
}
`

	testAccFeatureFlagEnvironmentEmpty = `
resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Basic feature flag"
	variation_type = "number"
	variations {
		value = 10
	}
	variations {
		value = 20
	}
	variations {
		value = 30
	}
}

resource "launchdarkly_feature_flag_environment" "basic" {
	flag_id 		  = launchdarkly_feature_flag.basic.id
	env_key 		  = "test"
	fallthrough {
		variation = 0
	}
	off_variation = 2
}
`

	testAccFeatureFlagEnvironmentUpdate = `	
resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Basic feature flag"
	variation_type = "number"
	variations {
		value = 0
	}
	variations {
		value = 10
	}
	variations {
		value = 30
	}
}

resource "launchdarkly_feature_flag_environment" "basic" {
	flag_id 		  = launchdarkly_feature_flag.basic.id
	env_key 		  = "test"
	on = true
	track_events = true
	targets {
		values    = ["user1", "user2"]
		variation = 1
	}
	rules {
		clauses {
			attribute = "country"
			op        = "startsWith"
			values    = ["great", "amazing"]
			negate    = false
		}
		variation = 0
	}
	rules {
		description = "names that start with 'h'"
		clauses {
			attribute = "name"
			op        = "startsWith"
			values    = ["h"]
			negate    = false
		}
		rollout_weights = [90000, 10000, 0]
		bucket_by = "email"
		context_kind = "account"
	}

	fallthrough {
		rollout_weights = [60000, 40000, 0]
		bucket_by = "email"
		context_kind = "user"
	}
	off_variation = 1
}
`

	testAccFeatureFlagEnvironmentJSONVariations = `
resource "launchdarkly_feature_flag" "json" {
	project_key    = launchdarkly_project.test.key
	key            = "json-flag"
	name           = "json flag"
	variation_type = "json"
	variations {
		value = jsonencode({ "foo" : "bar" })
	}
	variations {
		value = jsonencode({ "bar" : "foo", "bars" : "foos" })
	}
}

resource "launchdarkly_feature_flag_environment" "json_variations" {
	flag_id = launchdarkly_feature_flag.json.id
	env_key = "test"

	fallthrough {
		variation = 1
	}
	
	off_variation = 0
}
`

	testAccFeatureFlagEnvironmentPrereq = `
resource "launchdarkly_feature_flag" "bool" {
	project_key = launchdarkly_project.test.key
	key = "bool-flag"
	name = "boolean flag"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Basic feature flag"
	variation_type = "number"
	variations {
		value = 10
	}
	variations {
		value = 20
	}
	variations {
		value = 30
	}
}

resource "launchdarkly_feature_flag_environment" "prereq" {
	flag_id 		  = launchdarkly_feature_flag.basic.id
	env_key 		  = "test"
	on = true
	prerequisites {
		flag_key = launchdarkly_feature_flag.bool.key
		variation = 0
	}
	fallthrough {
		variation = 1
	}
	off_variation = 0
}
`

	testAccFeatureFlagEnvironmentRemovePrereq = `
resource "launchdarkly_feature_flag" "bool" {
	project_key = launchdarkly_project.test.key
	key = "bool-flag"
	name = "boolean flag"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Basic feature flag"
	variation_type = "number"
	variations {
		value = 10
	}
	variations {
		value = 20
	}
	variations {
		value = 30
	}
}

resource "launchdarkly_feature_flag_environment" "prereq" {
	flag_id 		  = launchdarkly_feature_flag.basic.id
	env_key 		  = "test"
	fallthrough {
		variation = 1
	}
	off_variation = 0
}
`

	testAccFeatureFlagEnvironmentBoolClauseValue = `
resource "launchdarkly_feature_flag" "bool_flag" {
	project_key = launchdarkly_project.test.key
	key = "bool-flag"
	name = "boolean flag"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag_environment" "bool_clause" {
	flag_id 		  = launchdarkly_feature_flag.bool_flag.id
	env_key 		  = "test"
	on = true
	rules {
		clauses {
			attribute  = "is_vip"
			op         = "startsWith"
			values     = [true]
			value_type = "boolean"
			negate     = false
		}
		variation = 0
	}
	fallthrough {
		variation = 0
	}
	off_variation = 1
}
`

	testAccFeatureFlagEnvironmentNumberClauseValue = `
resource "launchdarkly_feature_flag" "bool_flag" {
	project_key = launchdarkly_project.test.key
	key = "bool-flag"
	name = "boolean flag"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag_environment" "number_clause" {
	flag_id 		  = launchdarkly_feature_flag.bool_flag.id
	env_key 		  = "test"
	on = true
	rules {
		clauses {
			attribute  = "answer"
			op         = "in"
			values     = [42,84]
			value_type = "number"
			negate     = false
		}
		variation = 0
	}
	fallthrough {
		variation = 0
	}
	off_variation = 1
}
`

	testAccInvalidFallthroughBucketBy = `
resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Basic feature flag"
	variation_type = "number"
	variations {
		value = 10
	}
	variations {
		value = 20
	}
	variations {
		value = 30
	}
}

resource "launchdarkly_feature_flag_environment" "invalid_bucket_by" {
	flag_id 		  = launchdarkly_feature_flag.basic.id
	env_key 		  = "test"
	on = true
	  
	fallthrough {
		bucket_by = "email"
	}
	off_variation = 0
}
`

	testAccPercentageRollout = `
resource "launchdarkly_feature_flag" "rollout" {
	project_key    = launchdarkly_project.test.key
	key            = "bool-flag"
	name           = "Basic boolean flag"
	variation_type = "boolean"
  variations {
    value = true
  }
  variations {
    value = false
  }

  defaults {
    on_variation  = 1
    off_variation = 0
  }
}	

resource "launchdarkly_feature_flag_environment" "rollout" {
	flag_id = launchdarkly_feature_flag.rollout.id
	env_key = "test"
	on      = true	  
	rules {
		clauses {
			attribute = "country"
			op        = "startsWith"
			values    = ["aus", "nz", "united"]
			negate    = false
		}
		variation = 0
	}
	fallthrough {
		variation       = 0
		rollout_weights = [60000, 40000]
		bucket_by       = "country"
		context_kind = "other"
	}
  off_variation = 1
}
`

	testAccPercentageRolloutClauseUpdate = `
resource "launchdarkly_feature_flag" "rollout" {
	project_key    = launchdarkly_project.test.key
	key            = "bool-flag"
	name           = "Basic boolean flag"
	variation_type = "boolean"
  variations {
    value = true
  }
  variations {
    value = false
  }

  defaults {
    on_variation  = 1
    off_variation = 0
  }
}	

resource "launchdarkly_feature_flag_environment" "rollout" {
	flag_id = launchdarkly_feature_flag.rollout.id
	env_key = "test"
	on      = true	  
	rules {
		clauses {
			attribute = "country"
			op        = "startsWith"
			values    = ["aus", "us", "united"]
			negate    = false
		}
		variation = 0
	}
	fallthrough {
		variation       = 0
		rollout_weights = [60000, 40000]
		bucket_by       = "country"
		context_kind = "other"
	}
  off_variation = 1
}
`

	testAccInvalidRuleBucketBy = `
resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Basic feature flag"
	variation_type = "number"
	variations {
		value = 10
	}
	variations {
		value = 20
	}
	variations {
		value = 30
	}
}

resource "launchdarkly_feature_flag_environment" "invalid_bucket_by" {
	flag_id 		  = launchdarkly_feature_flag.basic.id
	env_key 		  = "test"
	on = true	  
	rules {
		clauses {
			attribute = "name"
			op        = "startsWith"
			values    = ["h"]
			negate    = false
		}
		variation = 0
		bucket_by = "name"
	}
	fallthrough {
		variation = 0
	}
	off_variation = 1
}
`

	testAccContextKind = `
resource "launchdarkly_feature_flag" "context_test" {
	project_key = "%s"
	key = "test-flag"
	name = "Context Kind Test Flag"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag_environment" "custom_context" {
	flag_id 		  = launchdarkly_feature_flag.context_test.id
	env_key 		  = "test"
	on = true
	off_variation = 0
	targets {
		values    = ["user1", "user2"]
		variation = 1
	}
	context_targets {
		values = ["account1", "account2"]
		variation = 0
		context_kind = "%s"
	}
	context_targets {
		values = ["other1", "other2"]
		variation = 1
		context_kind = "%s"
	}
	fallthrough {
		variation = 0
	}
}
`

	testAccContextKindUpdate = `
resource "launchdarkly_feature_flag" "context_test" {
	project_key = "%s"
	key = "test-flag"
	name = "Context Kind Test Flag"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag_environment" "custom_context" {
	flag_id 		  = launchdarkly_feature_flag.context_test.id
	env_key 		  = "test"
	on = true
	off_variation = 0
	targets {
		values    = ["user1"]
		variation = 0
	}
	targets {
		values    = ["user2"]
		variation = 1
	}
	context_targets {
		values = ["account1"]
		variation = 1
		context_kind = "%s"
	}
	fallthrough {
		variation = 0
	}
}
`

	testAccContextKindReorderTargets = `
resource "launchdarkly_feature_flag" "context_test" {
	project_key = "%s"
	key = "test-flag"
	name = "Context Kind Test Flag"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag_environment" "custom_context" {
	flag_id 		  = launchdarkly_feature_flag.context_test.id
	env_key 		  = "test"
	on = true
	off_variation = 0
	targets {
		values    = ["user2"]
		variation = 1
	}
	context_targets {
		values = ["account1"]
		variation = 1
		context_kind = "%s"
	}
	targets {
		values    = ["user1"]
		variation = 0
	}
	fallthrough {
		variation = 0
	}
}
`

	testAccFallthroughAndRulesContextKind = `
resource "launchdarkly_feature_flag" "context_test" {
	project_key = "%s"
	key = "test-flag"
	name = "Context Kind Test Flag"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag_environment" "rules_custom_context" {
	flag_id 		  = launchdarkly_feature_flag.context_test.id
	env_key 		  = "production"
	on = false
	off_variation = 1
	rules {
		clauses {
			attribute = "name"
			op = "startsWith"
			values = ["X", "O"]
			negate = true
		}
		clauses {
			attribute = "account_type"
			op = "matches"
			values = ["professional", "enterprise"]
			negate = false
			context_kind = "%s"
		}
		variation = 0
	}
	fallthrough {
		rollout_weights = [30000, 70000]
		bucket_by = "account_id"
	}
}
`

	testAccDefaultOffVariation = `
	resource "launchdarkly_feature_flag" "off_variation_test" {
		project_key    = launchdarkly_project.test.key
		key            = "off-variation-test-flag"
		name           = "off variation test"
		variation_type = "boolean"
	
		variations {
			value = false
		}
	
		variations {
			value = true
		}
	
		defaults {
			off_variation = 0
			on_variation  = 1
		}
	
		client_side_availability {
			using_environment_id = true
		}
	}
`

	testAccDefaultOffVariationOnDelete = `
resource "launchdarkly_feature_flag" "off_variation_test" {
	project_key    = launchdarkly_project.test.key
	key            = "off-variation-test-flag"
	name           = "off variation test"
	variation_type = "boolean"

	variations {
		value = false
	}

	variations {
		value = true
	}

	defaults {
		off_variation = 0
		on_variation  = 1
	}

	client_side_availability {
		using_environment_id = true
	}
}

resource "launchdarkly_feature_flag_environment" "off_variation_test_configuration" {
	flag_id       = launchdarkly_feature_flag.off_variation_test.id
	env_key       = "test"
	on            = true
	off_variation = 0

	targets {
		values    = ["context-value"]
		variation = 1
	}

	fallthrough {
		variation = 0
	}
}
`
)

func TestAccFeatureFlagEnvironment_Empty(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag_environment.basic"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEnvironmentEmpty),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ON, "false"),
					resource.TestCheckResourceAttr(resourceName, OFF_VARIATION, "2"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, TRACK_EVENTS, "false"),
					resource.TestCheckNoResourceAttr(resourceName, fmt.Sprintf("%s.0", RULES)),
					resource.TestCheckNoResourceAttr(resourceName, "rules.#"),
					resource.TestCheckNoResourceAttr(resourceName, fmt.Sprintf("%s.0", PREREQUISITES)),
					resource.TestCheckNoResourceAttr(resourceName, "prerequisites.#"),
					resource.TestCheckNoResourceAttr(resourceName, fmt.Sprintf("%s.0", TARGETS)),
					resource.TestCheckNoResourceAttr(resourceName, "targets.#"),
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

func TestAccFeatureFlagEnvironment_Update(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag_environment.basic"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEnvironmentBasic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ON, "false"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.rollout.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "targets.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "0"),
					resource.TestCheckResourceAttr(resourceName, OFF_VARIATION, "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEnvironmentUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, TRACK_EVENTS, "true"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.rollout_weights.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.rollout_weights.0", "60000"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.rollout_weights.1", "40000"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.rollout_weights.2", "0"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.bucket_by", "email"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.context_kind", "user"),
					resource.TestCheckResourceAttr(resourceName, "targets.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.1", "user2"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.description", ""),
					resource.TestCheckResourceAttr(resourceName, "rules.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", "country"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", "startsWith"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "great"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.1", "amazing"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.description", "names that start with 'h'"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.rollout_weights.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.rollout_weights.0", "90000"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.rollout_weights.1", "10000"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.rollout_weights.2", "0"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.bucket_by", "email"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.attribute", "name"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.op", "startsWith"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.values.0", "h"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, OFF_VARIATION, "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// After changes have been made to the resource, removing optional values should revert to their default / null values.
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEnvironmentEmpty),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ON, "false"),
					resource.TestCheckResourceAttr(resourceName, TRACK_EVENTS, "false"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, OFF_VARIATION, "2"),
					resource.TestCheckNoResourceAttr(resourceName, "targets.#"),
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

func TestAccFeatureFlagEnvironment_JSON_variations(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag_environment.json_variations"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEnvironmentJSONVariations),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ON, "false"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, OFF_VARIATION, "0"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{FALLTHROUGH, OFF_VARIATION},
			},
		},
	})
}

func TestAccFeatureFlagEnvironment_BoolClauseValue(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag_environment.bool_clause"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEnvironmentBoolClauseValue),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.value_type", "boolean"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "true"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, OFF_VARIATION, "1"),
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

func TestAccFeatureFlagEnvironment_NumberClauseValue(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag_environment.number_clause"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEnvironmentNumberClauseValue),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.value_type", "number"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "42"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.1", "84"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, OFF_VARIATION, "1"),
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

func TestAccFeatureFlagEnvironment_UpdateClauseWithRollout(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag_environment.rollout"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccPercentageRollout),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.value_type", "string"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "aus"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.1", "nz"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.2", "united"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.rollout_weights.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.rollout_weights.0", "60000"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.rollout_weights.1", "40000"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.bucket_by", "country"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.context_kind", "other"),
					resource.TestCheckResourceAttr(resourceName, OFF_VARIATION, "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccPercentageRolloutClauseUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.value_type", "string"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "aus"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.1", "us"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.2", "united"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.rollout_weights.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.rollout_weights.0", "60000"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.rollout_weights.1", "40000"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.bucket_by", "country"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.context_kind", "other"),
					resource.TestCheckResourceAttr(resourceName, OFF_VARIATION, "1"),
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

func TestAccFeatureFlagEnvironment_InvalidBucketBy(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	// resourceName := "launchdarkly_feature_flag_environment.invalid_bucket_by"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      withRandomProject(projectKey, testAccInvalidFallthroughBucketBy),
				ExpectError: regexp.MustCompile("cannot use bucket_by argument with variation, only with rollout_weights"),
			},
			{
				Config:      withRandomProject(projectKey, testAccInvalidRuleBucketBy),
				ExpectError: regexp.MustCompile("cannot use bucket_by argument with variation, only with rollout_weights"),
			},
		},
	})
}

func TestAccFeatureFlagEnvironment_Prereq(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag_environment.prereq"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEnvironmentPrereq),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "prerequisites.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "prerequisites.0.flag_key", "bool-flag"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, OFF_VARIATION, "0"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEnvironmentRemovePrereq),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ON, "false"),
					resource.TestCheckNoResourceAttr(resourceName, "prerequisites.#"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, OFF_VARIATION, "0"),
				),
			},
		},
	})
}

func TestAccFeatureFlagEnvironment_ContextTargets(t *testing.T) {
	// scaffold. we have to do it via API request because we do not yet have the ability to add context_kind resources
	// to projects via terraform
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)
	accountContextKind := "account"
	otherContextKind := "other"
	err = testAccProjectWithCustomContextKindScaffold(client, projectKey, []string{accountContextKind})
	require.NoError(t, err)
	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
		require.NoError(t, err)
	}()
	resourceName := "launchdarkly_feature_flag_environment.custom_context"
	resourceName2 := "launchdarkly_feature_flag_environment.rules_custom_context"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccContextKind, projectKey, accountContextKind, otherContextKind),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "targets.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.1", "user2"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.variation", "1"),
					resource.TestCheckNoResourceAttr(resourceName, "targets.0.context_kind"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.values.0", "account1"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.values.1", "account2"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.context_kind", accountContextKind),
					resource.TestCheckResourceAttr(resourceName, "context_targets.1.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.1.values.0", "other1"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.1.values.1", "other2"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.1.variation", "1"),
					// this should simply create a new context kind on the project and not error -
					// have confirmed via UI this is happening
					resource.TestCheckResourceAttr(resourceName, "context_targets.1.context_kind", otherContextKind),
					resource.TestCheckResourceAttr(resourceName, OFF_VARIATION, "0"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccContextKindUpdate, projectKey, accountContextKind),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "targets.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "targets.1.values.0", "user2"),
					resource.TestCheckResourceAttr(resourceName, "targets.1.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.values.0", "account1"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.context_kind", accountContextKind),
					resource.TestCheckNoResourceAttr(resourceName, "context_targets.1"),
					resource.TestCheckResourceAttr(resourceName, OFF_VARIATION, "0"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// this should be exactly the same as the previous one, as target reordering shouldn't matter
				Config: fmt.Sprintf(testAccContextKindReorderTargets, projectKey, accountContextKind),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "targets.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "targets.1.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "targets.1.values.0", "user2"),
					resource.TestCheckResourceAttr(resourceName, "targets.1.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.values.0", "account1"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.context_kind", accountContextKind),
					resource.TestCheckNoResourceAttr(resourceName, "context_targets.1"),
					resource.TestCheckResourceAttr(resourceName, OFF_VARIATION, "0"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// this will actually create a new feature flag env resource on a different environment (production)
				// against the same flag. the previous resource will be torn down
				Config: fmt.Sprintf(testAccFallthroughAndRulesContextKind, projectKey, accountContextKind),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName2),
					resource.TestCheckResourceAttr(resourceName2, ON, "false"),
					resource.TestCheckResourceAttr(resourceName2, OFF_VARIATION, "1"),
					resource.TestCheckResourceAttr(resourceName2, "rules.0.clauses.#", "2"),
					resource.TestCheckResourceAttr(resourceName2, "rules.0.clauses.0.attribute", "name"),
					resource.TestCheckResourceAttr(resourceName2, "rules.0.clauses.0.op", "startsWith"),
					resource.TestCheckResourceAttr(resourceName2, "rules.0.clauses.0.value_type", "string"), // should default even if not set
					resource.TestCheckResourceAttr(resourceName2, "rules.0.clauses.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName2, "rules.0.clauses.0.values.0", "X"),
					resource.TestCheckResourceAttr(resourceName2, "rules.0.clauses.0.values.1", "O"),
					resource.TestCheckResourceAttr(resourceName2, "rules.0.clauses.0.negate", "true"),
					resource.TestCheckResourceAttr(resourceName2, "rules.0.clauses.0.context_kind", "user"),
					resource.TestCheckResourceAttr(resourceName2, "rules.0.clauses.1.attribute", "account_type"),
					resource.TestCheckResourceAttr(resourceName2, "rules.0.clauses.1.op", "matches"),
					resource.TestCheckResourceAttr(resourceName2, "rules.0.clauses.1.value_type", "string"), // should default even if not set
					resource.TestCheckResourceAttr(resourceName2, "rules.0.clauses.1.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName2, "rules.0.clauses.1.values.0", "professional"),
					resource.TestCheckResourceAttr(resourceName2, "rules.0.clauses.1.values.1", "enterprise"),
					resource.TestCheckResourceAttr(resourceName2, "rules.0.clauses.1.negate", "false"),
					resource.TestCheckResourceAttr(resourceName2, "rules.0.clauses.1.context_kind", accountContextKind),
					resource.TestCheckResourceAttr(resourceName2, "rules.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName2, "fallthrough.0.rollout_weights.0", "30000"),
					resource.TestCheckResourceAttr(resourceName2, "fallthrough.0.rollout_weights.1", "70000"),
					resource.TestCheckResourceAttr(resourceName2, "fallthrough.0.context_kind", "user"), // this should be automatically set by the API
					resource.TestCheckResourceAttr(resourceName2, "fallthrough.0.bucket_by", "account_id"),
				),
			},
			{
				ResourceName:      resourceName2,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFeatureFlagEnvironment_OffVariationResetsToCorrectDefaultOnDelete(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	globalFlagResourceName := "launchdarkly_feature_flag.off_variation_test"
	resourceName := "launchdarkly_feature_flag_environment.off_variation_test_configuration"
	flagKey := "off-variation-test-flag"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccDefaultOffVariationOnDelete),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, OFF_VARIATION, "0"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, TRACK_EVENTS, "false"),
					resource.TestCheckResourceAttr(globalFlagResourceName, "defaults.0.off_variation", "0"),
					resource.TestCheckResourceAttr(globalFlagResourceName, "defaults.0.on_variation", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccDefaultOffVariation),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(globalFlagResourceName, "defaults.0.off_variation", "0"),
					resource.TestCheckResourceAttr(globalFlagResourceName, "defaults.0.on_variation", "1"),
					testAccCheckFeatureFlagEnvironmentDefaults(t, projectKey, flagKey),
				),
			},
			{
				ResourceName:      globalFlagResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckFeatureFlagEnvironmentExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		flagId, ok := rs.Primary.Attributes[FLAG_ID]
		if !ok {
			return fmt.Errorf("feature flag id not found: %s", resourceName)
		}
		projKey, flagKey, err := flagIdToKeys(flagId)
		if err != nil {
			return err
		}
		envKey, ok := rs.Primary.Attributes[ENV_KEY]
		if !ok {
			return fmt.Errorf("environent key not found: %s", resourceName)
		}
		client := testAccProvider.Meta().(*Client)
		_, _, err = client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projKey, flagKey).Env(envKey).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting feature flag environment. %s", err)
		}
		return nil
	}
}

// this is a bespoke helper function to check that the env config's off variation
// has defaulted to the expected global config variation (in this case 0)
func testAccCheckFeatureFlagEnvironmentDefaults(t *testing.T, projectKey, flagKey string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
		require.NoError(t, err)
		flag, _, err := client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, flagKey).Execute()
		require.NoError(t, err)
		envConfig := flag.Environments["test"]
		require.Equal(t, int32(0), *envConfig.OffVariation)
		return nil
	}
}

func testAccProjectWithCustomContextKindScaffold(client *Client, projectKey string, contextKindKeys []string) error {
	projectBody := ldapi.ProjectPost{
		Name: "Context Kind Test Project",
		Key:  projectKey,
	}
	_, err := testAccProjectScaffoldCreate(client, projectBody)
	if err != nil {
		return err
	}

	for _, key := range contextKindKeys {
		err := addContextKindToProject(client, projectKey, key)
		if err != nil {
			return err
		}
	}
	return nil
}

package launchdarkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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
	targets {
		values    = []
		variation = 2
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
		clauses {
			attribute = "name"
			op        = "startsWith"
			values    = ["h"]
			negate    = false
		}
		rollout_weights = [90000, 10000, 0]
		bucket_by = "email"
	}

	fallthrough {
		rollout_weights = [60000, 40000, 0]
		bucket_by = "email"
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
)

func TestAccFeatureFlagEnvironment_Basic(t *testing.T) {
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
					resource.TestCheckResourceAttr(resourceName, "on", "false"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "off_variation", "2"),
					resource.TestCheckResourceAttr(resourceName, "targets.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.variation", "0"),
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
					resource.TestCheckResourceAttr(resourceName, "on", "false"),
					resource.TestCheckResourceAttr(resourceName, "off_variation", "2"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "track_events", "false"),
					resource.TestCheckNoResourceAttr(resourceName, "rules"),
					resource.TestCheckNoResourceAttr(resourceName, "rules.#"),
					resource.TestCheckNoResourceAttr(resourceName, "prerequisites"),
					resource.TestCheckNoResourceAttr(resourceName, "prerequisites.#"),
					resource.TestCheckNoResourceAttr(resourceName, "targets"),
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
					resource.TestCheckResourceAttr(resourceName, "on", "false"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.rollout.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "targets.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "off_variation", "2"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEnvironmentUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "on", "true"),
					resource.TestCheckResourceAttr(resourceName, "track_events", "true"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.rollout_weights.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.rollout_weights.0", "60000"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.rollout_weights.1", "40000"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.rollout_weights.2", "0"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.bucket_by", "email"),
					resource.TestCheckResourceAttr(resourceName, "targets.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.1", "user2"),
					resource.TestCheckResourceAttr(resourceName, "targets.1.values.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "targets.1.variation", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", "country"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", "startsWith"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "great"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.1", "amazing"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.rollout_weights.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.rollout_weights.0", "90000"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.rollout_weights.1", "10000"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.rollout_weights.2", "0"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.bucket_by", "email"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.attribute", "name"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.op", "startsWith"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.values.0", "h"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, "off_variation", "1"),
				),
			},
			// After changes have been made to the resource, removing optional values should revert to their default / null values.
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEnvironmentEmpty),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "on", "false"),
					resource.TestCheckResourceAttr(resourceName, "track_events", "false"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "off_variation", "2"),
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
					resource.TestCheckResourceAttr(resourceName, "on", "false"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "off_variation", "0"),
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
					resource.TestCheckResourceAttr(resourceName, "on", "true"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.value_type", "boolean"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "true"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "off_variation", "1"),
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
					resource.TestCheckResourceAttr(resourceName, "on", "true"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.value_type", "number"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "42"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.1", "84"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "off_variation", "1"),
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
					resource.TestCheckResourceAttr(resourceName, "on", "true"),
					resource.TestCheckResourceAttr(resourceName, "prerequisites.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "prerequisites.0.flag_key", "bool-flag"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "off_variation", "0"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEnvironmentRemovePrereq),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "on", "false"),
					resource.TestCheckNoResourceAttr(resourceName, "prerequisites.#"),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "off_variation", "0"),
				),
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

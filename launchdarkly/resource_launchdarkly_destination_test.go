package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccDestinationCreateKinesis = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key     = "test"
	name        = "kinesis-dest"
	kind        = "kinesis"
	config = {
		region      = "us-east-1"
		role_arn    = "arn:aws:iam::123456789012:role/marketingadmin"
		stream_name = "cat-stream"
	}
	on = true
	tags = [ "terraform" ]
}
`
	testAccDestinationCreatePubsub = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key     = "test"
	name        = "pubsub-dest"
	kind        = "google-pubsub"
	config = {
		project = "test-project"
		topic   = "test-topic"
	}
	tags = [ "terraform" ]
}
`
	testAccDestinationCreateMparticle = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "test"
	name = "mparticle-dest"
	kind = "mparticle"
	config = {
		api_key       = "apiKeyfromMParticle"
		secret        = "mParticleSecret"
		user_identity = "customer_id"
		environment   = "production"
	}
	on = true
	tags = [ "terraform" ]
}
`

	testAccDestinationCreateSegment = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "test"
	name    = "segment-dest"
	kind    = "segment"
	config  = {
		write_key = "super-secret-write-key"
	}
	on = true
	tags = [ "terraform" ]
}
`

	testAccDestinationCreateAzureEventHubs = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "test"
	name    = "azure-event-hubs-dest"
	kind    = "azure-event-hubs"
	config  = {
		namespace = "namespace"
		name = "name"
		policy_name = "policy-name"
		policy_key = "super-secret-policy-key"
	}
	on = true
	tags = [ "terraform" ]
}
`

	testAccDestinationUpdateKinesis = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "test"
	name = "updated-kinesis-dest"
	kind = "kinesis"
	config = {
		region = "us-west-1",
		role_arn = "arn:aws:iam::123456789012:role/marketingadmin",
		stream_name = "cat-stream"
	}
	on = true
	tags = [ "terraform", "updated" ]
}
`
	testAccDestinationUpdatePubsub = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "test"
	name = "updated-pubsub-dest"
	kind = "google-pubsub"
	config = {
		"project": "renamed-project",
		"topic": "test-topic"
	}
	on = true
	tags = [ "terraform", "updated" ]
}
`

	testAccDestinationUpdateMparticle = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "test"
	name = "updated-mparticle-dest"
	kind = "mparticle"
	config = {
		api_key = "updatedApiKey"
		secret = "updatedSecret"
		user_identity = "customer_id"
		environment = "production"
	}
	on = true
	tags = [ "terraform", "updated" ]
}
`

	testAccDestinationUpdateSegment = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "test"
	name = "segment-dest"
	kind = "segment"
	config = {
		write_key = "updated-write-key"
	}
	tags = [ "terraform" ]
}
`

	testAccDestinationUpdateAzureEventHubs = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "test"
	name    = "updated-azure-event-hubs-dest"
	kind    = "azure-event-hubs"
	config  = {
		namespace = "namespace"
		name = "updated-name"
		policy_name = "updated-policy-name"
		policy_key = "updated-policy-key"
	}
	on = false
	tags = [ "terraform" ]
}
`
)

func TestAccDestination_CreateKinesis(t *testing.T) {
	// will implicitly test resourceDestinationRead
	// make sure you also test that the kind conforms to one of the three acceptable ones
	// kinesis, google-pubsub, or mparticle
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_destination.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccDestinationCreateKinesis),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "kinesis-dest"),
					resource.TestCheckResourceAttr(resourceName, "kind", "kinesis"),
					resource.TestCheckResourceAttr(resourceName, "config.region", "us-east-1"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerifyIgnore: []string{"tags"},
			},
		},
	})
}

func TestAccDestination_CreateMparticle(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_destination.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccDestinationCreateMparticle),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "mparticle-dest"),
					resource.TestCheckResourceAttr(resourceName, "kind", "mparticle"),
					resource.TestCheckResourceAttr(resourceName, "config.api_key", "apiKeyfromMParticle"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
		},
	})
}

func TestAccDestination_CreatePubsub(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_destination.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccDestinationCreatePubsub),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "pubsub-dest"),
					resource.TestCheckResourceAttr(resourceName, "kind", "google-pubsub"),
					resource.TestCheckResourceAttr(resourceName, "on", "false"),
					resource.TestCheckResourceAttr(resourceName, "config.project", "test-project"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
		},
	})
}

func TestAccDestination_CreateSegment(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_destination.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccDestinationCreateSegment),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "segment-dest"),
					resource.TestCheckResourceAttr(resourceName, "kind", "segment"),
					resource.TestCheckResourceAttr(resourceName, "config.write_key", "super-secret-write-key"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
		},
	})
}

func TestAccDestination_CreateAzureEventHubs(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_destination.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccDestinationCreateAzureEventHubs),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "azure-event-hubs-dest"),
					resource.TestCheckResourceAttr(resourceName, "kind", "azure-event-hubs"),
					resource.TestCheckResourceAttr(resourceName, "config.namespace", "namespace"),
					resource.TestCheckResourceAttr(resourceName, "config.name", "name"),
					resource.TestCheckResourceAttr(resourceName, "config.policy_name", "policy-name"),
					resource.TestCheckResourceAttr(resourceName, "config.policy_key", "super-secret-policy-key"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
		},
	})
}

func TestAccDestination_UpdateKinesis(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_destination.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccDestinationCreateKinesis),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "kinesis-dest"),
					resource.TestCheckResourceAttr(resourceName, "kind", "kinesis"),
					resource.TestCheckResourceAttr(resourceName, "config.role_arn", "arn:aws:iam::123456789012:role/marketingadmin"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccDestinationUpdateKinesis),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "updated-kinesis-dest"),
					resource.TestCheckResourceAttr(resourceName, "kind", "kinesis"),
					resource.TestCheckResourceAttr(resourceName, "config.role_arn", "arn:aws:iam::123456789012:role/marketingadmin"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
		},
	})
}

func TestAccDestination_UpdatePubsub(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_destination.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccDestinationCreatePubsub),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "pubsub-dest"),
					resource.TestCheckResourceAttr(resourceName, "kind", "google-pubsub"),
					resource.TestCheckResourceAttr(resourceName, "config.project", "test-project"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccDestinationUpdatePubsub),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "updated-pubsub-dest"),
					resource.TestCheckResourceAttr(resourceName, "kind", "google-pubsub"),
					resource.TestCheckResourceAttr(resourceName, "config.project", "renamed-project"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
		},
	})
}

func TestAccDestination_UpdateMparticle(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_destination.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccDestinationCreateMparticle),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "mparticle-dest"),
					resource.TestCheckResourceAttr(resourceName, "kind", "mparticle"),
					resource.TestCheckResourceAttr(resourceName, "config.secret", "mParticleSecret"),
					resource.TestCheckResourceAttr(resourceName, "config.environment", "production"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccDestinationUpdateMparticle),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "updated-mparticle-dest"),
					resource.TestCheckResourceAttr(resourceName, "kind", "mparticle"),
					resource.TestCheckResourceAttr(resourceName, "config.secret", "updatedSecret"),
					resource.TestCheckResourceAttr(resourceName, "config.environment", "production"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "updated"),
				),
			},
		},
	})
}

func TestAccDestination_UpdateSegment(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_destination.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccDestinationCreateSegment),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "segment-dest"),
					resource.TestCheckResourceAttr(resourceName, "kind", "segment"),
					resource.TestCheckResourceAttr(resourceName, "on", "true"),
					resource.TestCheckResourceAttr(resourceName, "config.write_key", "super-secret-write-key"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccDestinationUpdateSegment),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "segment-dest"),
					resource.TestCheckResourceAttr(resourceName, "kind", "segment"),
					resource.TestCheckResourceAttr(resourceName, "on", "false"), // should default to false when removed
					resource.TestCheckResourceAttr(resourceName, "config.write_key", "updated-write-key"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
		},
	})
}

func TestAccDestination_UpdateAzureEventHubs(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_destination.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccDestinationCreateAzureEventHubs),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "azure-event-hubs-dest"),
					resource.TestCheckResourceAttr(resourceName, "kind", "azure-event-hubs"),
					resource.TestCheckResourceAttr(resourceName, "config.namespace", "namespace"),
					resource.TestCheckResourceAttr(resourceName, "config.name", "name"),
					resource.TestCheckResourceAttr(resourceName, "config.policy_name", "policy-name"),
					resource.TestCheckResourceAttr(resourceName, "config.policy_key", "super-secret-policy-key"),
					resource.TestCheckResourceAttr(resourceName, "on", "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccDestinationUpdateAzureEventHubs),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "env_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "name", "updated-azure-event-hubs-dest"),
					resource.TestCheckResourceAttr(resourceName, "kind", "azure-event-hubs"),
					resource.TestCheckResourceAttr(resourceName, "config.namespace", "namespace"),
					resource.TestCheckResourceAttr(resourceName, "config.name", "updated-name"),
					resource.TestCheckResourceAttr(resourceName, "config.policy_name", "updated-policy-name"),
					resource.TestCheckResourceAttr(resourceName, "config.policy_key", "updated-policy-key"),
					resource.TestCheckResourceAttr(resourceName, "on", "false"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
		},
	})
}

func testAccCheckDestinationExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("destination ID not set")
		}
		projKey, ok := rs.Primary.Attributes[PROJECT_KEY]
		if !ok {
			return fmt.Errorf("destination project key not found: %s", resourceName)
		}
		envKey, ok := rs.Primary.Attributes[ENV_KEY]
		if !ok {
			return fmt.Errorf("destination environment key not found: %s", resourceName)
		}
		client := testAccProvider.Meta().(*Client)
		_, _, destID, err := destinationImportIDtoKeys(rs.Primary.ID)
		if err != nil {
			return err
		}
		_, _, err = client.ld.DataExportDestinationsApi.GetDestination(client.ctx, projKey, envKey, destID)
		if err != nil {
			return fmt.Errorf("error getting destination: %s", err)
		}
		return nil
	}
}

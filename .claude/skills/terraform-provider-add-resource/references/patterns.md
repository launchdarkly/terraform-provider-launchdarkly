# Code Patterns Reference

Full code templates for implementing resources in the LaunchDarkly Terraform provider. Read this file when implementing the steps from SKILL.md.

## Table of contents

- [Helper file patterns](#helper-file-patterns)
- [Resource file pattern](#resource-file-pattern)
- [Data source file pattern](#data-source-file-pattern)
- [Acceptance test patterns](#acceptance-test-patterns)
- [Doc template](#doc-template)
- [JSON-encoded fields](#json-encoded-fields)
- [Mutually exclusive fields (clear-both-then-set)](#mutually-exclusive-fields-clear-both-then-set)
- [Empty-default stripper](#empty-default-stripper)
- [Version-advancement retry](#version-advancement-retry)
- [Transient delete retry](#transient-delete-retry)
- [Last-child cascade delete](#last-child-cascade-delete)
- [Shared test helpers and cooldown](#shared-test-helpers-and-cooldown)

## Helper file patterns

### Shared schema function

```go
func base<Name>Schema(isDataSource bool) map[string]*schema.Schema {
    schemaMap := map[string]*schema.Schema{
        PROJECT_KEY: {
            Type:             schema.TypeString,
            Required:         true,
            ForceNew:         !isDataSource,
            Description:      addForceNewDescription("The project key.", !isDataSource),
            ValidateDiagFunc: validateKey(),
        },
        // Use `Computed: isDataSource` for fields that are Required in the resource
        // but read-only in the data source.
        //
        // Use ValidateDiagFunc (not ValidateFunc) so removeInvalidFieldsForDataSource
        // can clear it on Computed fields.
        //
        // Add ConflictsWith to mutually exclusive Optional fields.
    }

    if isDataSource {
        schemaMap = removeInvalidFieldsForDataSource(schemaMap)
    }

    return schemaMap
}
```

### Shared read logic

```go
func <name>Read(ctx context.Context, d *schema.ResourceData, meta interface{}, isDataSource bool) diag.Diagnostics {
    var diags diag.Diagnostics
    client := meta.(*Client)

    // Fetch from API ...

    // Handle 404 — remove from state for resources, error for data sources
    if isStatusNotFound(res) && !isDataSource {
        log.Printf("[WARN] failed to find <name> for project %q, removing from state", projectKey)
        diags = append(diags, diag.Diagnostic{
            Severity: diag.Warning,
            Summary:  fmt.Sprintf("[WARN] failed to find <name> for project %q, removing from state", projectKey),
        })
        d.SetId("")
        return diags
    }
    if err != nil {
        return diag.Errorf("failed to get <name> for project %q: %s", projectKey, handleLdapiErr(err))
    }

    // For data sources, set the ID here (resources set it on Create)
    if isDataSource {
        d.SetId(projectKey)
    }

    // Populate state
    _ = d.Set(PROJECT_KEY, projectKey)
    // ... set other fields ...

    return diags
}
```

### API wrapper — generated client (preferred)

```go
func get<Name>(client *Client, projectKey string) (*ldapi.<Name>Rep, *http.Response, error) {
    var result *ldapi.<Name>Rep
    var res *http.Response
    var err error
    err = client.withConcurrency(client.ctx, func() error {
        result, res, err = client.ld.<Api>.Get<Name>(client.ctx, projectKey).Execute()
        return err
    })
    return result, res, err
}
```

### API wrapper — beta API

```go
func get<Name>(client *Client, projectKey, key string) (*ldapi.<Name>, *http.Response, error) {
    var result *ldapi.<Name>
    var res *http.Response
    var err error
    err = client.withConcurrency(client.ctx, func() error {
        result, res, err = client.ld.<Api>BetaApi.Get<Name>(client.ctx, projectKey, key).
            LDAPIVersion("beta").
            Execute()
        return err
    })
    return result, res, err
}
```

### Payload builder

```go
func <name>PayloadFromResourceData(d *schema.ResourceData, ...) ldapi.<Payload>Type {
    // Extract fields from ResourceData
    // Build and return the API payload struct
}
```

## Resource file pattern

```go
func resource<Name>() *schema.Resource {
    return &schema.Resource{
        CreateContext: resource<Name>Create,
        ReadContext:   resource<Name>Read,
        UpdateContext: resource<Name>Update,
        DeleteContext: resource<Name>Delete,
        Exists:        resource<Name>Exists,

        Importer: &schema.ResourceImporter{
            StateContext: resource<Name>Import,
        },

        Description: `Provides a LaunchDarkly <name> resource.

This resource allows you to create and manage <name> within your LaunchDarkly project.`,

        Schema: base<Name>Schema(false),
    }
}
```

### Import function

```go
func resource<Name>Import(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
    parts := splitID(d.Id(), 2)
    if len(parts) != 2 {
        return nil, fmt.Errorf("import ID must be in the format project_key/resource_key")
    }
    _ = d.Set(PROJECT_KEY, parts[0])
    _ = d.Set(KEY, parts[1])
    return []*schema.ResourceData{d}, nil
}
```

## Data source file pattern

```go
func dataSource<Name>() *schema.Resource {
    return &schema.Resource{
        ReadContext: dataSource<Name>Read,

        Description: `Provides a LaunchDarkly <name> data source.

This data source allows you to retrieve <name> settings for a LaunchDarkly project.`,

        Schema: base<Name>Schema(true), // removeInvalidFieldsForDataSource called internally
    }
}

func dataSource<Name>Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
    return <name>Read(ctx, d, meta, true)
}
```

## Acceptance test patterns

### Resource test

```go
func TestAcc<Name>_CreateAndUpdate(t *testing.T) {
    projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
    resourceName := "launchdarkly_<name>.test"

    resource.Test(t, resource.TestCase{
        PreCheck:     func() { testAccPreCheck(t) },
        Providers:    testAccProviders,
        CheckDestroy: testAccCheck<Name>Destroy,
        Steps: []resource.TestStep{
            {
                Config: testAcc<Name>Config(projectKey),
                Check: resource.ComposeTestCheckFunc(
                    testAccCheck<Name>Exists(resourceName),
                    resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
                ),
            },
            {
                ResourceName:      resourceName,
                ImportState:       true,
                ImportStateVerify: true,
            },
            {
                Config: testAcc<Name>ConfigUpdate(projectKey),
                Check: resource.ComposeTestCheckFunc(
                    testAccCheck<Name>Exists(resourceName),
                    // ... updated attribute checks
                ),
            },
        },
    })
}
```

Required test helpers:

- `testAcc<Name>Config(projectKey string) string` — HCL config for create step. Must include a `launchdarkly_project` resource with a random key.
- `testAcc<Name>ConfigUpdate(projectKey string) string` — HCL config for update step.
- `testAccCheck<Name>Exists(resourceName string) resource.TestCheckFunc` — verifies resource exists via the API.
- `testAccCheck<Name>Destroy` — verifies all resources of this type are deleted after the test:

```go
var testAccCheck<Name>Destroy = func(s *terraform.State) error {
    client := testAccProvider.Meta().(*Client)
    for addr, rs := range s.RootModule().Resources {
        // Skip data sources — they don't own remote objects
        if strings.HasPrefix(addr, "data.") {
            continue
        }
        if rs.Type != "launchdarkly_<name>" {
            continue
        }
        projectKey := rs.Primary.Attributes[PROJECT_KEY]
        key := rs.Primary.Attributes[KEY]

        _, res, err := client.ld.<Api>.Get<Name>(client.ctx, projectKey, key).Execute()
        if isStatusNotFound(res) {
            continue
        }
        if err != nil {
            return err
        }
        return fmt.Errorf("<name> %s/%s still exists", projectKey, key)
    }
    return nil
}
```

### Data source test

Data source tests scaffold a project via the API (not Terraform), then read it back. Data source tests generally do not need `CheckDestroy` since they don't create remote objects:

```go
func TestAccDataSource<Name>_exists(t *testing.T) {
    accTest := os.Getenv("TF_ACC")
    if accTest == "" {
        t.SkipNow()
    }

    projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
    client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
    require.NoError(t, err)

    projectBody := ldapi.ProjectPost{Name: "<Name> DS Test", Key: projectKey}
    _, err = testAccProjectScaffoldCreate(client, projectBody)
    require.NoError(t, err)
    defer func() {
        err := testAccProjectScaffoldDelete(client, projectKey)
        require.NoError(t, err)
    }()

    resourceName := "data.launchdarkly_<name>.test"
    resource.Test(t, resource.TestCase{
        PreCheck:  func() { testAccPreCheck(t) },
        Providers: testAccProviders,
        Steps: []resource.TestStep{
            {
                Config: fmt.Sprintf(`
data "launchdarkly_<name>" "test" {
    project_key = "%s"
}
`, projectKey),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
                ),
            },
        },
    })
}
```

## JSON-encoded fields

For API fields typed as `map[string]interface{}` or arbitrary JSON Schema, expose them as `schema.TypeString` and use the helpers in `json_helper.go`.

### Schema declaration

```go
JSON_FIELD: {
    Type:             schema.TypeString,
    Optional:         !isDataSource,
    Computed:         isDataSource,
    Description:      "A JSON string representing the <thing>.",
    ValidateDiagFunc: emptyValueIfDataSource(validateJsonStringDiagFunc(), isDataSource),
    DiffSuppressFunc: emptyValueIfDataSource(suppressEquivalentJsonDiffs, isDataSource),
},
```

If the field must also be valid JSON Schema (not just valid JSON), swap in `validateJsonSchemaStringDiagFunc()`.

### HCL ↔ API conversion

```go
// HCL string → API map (write path)
m, err := jsonStringToMap(d.Get(JSON_FIELD).(string))
if err != nil {
    return diag.Errorf("failed to parse %s JSON: %s", JSON_FIELD, err)
}
patch.<Field> = m

// API map → HCL string (read path), stripping API-injected defaults.
// Only strip if the API is known to inject defaults — otherwise pass through unchanged.
cleaned := stripEmptyMapValues(apiResp.<Field>)
if len(cleaned) > 0 && !isEmpty<Field>Map(cleaned) {
    s, err := mapToJsonString(cleaned)
    if err != nil {
        return diag.Errorf("failed to serialize %s: %s", JSON_FIELD, err)
    }
    _ = d.Set(JSON_FIELD, s)
} else {
    _ = d.Set(JSON_FIELD, nil)
}
```

## Mutually exclusive fields (clear-both-then-set)

When two `Optional` fields share `ConflictsWith`, always clear both before setting the one returned by the API. Otherwise switching from variant A to variant B leaves stale state on A and produces a perpetual diff.

```go
// FIELD_A and FIELD_B conflict with each other.
// Clear both first; then set whichever the API returned.
_ = d.Set(FIELD_A, "")
_ = d.Set(FIELD_B, "")

if resp.Owner != nil {
    if resp.Owner.A != nil {
        _ = d.Set(FIELD_A, *resp.Owner.A)
    } else if resp.Owner.B != nil {
        _ = d.Set(FIELD_B, *resp.Owner.B)
    }
}
```

## Empty-default stripper

Some APIs return synthetic "empty" objects for optional fields the user never set (an object whose every nested value is the zero value). Storing that in state produces a perpetual diff against the user's HCL. Strip them before `d.Set`.

`json_helper.go` already exposes `stripEmptyMapValues` (see source). Pair it with a per-field `isEmpty<Field>Map` if the API also wraps the empty object in a non-empty outer key set:

```go
// isEmpty<Field>Map returns true if every value in the map is empty/zero —
// use to detect the API's synthetic default object for <Field>.
func isEmpty<Field>Map(m map[string]interface{}) bool {
    for _, v := range m {
        switch val := v.(type) {
        case string:
            if val != "" {
                return false
            }
        case map[string]interface{}:
            if len(val) > 0 {
                return false
            }
        default:
            if v != nil {
                return false
            }
        }
    }
    return true
}
```

Only add this when you've actually seen the API inject defaults — don't apply it speculatively, since a too-aggressive stripper can hide real user data.

## Version-advancement retry

For APIs that create a new version on every write (rather than updating in place), the GET endpoint may briefly return a stale version. Use a retry loop keyed on the version field — more reliable than a fixed sleep.

```go
import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

func resource<Name>ReadWithRetry(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
    previousVersion := d.Get(VERSION).(int)
    const maxAttempts = 10
    attempt := 0

    return diag.FromErr(retry.RetryContext(ctx, 30*time.Second, func() *retry.RetryError {
        attempt++
        diags := <name>Read(ctx, d, metaRaw, false)
        if diags.HasError() {
            return retry.NonRetryableError(fmt.Errorf("%s", diags[0].Summary))
        }

        // On Create previousVersion is 0 — any returned version is fine.
        // On Update wait until the version moves past what we had before the write.
        currentVersion := d.Get(VERSION).(int)
        if previousVersion > 0 && currentVersion <= previousVersion {
            if attempt >= maxAttempts {
                return retry.NonRetryableError(fmt.Errorf("%s %q: version did not advance past %d after %d reads (current %d)",
                    "<name>", d.Get(KEY).(string), previousVersion, maxAttempts, currentVersion))
            }
            return retry.RetryableError(fmt.Errorf("waiting for version to advance past %d (currently %d)", previousVersion, currentVersion))
        }

        return nil
    }))
}
```

Call this from Create and Update instead of the plain read function.

## Transient delete retry

**Use only when an actual transient failure mode is observed in the field.** Some endpoints return non-deterministic 4xx errors on delete that succeed on retry (typically when an internal rate limit or async cleanup leaks through as a synchronous error). Wrap the delete in `retry.RetryContext` and gate retries on a tight body-substring check so genuine validation errors still surface immediately.

```go
const <name>DeleteRetryTimeout = 45 * time.Second

func resource<Name>Delete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
    var diags diag.Diagnostics
    client := metaRaw.(*Client)
    projectKey := d.Get(PROJECT_KEY).(string)
    key := d.Get(KEY).(string)

    var lastErr error
    err := retry.RetryContext(ctx, <name>DeleteRetryTimeout, func() *retry.RetryError {
        var res *http.Response
        deleteErr := client.withConcurrency(client.ctx, func() error {
            var err error
            res, err = client.ld.<Api>.Delete<Name>(client.ctx, projectKey, key).Execute()
            return err
        })

        if deleteErr == nil || isStatusNotFound(res) {
            return nil
        }
        if shouldRetry<Name>Delete(res, deleteErr) {
            lastErr = deleteErr
            return retry.RetryableError(deleteErr)
        }
        lastErr = deleteErr
        return retry.NonRetryableError(deleteErr)
    })
    if err != nil {
        if lastErr != nil {
            err = lastErr
        }
        return diag.Errorf("failed to delete <name> with key %q in project %q: %s", key, projectKey, handleLdapiErr(err))
    }
    return diags
}

// shouldRetry<Name>Delete narrows retries to known-transient bodies. Keep the
// substring tight — a broad match will hide real validation errors. Match the
// status code AND a phrase unique to the transient failure mode.
func shouldRetry<Name>Delete(res *http.Response, err error) bool {
    if err == nil || res == nil || res.StatusCode != http.StatusBadRequest {
        return false
    }
    msg := strings.ToLower(handleLdapiErr(err).Error())
    return strings.Contains(msg, "<unique transient phrase>")
}
```

## Last-child cascade delete

When a parent's delete cascades to children, child delete may race the cascade and hit "Cannot delete the last <child>". Treat it as success — the parent will clean up.

```go
func resource<Child>Delete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
    var diags diag.Diagnostics
    client := metaRaw.(*Client)
    projectKey := d.Get(PROJECT_KEY).(string)
    parentKey := d.Get(PARENT_KEY).(string)
    key := d.Get(KEY).(string)

    var res *http.Response
    err := client.withConcurrency(client.ctx, func() error {
        var err error
        res, err = client.ld.<Api>.Delete<Child>(client.ctx, projectKey, parentKey, key).Execute()
        return err
    })
    if err != nil {
        if isStatusNotFound(res) {
            return diags // parent already cascaded
        }
        // handleLdapiErr is required — GenericOpenAPIError.Error() omits the body.
        if strings.Contains(handleLdapiErr(err).Error(), "Cannot delete the last <child>") {
            log.Printf("[WARN] cannot delete last <child> %q in <parent> %q project %q — will be removed when parent is deleted", key, parentKey, projectKey)
            return diags
        }
        return diag.Errorf("failed to delete <child> with key %q in <parent> %q project %q: %s", key, parentKey, projectKey, handleLdapiErr(err))
    }
    return diags
}
```

## Shared test helpers and cooldown

When multiple resource/data-source test files in the same domain share scaffolding, extract them into `<domain>_test_helpers_test.go`. Bake in a small cooldown only when the underlying API is rate-limit-prone in a way the standard retry client can't absorb.

```go
package launchdarkly

import (
    "fmt"
    "time"
)

// <domain>TestCooldown adds a brief delay between tests in this domain.
// These tests use resource.Test (serial) instead of resource.ParallelTest
// because <document the specific rate-limit reason here — e.g. an internal
// fan-out to a more constrained endpoint, or a status-code translation that
// hides the underlying 429 from the retry client>. Always document the *why*
// so a future reader doesn't strip the cooldown.
func <domain>TestCooldown() {
    time.Sleep(2 * time.Second)
}

// with<Domain>TestProject wraps a Terraform config string with a random project resource.
func with<Domain>TestProject(projectKey, resource string) string {
    return fmt.Sprintf(`
resource "launchdarkly_project" "test" {
    key  = "%s"
    name = "<Domain> Test Project"
    environments {
        name  = "Test Environment"
        key   = "test-env"
        color = "000000"
    }
}

%s`, projectKey, resource)
}
```

Call the cooldown at the very top of each test:

```go
func TestAcc<Name>_CreateAndUpdate(t *testing.T) {
    <domain>TestCooldown()
    // ... rest of test
}
```

## Doc template

Only needed if customizing beyond auto-generation. Create as `templates/resources/<name>.md.tmpl`:

```
---
page_title: "{{.Name}} {{.Type}} - {{.ProviderName}}"
subcategory: ""
description: |-
{{ .Description | plainmarkdown | trimspace | prefixlines "  " }}
---

# {{.Name}} ({{.Type}})

{{ .Description | trimspace }}

## Example Usage

{{ tffile (printf "examples/resources/%s/resource.tf" .Name)}}

{{ .SchemaMarkdown | trimspace }}

## Import

Import is supported using the following syntax:

{{ codefile "sh" .ImportFile | trimspace }}
```

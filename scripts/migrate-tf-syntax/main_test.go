package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func parseBody(t *testing.T, src string) (*hclwrite.File, *hclwrite.Body) {
	t.Helper()
	f, diag := hclwrite.ParseConfig([]byte(src), "test.tf", hcl.Pos{Line: 1, Column: 1})
	if diag.HasErrors() {
		t.Fatalf("parse fixture: %s", diag)
	}
	blocks := f.Body().Blocks()
	if len(blocks) == 0 {
		t.Fatal("fixture has no blocks")
	}
	return f, blocks[0].Body()
}

func TestCollectTFFilesRecursive(t *testing.T) {
	root := t.TempDir()
	mk := func(rel string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("# tf\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("main.tf")
	mk("modules/flags/flags.tf")
	mk(".terraform/modules/cached/cached.tf")
	mk(".git/objects/fake.tf")
	mk("README.md")

	flat, err := collectTFFiles(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(flat) != 1 {
		t.Errorf("non-recursive = %d files, want 1: %v", len(flat), flat)
	}

	rec, err := collectTFFiles(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec) != 2 {
		t.Errorf("recursive = %d files, want 2 (main + module, no caches): %v", len(rec), rec)
	}
	for _, f := range rec {
		if strings.Contains(f, ".terraform") || strings.Contains(f, ".git") {
			t.Errorf("recursive walk leaked cache path %s", f)
		}
	}
}

func TestForwardSkipsDynamicBlocks(t *testing.T) {
	src := `resource "launchdarkly_feature_flag" "f" {
  dynamic "variations" {
    for_each = var.vals
    content {
      value = variations.value
    }
  }
  variations {
    value = "static"
  }
  tags = ["a"]
}
`
	warningsBefore := warningCount
	f, body := parseBody(t, src)
	changed := forward(body, []*AttrSpec{{Name: "variations"}}, "test.tf: resource launchdarkly_feature_flag.f")
	if changed {
		t.Error("forward must skip a name with a dynamic generator, even when static siblings exist")
	}
	if warningCount != warningsBefore+1 {
		t.Errorf("warningCount delta = %d, want 1", warningCount-warningsBefore)
	}
	out := string(f.Bytes())
	if !strings.Contains(out, `dynamic "variations"`) || !strings.Contains(out, "value = \"static\"") {
		t.Errorf("body must be left untouched, got:\n%s", out)
	}
}

func TestForwardConvertsStaticBlocks(t *testing.T) {
	src := `resource "launchdarkly_feature_flag" "f" {
  variations {
    value = "true"
  }
  variations {
    value = "false"
  }
}
`
	f, body := parseBody(t, src)
	if !forward(body, []*AttrSpec{{Name: "variations"}}, "test.tf") {
		t.Fatal("expected conversion")
	}
	out := string(hclwrite.Format(f.Bytes()))
	if !strings.Contains(out, "variations = [{") {
		t.Errorf("expected nested-attribute syntax, got:\n%s", out)
	}
}

func TestApplyDeprecationsRename(t *testing.T) {
	src := `resource "launchdarkly_access_token" "t" {
  policy_statements = [{ actions = ["*"] }]
}
`
	f, body := parseBody(t, src)
	if !applyDeprecations(body, []*DeprecationSpec{{Name: "policy_statements", Action: "rename", To: "inline_roles"}}, "test.tf") {
		t.Fatal("expected rename")
	}
	out := string(hclwrite.Format(f.Bytes()))
	if !strings.Contains(out, "inline_roles") || strings.Contains(out, "policy_statements") {
		t.Errorf("rename failed:\n%s", out)
	}
}

func TestApplyDeprecationsIISToCSA(t *testing.T) {
	src := `resource "launchdarkly_feature_flag" "f" {
  include_in_snippet = var.snippet
}
`
	f, body := parseBody(t, src)
	if !applyDeprecations(body, []*DeprecationSpec{{Name: "include_in_snippet", Action: "iis_to_csa", To: "client_side_availability"}}, "test.tf") {
		t.Fatal("expected conversion")
	}
	out := string(hclwrite.Format(f.Bytes()))
	for _, want := range []*regexp.Regexp{
		// client_side_availability is a single object (SingleNestedAttribute) in v3.
		regexp.MustCompile(`client_side_availability = \{`),
		regexp.MustCompile(`using_environment_id\s+= var\.snippet`),
		regexp.MustCompile(`using_mobile_key\s+= false`),
	} {
		if !want.MatchString(out) {
			t.Errorf("missing %v in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "client_side_availability = [{") {
		t.Errorf("must not emit list syntax for the single-object client_side_availability:\n%s", out)
	}
}

func TestForwardConvertsObjectBlock(t *testing.T) {
	src := `resource "launchdarkly_feature_flag" "f" {
  client_side_availability {
    using_environment_id = true
    using_mobile_key     = false
  }
}
`
	f, body := parseBody(t, src)
	if !forward(body, []*AttrSpec{{Name: "client_side_availability", Object: true}}, "test.tf") {
		t.Fatal("expected conversion")
	}
	out := string(hclwrite.Format(f.Bytes()))
	if !strings.Contains(out, "client_side_availability = {") {
		t.Errorf("expected single-object syntax, got:\n%s", out)
	}
	if strings.Contains(out, "client_side_availability = [{") {
		t.Errorf("object attribute must not be wrapped in a list:\n%s", out)
	}
	if _, diag := hclwrite.ParseConfig([]byte(out), "out.tf", hcl.Pos{Line: 1, Column: 1}); diag.HasErrors() {
		t.Errorf("converted output does not parse: %s", diag)
	}
}

func TestReverseObjectBlock(t *testing.T) {
	src := `resource "launchdarkly_feature_flag" "f" {
  client_side_availability = {
    using_environment_id = true
    using_mobile_key     = false
  }
}
`
	f, body := parseBody(t, src)
	if !reverse(body, []*AttrSpec{{Name: "client_side_availability", Object: true}}) {
		t.Fatal("expected reverse conversion")
	}
	out := string(hclwrite.Format(f.Bytes()))
	if !regexp.MustCompile(`client_side_availability\s*\{`).MatchString(out) {
		t.Errorf("expected block syntax, got:\n%s", out)
	}
	if strings.Contains(out, "client_side_availability = {") || strings.Contains(out, "client_side_availability = [") {
		t.Errorf("reverse must drop the attribute-assignment form:\n%s", out)
	}
	if _, diag := hclwrite.ParseConfig([]byte(out), "out.tf", hcl.Pos{Line: 1, Column: 1}); diag.HasErrors() {
		t.Errorf("reversed output does not parse: %s", diag)
	}
}

func TestForwardConvertsMapBlock(t *testing.T) {
	src := `resource "launchdarkly_project" "p" {
  environments {
    key   = "production"
    name  = "Production"
    color = "417505"
    approval_settings {
      required          = true
      min_num_approvals = 2
    }
  }
  environments {
    key   = "test"
    name  = "Test"
    color = "f5a623"
  }
}
`
	f, body := parseBody(t, src)
	spec := []*AttrSpec{{Name: "environments", MapKey: "key", Nested: []*AttrSpec{{Name: "approval_settings", Object: true}}}}
	if !forward(body, spec, "test.tf") {
		t.Fatal("expected conversion")
	}
	out := string(hclwrite.Format(f.Bytes()))
	for _, want := range []string{
		"environments = {",
		`"production" = {`,
		`"test" = {`,
		"approval_settings = {",
		// the key attribute is kept inside each object (Optional+Computed in
		// v3, equals the map key) — it is NOT stripped.
		`key   = "production"`,
		`key   = "test"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "environments = [{") {
		t.Errorf("map attribute must not be a list:\n%s", out)
	}
	if _, diag := hclwrite.ParseConfig([]byte(out), "out.tf", hcl.Pos{Line: 1, Column: 1}); diag.HasErrors() {
		t.Errorf("converted output does not parse: %s", diag)
	}
}

func TestReverseMapBlock(t *testing.T) {
	// v3 map as the forward pass produces it: `key` kept inside each object.
	src := `resource "launchdarkly_project" "p" {
  environments = {
    "production" = {
      key   = "production"
      name  = "Production"
      color = "417505"
      approval_settings = {
        required          = true
        min_num_approvals = 2
      }
    }
    "test" = {
      key   = "test"
      name  = "Test"
      color = "f5a623"
    }
  }
}
`
	f, body := parseBody(t, src)
	spec := []*AttrSpec{{Name: "environments", MapKey: "key", Nested: []*AttrSpec{{Name: "approval_settings", Object: true}}}}
	if !reverse(body, spec) {
		t.Fatal("expected reverse conversion")
	}
	out := string(hclwrite.Format(f.Bytes()))
	if n := strings.Count(out, "environments {"); n != 2 {
		t.Errorf("expected 2 environments blocks, got %d:\n%s", n, out)
	}
	for _, want := range []string{
		`key   = "production"`,
		`key   = "test"`,
		"approval_settings {",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// no duplicate key (the kept key is reused, not re-injected on top).
	if strings.Count(out, `key   = "production"`) != 1 {
		t.Errorf("expected exactly one production key, got:\n%s", out)
	}
	if strings.Contains(out, "environments = {") || strings.Contains(out, "environments = [") {
		t.Errorf("reverse must drop the map-assignment form:\n%s", out)
	}
	if _, diag := hclwrite.ParseConfig([]byte(out), "out.tf", hcl.Pos{Line: 1, Column: 1}); diag.HasErrors() {
		t.Errorf("reversed output does not parse: %s", diag)
	}
}

func TestReverseMapBlockReinjectsMissingKey(t *testing.T) {
	// A hand-written v3 map that omitted the inner key: reverse must re-inject
	// `key = <map key>` so the v2 block is valid.
	src := `resource "launchdarkly_project" "p" {
  environments = {
    "production" = {
      name  = "Production"
      color = "417505"
    }
  }
}
`
	f, body := parseBody(t, src)
	spec := []*AttrSpec{{Name: "environments", MapKey: "key"}}
	if !reverse(body, spec) {
		t.Fatal("expected reverse conversion")
	}
	out := string(hclwrite.Format(f.Bytes()))
	if !strings.Contains(out, `key = "production"`) && !strings.Contains(out, `key   = "production"`) {
		t.Errorf("missing re-injected key, got:\n%s", out)
	}
	if _, diag := hclwrite.ParseConfig([]byte(out), "out.tf", hcl.Pos{Line: 1, Column: 1}); diag.HasErrors() {
		t.Errorf("reversed output does not parse: %s", diag)
	}
}

func TestForwardMapSkipsDuplicateKey(t *testing.T) {
	// Two environments blocks with the same key cannot become a map (duplicate
	// keys) — abort with the file untouched rather than silently drop one.
	src := `resource "launchdarkly_project" "p" {
  environments {
    key   = "production"
    name  = "Production"
    color = "000000"
  }
  environments {
    key   = "production"
    name  = "Dup"
    color = "111111"
  }
}
`
	before := warningCount
	f, body := parseBody(t, src)
	spec := []*AttrSpec{{Name: "environments", MapKey: "key"}}
	if forward(body, spec, "test.tf") {
		t.Error("forward must skip when two blocks share a map key")
	}
	if warningCount != before+1 {
		t.Errorf("warningCount delta = %d, want 1", warningCount-before)
	}
	if strings.Contains(string(f.Bytes()), "environments = {") {
		t.Errorf("must not emit a lossy map, got:\n%s", string(f.Bytes()))
	}
}

func TestWarnEnvIndexRefs(t *testing.T) {
	src := []byte(`resource "launchdarkly_segment" "s" {
  a = launchdarkly_project.ex.environments[1].key
  b = launchdarkly_project.ex.environments[0].client_side_id
  c = launchdarkly_project.ex.environments[*].client_side_id
}
`)
	before := warningCount
	warnEnvIndexRefs("t.tf", src, map[string][]string{"ex": {`"production"`, `"test"`}})
	if got := warningCount - before; got != 3 {
		t.Errorf("expected 3 warnings (indices 1, 0, splat), got %d", got)
	}
}

func TestCollectProjectEnvKeysKeepsNonLiteralIndices(t *testing.T) {
	// A non-literal key must still occupy its list slot, or every later
	// index resolves to the wrong env key in the warning text.
	path := filepath.Join(t.TempDir(), "p.tf")
	src := `resource "launchdarkly_project" "p" {
  environments {
    key = "production"
  }
  environments {
    key = local.env_key
  }
  environments {
    key = "qa"
  }
}
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	keys := collectProjectEnvKeys([]string{path})["p"]
	want := []string{`"production"`, `local.env_key`, `"qa"`}
	if len(keys) != len(want) {
		t.Fatalf("got %v, want %v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Errorf("keys[%d] = %q, want %q", i, keys[i], want[i])
		}
	}
}

// collapseSpaces normalizes runs of spaces to one, mirroring what terraform
// fmt does to the tool's output (the emitter pads tokens with extra spaces).
func collapseSpaces(s string) string {
	return regexp.MustCompile(` +`).ReplaceAllString(s, " ")
}

func TestForwardMapConvertsNonLiteralKey(t *testing.T) {
	src := `resource "launchdarkly_project" "p" {
  environments {
    key   = local.env_key
    name  = "X"
    color = "000000"
  }
}
`
	warningsBefore := warningCount
	f, body := parseBody(t, src)
	spec := []*AttrSpec{{Name: "environments", MapKey: "key"}}
	if !forward(body, spec, "test.tf: resource launchdarkly_project.p") {
		t.Error("forward must convert a map whose key is a non-literal expression")
	}
	if warningCount != warningsBefore {
		t.Errorf("warningCount delta = %d, want 0 (a note, not a warning)", warningCount-warningsBefore)
	}
	out := collapseSpaces(string(f.Bytes()))
	if !strings.Contains(out, "(local.env_key) = {") {
		t.Errorf("expected a parenthesized-expression map key, got:\n%s", out)
	}
	if !strings.Contains(out, "key = local.env_key") {
		t.Errorf("inner key attribute must be preserved, got:\n%s", out)
	}
}

func TestForwardMapMixedLiteralAndExpressionKeys(t *testing.T) {
	src := `resource "launchdarkly_project" "p" {
  environments {
    key   = "production"
    name  = "Production"
    color = "000000"
  }
  environments {
    key   = local.staging_key
    name  = "Staging"
    color = "111111"
  }
}
`
	warningsBefore := warningCount
	f, body := parseBody(t, src)
	spec := []*AttrSpec{{Name: "environments", MapKey: "key"}}
	if !forward(body, spec, "test.tf: resource launchdarkly_project.p") {
		t.Error("forward must convert a map mixing literal and expression keys")
	}
	if warningCount != warningsBefore {
		t.Errorf("warningCount delta = %d, want 0", warningCount-warningsBefore)
	}
	out := collapseSpaces(string(f.Bytes()))
	if !strings.Contains(out, `"production" = {`) || !strings.Contains(out, "(local.staging_key) = {") {
		t.Errorf("expected both map keys, got:\n%s", out)
	}
	if strings.Contains(out, "environments {") {
		t.Errorf("no blocks may remain, got:\n%s", out)
	}
}

func TestForwardMapSkipsDuplicateExpressionKey(t *testing.T) {
	// Two blocks keyed by the same expression text would collapse into one
	// map entry — abort like the literal duplicate case.
	src := `resource "launchdarkly_project" "p" {
  environments {
    key  = local.env_key
    name = "A"
  }
  environments {
    key  = local.env_key
    name = "B"
  }
}
`
	warningsBefore := warningCount
	f, body := parseBody(t, src)
	spec := []*AttrSpec{{Name: "environments", MapKey: "key"}}
	if forward(body, spec, "test.tf: resource launchdarkly_project.p") {
		t.Error("forward must skip duplicate expression keys")
	}
	if warningCount != warningsBefore+1 {
		t.Errorf("warningCount delta = %d, want 1", warningCount-warningsBefore)
	}
	if out := string(f.Bytes()); strings.Contains(out, "environments = {") {
		t.Errorf("must not emit a partial map:\n%s", out)
	}
}

func TestEnsureBooleanVariations(t *testing.T) {
	rule := []*DeprecationSpec{{Name: "variations", Action: "ensure_boolean_variations"}}

	t.Run("synthesizes for literal boolean", func(t *testing.T) {
		src := `resource "launchdarkly_feature_flag" "f" {
  variation_type = "boolean"
}
`
		f, body := parseBody(t, src)
		if !applyDeprecations(body, rule, "test.tf") {
			t.Fatal("expected synthesis")
		}
		out := string(hclwrite.Format(f.Bytes()))
		for _, want := range []string{"variations = [", `value = "true"`, `value = "false"`} {
			if !strings.Contains(out, want) {
				t.Errorf("missing %q in:\n%s", want, out)
			}
		}
		if _, diag := hclwrite.ParseConfig([]byte(out), "out.tf", hcl.Pos{Line: 1, Column: 1}); diag.HasErrors() {
			t.Errorf("synthesized output does not parse: %s", diag)
		}
	})

	t.Run("skips when variations already present", func(t *testing.T) {
		src := `resource "launchdarkly_feature_flag" "f" {
  variation_type = "boolean"
  variations     = [{ value = "true" }, { value = "false" }]
}
`
		_, body := parseBody(t, src)
		if applyDeprecations(body, rule, "test.tf") {
			t.Error("must not change a flag that already declares variations")
		}
	})

	t.Run("warns and skips for non-literal variation_type", func(t *testing.T) {
		src := `resource "launchdarkly_feature_flag" "f" {
  variation_type = var.kind
}
`
		_, body := parseBody(t, src)
		before := warningCount
		if applyDeprecations(body, rule, "test.tf") {
			t.Error("must not synthesize when variation_type is not a literal boolean")
		}
		if warningCount != before+1 {
			t.Errorf("warningCount delta = %d, want 1", warningCount-before)
		}
	})
}

func TestApplyDSAttrRewrites(t *testing.T) {
	// default_client_side_availability is a single object in v3, so the rename also
	// strips the v2 list index ([0]) that followed the old list-shaped attribute.
	src := []byte(`output "csa" {
  value = data.launchdarkly_project.p.client_side_availability[0].using_environment_id
}
`)
	spec := Spec{"launchdarkly_project": {DSAttrRewrites: []*DSAttrRewrite{{From: "client_side_availability", To: "default_client_side_availability", StripIndex: true}}}}
	out, changed := applyDSAttrRewrites(src, spec)
	if !changed {
		t.Fatal("expected rewrite")
	}
	if !strings.Contains(string(out), "data.launchdarkly_project.p.default_client_side_availability.using_environment_id") {
		t.Errorf("reference not renamed / index not stripped:\n%s", out)
	}
	if strings.Contains(string(out), "default_client_side_availability[0]") {
		t.Errorf("list index must be stripped for the single-object attribute:\n%s", out)
	}
}

// TestEmbeddedMappingsStripDataSourceIndices guards the shipped mappings.json:
// every v3 single-object data-source attribute must have a ds_attr_rewrite that
// drops a v2 list index, so v2 readers like `data.X.Y.attr[0].field` migrate to
// the v3 object access `data.X.Y.attr.field` (project additionally renames
// client_side_availability -> default_client_side_availability).
func TestEmbeddedMappingsStripDataSourceIndices(t *testing.T) {
	var spec Spec
	if err := json.Unmarshal(defaultMappings, &spec); err != nil {
		t.Fatalf("parse embedded mappings: %v", err)
	}
	src := []byte(`
output "a" { value = data.launchdarkly_feature_flag.f.client_side_availability[0].using_environment_id }
output "b" { value = data.launchdarkly_feature_flag.f.defaults[0].on_variation }
output "c" { value = data.launchdarkly_project.p.client_side_availability[0].using_mobile_key }
output "d" { value = data.launchdarkly_feature_flag_environment.e.fallthrough[0].variation }
`)
	out, changed := applyDSAttrRewrites(src, spec)
	if !changed {
		t.Fatal("expected rewrites")
	}
	s := string(out)
	for _, want := range []string{
		"data.launchdarkly_feature_flag.f.client_side_availability.using_environment_id",
		"data.launchdarkly_feature_flag.f.defaults.on_variation",
		"data.launchdarkly_project.p.default_client_side_availability.using_mobile_key",
		"data.launchdarkly_feature_flag_environment.e.fallthrough.variation",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
	if strings.Contains(s, "[0]") {
		t.Errorf("residual v2 list index left in data-source reference:\n%s", s)
	}
}

func TestApplyDSAttrRewritesStripIndexToExpr(t *testing.T) {
	// feature_flag include_in_snippet → client_side_availability.using_environment_id.
	// A v2 reader that indexed the list ([0]) must collapse to the v3 object access.
	src := []byte(`output "iis" {
  value = data.launchdarkly_feature_flag.f.include_in_snippet
}
output "iis_indexed" {
  value = data.launchdarkly_feature_flag.f.client_side_availability[0].using_mobile_key
}
`)
	spec := Spec{"launchdarkly_feature_flag": {DSAttrRewrites: []*DSAttrRewrite{
		{From: "include_in_snippet", ToExpr: "client_side_availability.using_environment_id"},
		{From: "client_side_availability", ToExpr: "client_side_availability", StripIndex: true},
	}}}
	out, changed := applyDSAttrRewrites(src, spec)
	if !changed {
		t.Fatal("expected rewrite")
	}
	s := string(out)
	if !strings.Contains(s, "data.launchdarkly_feature_flag.f.client_side_availability.using_environment_id") {
		t.Errorf("include_in_snippet not rewritten to object access:\n%s", s)
	}
	if !strings.Contains(s, "data.launchdarkly_feature_flag.f.client_side_availability.using_mobile_key") {
		t.Errorf("list index not stripped:\n%s", s)
	}
}

func TestForwardConvertsPlainMapBlock(t *testing.T) {
	// role_attributes: {key, values} blocks collapse to a plain map of lists.
	src := `resource "launchdarkly_team" "t" {
  key  = "eng"
  name = "Engineering"
  role_attributes {
    key    = "testAttribute"
    values = ["staging", "production"]
  }
  role_attributes {
    key    = "otherAttribute"
    values = ["someValue"]
  }
}
`
	f, body := parseBody(t, src)
	spec := []*AttrSpec{{Name: "role_attributes", MapKey: "key", MapValue: "values"}}
	if !forward(body, spec, "test.tf") {
		t.Fatal("expected conversion")
	}
	out := string(hclwrite.Format(f.Bytes()))
	for _, want := range []string{
		"role_attributes = {",
		`"testAttribute"  = ["staging", "production"]`,
		`"otherAttribute" = ["someValue"]`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "values") {
		t.Errorf("plain-map form must not retain the values attribute name:\n%s", out)
	}
	if _, diag := hclwrite.ParseConfig([]byte(out), "out.tf", hcl.Pos{Line: 1, Column: 1}); diag.HasErrors() {
		t.Errorf("converted output does not parse: %s", diag)
	}
}

func TestReversePlainMapBlock(t *testing.T) {
	src := `resource "launchdarkly_team" "t" {
  key  = "eng"
  name = "Engineering"
  role_attributes = {
    "testAttribute" = ["staging", "production"]
    otherAttribute  = ["someValue"]
  }
}
`
	f, body := parseBody(t, src)
	spec := []*AttrSpec{{Name: "role_attributes", MapKey: "key", MapValue: "values"}}
	if !reverse(body, spec) {
		t.Fatal("expected reverse conversion")
	}
	out := string(hclwrite.Format(f.Bytes()))
	if n := strings.Count(out, "role_attributes {"); n != 2 {
		t.Errorf("expected 2 role_attributes blocks, got %d:\n%s", n, out)
	}
	for _, want := range []string{
		`key    = "testAttribute"`,
		`values = ["staging", "production"]`,
		`values = ["someValue"]`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "role_attributes = {") {
		t.Errorf("reverse must drop the map-assignment form:\n%s", out)
	}
	if _, diag := hclwrite.ParseConfig([]byte(out), "out.tf", hcl.Pos{Line: 1, Column: 1}); diag.HasErrors() {
		t.Errorf("reversed output does not parse: %s", diag)
	}
}

func TestForwardConvertsCustomPropertiesMap(t *testing.T) {
	src := `resource "launchdarkly_feature_flag" "f" {
  custom_properties {
    key   = "some.property"
    name  = "Some Property"
    value = ["a", "b"]
  }
  custom_properties {
    key   = "other.property"
    name  = "Other Property"
    value = ["c"]
  }
}
`
	f, body := parseBody(t, src)
	spec := []*AttrSpec{{Name: "custom_properties", MapKey: "key"}}
	if !forward(body, spec, "test.tf") {
		t.Fatal("expected conversion")
	}
	out := string(hclwrite.Format(f.Bytes()))
	for _, want := range []string{
		"custom_properties = {",
		`"some.property" = {`,
		`"other.property" = {`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "custom_properties = [{") {
		t.Errorf("map attribute must not be a list:\n%s", out)
	}
	if _, diag := hclwrite.ParseConfig([]byte(out), "out.tf", hcl.Pos{Line: 1, Column: 1}); diag.HasErrors() {
		t.Errorf("converted output does not parse: %s", diag)
	}
}

func TestForwardConvertsInstructionsObject(t *testing.T) {
	src := `resource "launchdarkly_flag_trigger" "t" {
  enabled = true
  instructions {
    kind = "turnFlagOn"
  }
}
`
	f, body := parseBody(t, src)
	if !forward(body, []*AttrSpec{{Name: "instructions", Object: true}}, "test.tf") {
		t.Fatal("expected conversion")
	}
	out := string(hclwrite.Format(f.Bytes()))
	if !strings.Contains(out, "instructions = {") {
		t.Errorf("expected single-object syntax, got:\n%s", out)
	}
	if strings.Contains(out, "instructions = [{") {
		t.Errorf("object attribute must not be wrapped in a list:\n%s", out)
	}
}

package main

import (
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
	if !applyDeprecations(body, []*DeprecationSpec{{Name: "policy_statements", Action: "rename", To: "inline_roles"}}) {
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
	if !applyDeprecations(body, []*DeprecationSpec{{Name: "include_in_snippet", Action: "iis_to_csa", To: "client_side_availability"}}) {
		t.Fatal("expected conversion")
	}
	out := string(hclwrite.Format(f.Bytes()))
	for _, want := range []*regexp.Regexp{
		regexp.MustCompile(`client_side_availability = \[\{`),
		regexp.MustCompile(`using_environment_id\s+= var\.snippet`),
		regexp.MustCompile(`using_mobile_key\s+= false`),
	} {
		if !want.MatchString(out) {
			t.Errorf("missing %v in:\n%s", want, out)
		}
	}
}

func TestApplyDSAttrRewrites(t *testing.T) {
	src := []byte(`output "csa" {
  value = data.launchdarkly_project.p.client_side_availability[0].using_environment_id
}
`)
	spec := Spec{"launchdarkly_project": {DSAttrRewrites: []*DSAttrRewrite{{From: "client_side_availability", To: "default_client_side_availability"}}}}
	out, changed := applyDSAttrRewrites(src, spec)
	if !changed {
		t.Fatal("expected rewrite")
	}
	if !strings.Contains(string(out), "data.launchdarkly_project.p.default_client_side_availability[0].using_environment_id") {
		t.Errorf("reference not renamed:\n%s", out)
	}
}

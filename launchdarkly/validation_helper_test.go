package launchdarkly

import (
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

func viewKeyTestPath() cty.Path {
	return cty.Path{cty.GetAttrStep{Name: KEY}}
}

func TestValidateViewKey(t *testing.T) {
	v := validateViewKey()
	good := []string{"a", "abc", "my-view", "my_view", "0abc", "my.dotted.view", "view-123"}
	for _, s := range good {
		if diags := v(s, viewKeyTestPath()); diags.HasError() {
			t.Fatalf("expected %q to pass, got %v", s, diags)
		}
	}
	// Uppercase is the case this validator exists for: it satisfies the key
	// regex but cannot round-trip through LD's lowercase normalization.
	uppercase := []string{"MyView", "ABC", "my-View", "Z.dot.path"}
	for _, s := range uppercase {
		diags := v(s, viewKeyTestPath())
		if !diags.HasError() {
			t.Fatalf("expected uppercase key %q to fail view key validation", s)
		}
		if got := diags[0].Detail; !strings.Contains(got, strings.ToLower(s)) {
			t.Fatalf("expected diagnostic for %q to suggest the lowercased key, got %q", s, got)
		}
	}
	// Key-pattern failures must still surface through the composite.
	bad := []string{"-leading-dash", ".dot-first", "has space", "has/slash", ""}
	for _, s := range bad {
		if diags := v(s, viewKeyTestPath()); !diags.HasError() {
			t.Fatalf("expected %q to fail view key validation", s)
		}
	}
}

func TestValidateViewKey_NonString(t *testing.T) {
	diags := validateViewKey()(42, viewKeyTestPath())
	if !diags.HasError() {
		t.Fatal("expected a non-string value to fail")
	}
}

func TestValidateLowercase(t *testing.T) {
	v := validateLowercase("because reasons")
	// Non-letter characters have no case and must not trip the check.
	for _, s := range []string{"", "abc", "a-b_c.1", "123"} {
		if diags := v(s, viewKeyTestPath()); diags.HasError() {
			t.Fatalf("expected %q to pass, got %v", s, diags)
		}
	}
	for _, s := range []string{"A", "aB", "ABC"} {
		diags := v(s, viewKeyTestPath())
		if !diags.HasError() {
			t.Fatalf("expected %q to fail lowercase validation", s)
		}
		if got := diags[0].Detail; !strings.Contains(got, "because reasons") {
			t.Fatalf("expected diagnostic for %q to carry the reason, got %q", s, got)
		}
	}
}

func TestValidateLowercase_EmptyReason(t *testing.T) {
	diags := validateLowercase("")("Abc", viewKeyTestPath())
	if !diags.HasError() {
		t.Fatal("expected uppercase value to fail")
	}
	got := diags[0].Detail
	// An empty reason must not leave a doubled or dangling separator.
	if strings.Contains(got, "..") {
		t.Fatalf("empty reason produced malformed detail: %q", got)
	}
	if !strings.Contains(got, `Use "abc" instead`) {
		t.Fatalf("expected normalized suggestion, got %q", got)
	}
}

func TestValidateLowercase_AttributePathPreserved(t *testing.T) {
	path := viewKeyTestPath()
	diags := validateLowercase("")("Abc", path)
	if !diags.HasError() {
		t.Fatal("expected uppercase value to fail")
	}
	// Terraform renders the offending attribute from the diagnostic's path, so
	// losing it would leave the user without a location for the error.
	if len(diags[0].AttributePath) != len(path) {
		t.Fatalf("expected the attribute path to be carried through, got %v", diags[0].AttributePath)
	}
}

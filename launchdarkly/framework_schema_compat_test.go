package launchdarkly

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

func TestIsOmittedFrameworkAttrDiag_PositiveMatch(t *testing.T) {
	attrPath := path.Root("deprecated_attr")
	d := diag.NewAttributeErrorDiagnostic(
		attrPath,
		"State Write Error",
		"An unexpected error was encountered trying to retrieve type information at a given path. The provider has likely been misconfigured.",
	)

	if !isOmittedFrameworkAttrDiag(d, attrPath) {
		t.Fatalf("expected canonical missing-attr diagnostic to be recognised")
	}
}

func TestIsOmittedFrameworkAttrDiag_WrongPath(t *testing.T) {
	d := diag.NewAttributeErrorDiagnostic(
		path.Root("some_other_attr"),
		"State Write Error",
		"An unexpected error was encountered trying to retrieve type information at a given path.",
	)
	if isOmittedFrameworkAttrDiag(d, path.Root("deprecated_attr")) {
		t.Fatalf("matcher must scope on attribute path; mismatched path should not match")
	}
}

func TestIsOmittedFrameworkAttrDiag_WrongSummary(t *testing.T) {
	attrPath := path.Root("deprecated_attr")
	d := diag.NewAttributeErrorDiagnostic(
		attrPath,
		"Some Other Error",
		"An unexpected error was encountered trying to retrieve type information at a given path.",
	)
	if isOmittedFrameworkAttrDiag(d, attrPath) {
		t.Fatalf("matcher must require Write Error summary suffix")
	}
}

func TestIsOmittedFrameworkAttrDiag_WrongDetail(t *testing.T) {
	attrPath := path.Root("deprecated_attr")
	d := diag.NewAttributeErrorDiagnostic(
		attrPath,
		"State Write Error",
		"Some completely unrelated detail about a validation failure.",
	)
	if isOmittedFrameworkAttrDiag(d, attrPath) {
		t.Fatalf("matcher must require the canonical detail prefix")
	}
}

func TestFilterOmittedAttrDiags_PassThroughUnrelated(t *testing.T) {
	attrPath := path.Root("deprecated_attr")

	matching := diag.NewAttributeErrorDiagnostic(
		attrPath,
		"State Write Error",
		"An unexpected error was encountered trying to retrieve type information at a given path.",
	)
	unrelated := diag.NewErrorDiagnostic("Other Error", "something else broke")

	diags := diag.Diagnostics{matching, unrelated}
	filtered := filterOmittedAttrDiags(diags, attrPath)

	if len(filtered) != 1 {
		t.Fatalf("expected 1 surviving diagnostic, got %d", len(filtered))
	}
	if filtered[0].Summary() != "Other Error" {
		t.Fatalf("unrelated diagnostic should survive; got %q", filtered[0].Summary())
	}
}

func TestIsOmittedFrameworkAttrDiag_NonAttributePathDoesNotMatch(t *testing.T) {
	// A plain error diagnostic (no path) must never match — the matcher
	// is scoped to a specific attribute.
	d := diag.NewErrorDiagnostic(
		"State Write Error",
		"An unexpected error was encountered trying to retrieve type information at a given path.",
	)
	if isOmittedFrameworkAttrDiag(d, path.Root("deprecated_attr")) {
		t.Fatalf("plain error diagnostics must not match path-scoped matcher")
	}
}

package launchdarkly

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestStringSliceFromSet_RoundTrip(t *testing.T) {
	ctx := context.Background()
	original := []string{"alpha", "beta", "gamma"}

	set, diags := setFromStringSlice(ctx, original)
	if diags.HasError() {
		t.Fatalf("setFromStringSlice diags: %v", diags)
	}

	got, diags := stringSliceFromSet(ctx, set)
	if diags.HasError() {
		t.Fatalf("stringSliceFromSet diags: %v", diags)
	}

	if len(got) != len(original) {
		t.Fatalf("round-trip lost elements: got %v, want %v", got, original)
	}
	seen := map[string]bool{}
	for _, v := range got {
		seen[v] = true
	}
	for _, v := range original {
		if !seen[v] {
			t.Fatalf("round-trip missing %q (got %v)", v, got)
		}
	}
}

func TestStringSliceFromSet_NullAndUnknown(t *testing.T) {
	ctx := context.Background()

	null := types.SetNull(types.StringType)
	got, diags := stringSliceFromSet(ctx, null)
	if diags.HasError() {
		t.Fatalf("null set diags: %v", diags)
	}
	if len(got) != 0 {
		t.Fatalf("null set should yield empty slice, got %v", got)
	}

	unknown := types.SetUnknown(types.StringType)
	got, diags = stringSliceFromSet(ctx, unknown)
	if diags.HasError() {
		t.Fatalf("unknown set diags: %v", diags)
	}
	if len(got) != 0 {
		t.Fatalf("unknown set should yield empty slice, got %v", got)
	}
}

// TestSetFromStringSlice_OrderInvariant guards the Phase 0.9a parity
// guarantee for set-hashed Sets (custom_roles on team_member /
// team_members / team data sources, tags on feature_flag) — reordered
// input must produce a SetValue that the framework treats as equal.
func TestSetFromStringSlice_OrderInvariant(t *testing.T) {
	ctx := context.Background()

	a, diags := setFromStringSlice(ctx, []string{"alpha", "beta", "gamma"})
	if diags.HasError() {
		t.Fatalf("first setFromStringSlice diags: %v", diags)
	}
	b, diags := setFromStringSlice(ctx, []string{"gamma", "alpha", "beta"})
	if diags.HasError() {
		t.Fatalf("second setFromStringSlice diags: %v", diags)
	}
	if !a.Equal(b) {
		t.Fatalf("reordered slices produced unequal sets: %v vs %v", a, b)
	}
}

func TestSetFromStringSlice_NilInput(t *testing.T) {
	ctx := context.Background()

	set, diags := setFromStringSlice(ctx, nil)
	if diags.HasError() {
		t.Fatalf("nil input diags: %v", diags)
	}
	if set.IsNull() {
		t.Fatalf("nil input should produce a non-null empty set, got null")
	}
	if len(set.Elements()) != 0 {
		t.Fatalf("nil input should produce empty set, got %d elements", len(set.Elements()))
	}
}

func TestStringSliceFromList_RoundTrip(t *testing.T) {
	ctx := context.Background()
	original := []string{"first", "second", "third"}

	list, diags := listFromStringSlice(ctx, original)
	if diags.HasError() {
		t.Fatalf("listFromStringSlice diags: %v", diags)
	}

	got, diags := stringSliceFromList(ctx, list)
	if diags.HasError() {
		t.Fatalf("stringSliceFromList diags: %v", diags)
	}

	if len(got) != len(original) {
		t.Fatalf("round-trip lost elements: got %v, want %v", got, original)
	}
	for i, v := range original {
		if got[i] != v {
			t.Fatalf("list ordering not preserved at index %d: got %q, want %q", i, got[i], v)
		}
	}
}

func TestStringValueOrNull(t *testing.T) {
	if !stringValueOrNull("").IsNull() {
		t.Fatalf("empty string should produce null")
	}
	v := stringValueOrNull("hello")
	if v.IsNull() {
		t.Fatalf("non-empty string should not be null")
	}
	if v.ValueString() != "hello" {
		t.Fatalf("got %q, want %q", v.ValueString(), "hello")
	}
}

func TestStringPointerFromAttr(t *testing.T) {
	if stringPointerFromAttr(types.StringNull()) != nil {
		t.Fatalf("null should produce nil pointer")
	}
	if stringPointerFromAttr(types.StringUnknown()) != nil {
		t.Fatalf("unknown should produce nil pointer")
	}
	p := stringPointerFromAttr(types.StringValue("ok"))
	if p == nil || *p != "ok" {
		t.Fatalf("known value lost: got %v", p)
	}
}

func TestConfigureResourceClient_NilProviderData(t *testing.T) {
	req := resource.ConfigureRequest{ProviderData: nil}
	resp := &resource.ConfigureResponse{}
	if got := configureResourceClient(req, resp); got != nil {
		t.Fatalf("expected nil client on nil ProviderData, got %v", got)
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("nil ProviderData should not produce diagnostics, got %v", resp.Diagnostics)
	}
}

func TestConfigureResourceClient_WrongType(t *testing.T) {
	req := resource.ConfigureRequest{ProviderData: "not a client"}
	resp := &resource.ConfigureResponse{}
	if got := configureResourceClient(req, resp); got != nil {
		t.Fatalf("expected nil client on type mismatch, got %v", got)
	}
	if !resp.Diagnostics.HasError() {
		t.Fatalf("type mismatch should produce an error diagnostic")
	}
}

func TestConfigureResourceClient_Success(t *testing.T) {
	want := &Client{}
	req := resource.ConfigureRequest{ProviderData: want}
	resp := &resource.ConfigureResponse{}
	got := configureResourceClient(req, resp)
	if got != want {
		t.Fatalf("expected configured client to flow through, got %v", got)
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("success path should not produce diagnostics, got %v", resp.Diagnostics)
	}
}

func TestConfigureDataSourceClient_Surfaces(t *testing.T) {
	want := &Client{}
	req := datasource.ConfigureRequest{ProviderData: want}
	resp := &datasource.ConfigureResponse{}
	if got := configureDataSourceClient(req, resp); got != want {
		t.Fatalf("expected configured client to flow through, got %v", got)
	}
}

func TestAddLdapiError(t *testing.T) {
	var diags diag.Diagnostics
	sink := diagSinkAdapter{diags: &diags}

	addLdapiError(sink, "Operation failed", nil)
	if diags.HasError() {
		t.Fatalf("nil error should not produce diagnostics")
	}

	addLdapiError(sink, "Operation failed", errors.New("boom"))
	if !diags.HasError() {
		t.Fatalf("non-nil error should produce a diagnostic")
	}
}

// diagSinkAdapter lets us drive addLdapiError against a standalone
// diag.Diagnostics without constructing a full *Response value.
type diagSinkAdapter struct {
	diags *diag.Diagnostics
}

func (s diagSinkAdapter) AddError(summary, detail string) {
	s.diags.AddError(summary, detail)
}

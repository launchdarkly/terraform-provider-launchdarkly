package launchdarkly

// framework_helpers.go centralises the small utilities that every
// terraform-plugin-framework resource and data source in this provider needs:
// extracting the configured *Client, converting between framework
// types.Set / types.List and Go []string, surfacing LD API errors as
// framework diagnostics. Keep the surface narrow — anything that needs
// non-trivial logic belongs next to the resource that consumes it.

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// diagnosticsSink is the narrow interface satisfied by every framework
// *Response type (CreateResponse, ReadResponse, etc.) that carries a
// Diagnostics field. We accept the interface rather than a concrete type so
// helpers work uniformly across resource and data source surfaces.
type diagnosticsSink interface {
	AddError(summary, detail string)
}

// configureResourceClient pulls the configured *Client out of a
// resource.ConfigureRequest. Returns nil when the provider has not yet
// supplied ResourceData (which Terraform Framework signals on the first
// Configure pass) so callers can early-return without an error.
func configureResourceClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *Client {
	if req.ProviderData == nil {
		return nil
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *launchdarkly.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return nil
	}
	return client
}

// configureDataSourceClient is the data source analogue of
// configureResourceClient. The framework distinguishes the two via different
// request types so we expose a matching pair rather than a generic.
func configureDataSourceClient(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *Client {
	if req.ProviderData == nil {
		return nil
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *launchdarkly.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return nil
	}
	return client
}

// addLdapiError appends an LD API error to a diagnostics sink, routing the
// raw error through handleLdapiErr so the response body of a
// GenericOpenAPIError surfaces to the user. The summary follows the framework
// convention of a short headline; the detail is the unwrapped error string.
func addLdapiError(diags diagnosticsSink, summary string, err error) {
	if err == nil {
		return
	}
	diags.AddError(summary, handleLdapiErr(err).Error())
}

// stringSliceFromSet converts a framework types.Set whose elements are
// types.String into a plain []string. Null / unknown sets return an empty
// slice rather than nil so callers can use len() uniformly.
func stringSliceFromSet(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return []string{}, nil
	}
	out := make([]string, 0, len(set.Elements()))
	diags := set.ElementsAs(ctx, &out, false)
	return out, diags
}

// setFromStringSlice builds a types.Set of types.String from a Go slice.
// A nil input produces a non-null empty set so state writes don't flip
// between null and empty across plans.
func setFromStringSlice(ctx context.Context, vals []string) (types.Set, diag.Diagnostics) {
	if vals == nil {
		vals = []string{}
	}
	return types.SetValueFrom(ctx, types.StringType, vals)
}

// stringSliceFromList is stringSliceFromSet for ordered lists.
func stringSliceFromList(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return []string{}, nil
	}
	out := make([]string, 0, len(list.Elements()))
	diags := list.ElementsAs(ctx, &out, false)
	return out, diags
}

// listFromStringSlice is setFromStringSlice for ordered lists.
func listFromStringSlice(ctx context.Context, vals []string) (types.List, diag.Diagnostics) {
	if vals == nil {
		vals = []string{}
	}
	return types.ListValueFrom(ctx, types.StringType, vals)
}

// stringValueOrNull returns a non-null types.String when v != "", else
// types.StringNull(). Use when mapping API responses that distinguish
// "absent" from "empty string" into framework state.
func stringValueOrNull(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}

// stringPointerFromAttr is the inverse: null / unknown framework values
// project to a nil *string, suitable for ldapi optional-field patches.
func stringPointerFromAttr(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

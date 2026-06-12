package launchdarkly

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// Metric group kinds. A funnel metric group orders its metrics into a
// conversion funnel and requires a name_in_group for each metric; a standard
// metric group is an unordered collection.
const (
	METRIC_GROUP_KIND_FUNNEL   = "funnel"
	METRIC_GROUP_KIND_STANDARD = "standard"
)

// metricGroupMetricAttrTypes is the object type of a single entry in the
// metric group's `metrics` list.
var metricGroupMetricAttrTypes = map[string]attr.Type{
	KEY:           types.StringType,
	NAME_IN_GROUP: types.StringType,
}

// metricGroupMetricModel mirrors one element of the `metrics` nested attribute.
type metricGroupMetricModel struct {
	Key         types.String `tfsdk:"key"`
	NameInGroup types.String `tfsdk:"name_in_group"`
}

// newMetricGroupBetaClient returns a beta-configured client suitable for the
// metric-groups endpoints. Unlike ViewsBetaApi, the generated MetricsBetaApi
// request builders do not expose a per-request .LDAPIVersion("beta") setter, so
// we set the LD-API-Version header as a client default instead. The header is
// read from the configuration at request-build time, so mutating it here takes
// effect for every metric-group call made through the returned client.
func newMetricGroupBetaClient(c *Client) (*Client, error) {
	beta, err := newBetaClient(c.apiKey, c.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return nil, err
	}
	beta.ld.GetConfig().AddDefaultHeader("LD-API-Version", "beta")
	beta.ld404Retry.GetConfig().AddDefaultHeader("LD-API-Version", "beta")
	return beta, nil
}

// metricGroupMaintainerID extracts the maintainer's member ID from the API
// representation. The beta API returns maintainer as {key, kind, _member}, a
// shape the v22 MaintainerRep model ({member, team}) predates — the member ID
// arrives in AdditionalProperties["key"]. The typed read stays first so a
// future client regen takes over transparently.
func metricGroupMaintainerID(m ldapi.MaintainerRep) string {
	if m.Member != nil && m.Member.GetId() != "" {
		return m.Member.GetId()
	}
	if k, ok := m.AdditionalProperties["key"].(string); ok {
		return k
	}
	return ""
}

// metricGroupIdToKeys splits a composite metric group ID into its project key
// and metric group key. The expected format is `project_key/metric_group_key`.
func metricGroupIdToKeys(id string) (projectKey string, metricGroupKey string, err error) {
	if strings.Count(id, "/") != 1 {
		return "", "", fmt.Errorf("found unexpected metric group id format: %q expected format: 'project_key/metric_group_key'", id)
	}
	parts := strings.SplitN(id, "/", 2)
	return parts[0], parts[1], nil
}

// metricGroupInputsFromModels converts the Terraform `metrics` list into the
// generated API input type, preserving order (the funnel order is significant).
func metricGroupInputsFromModels(in []metricGroupMetricModel) []ldapi.MetricInMetricGroupInput {
	out := make([]ldapi.MetricInMetricGroupInput, len(in))
	for i, m := range in {
		out[i] = ldapi.MetricInMetricGroupInput{
			Key:         m.Key.ValueString(),
			NameInGroup: m.NameInGroup.ValueString(),
		}
	}
	return out
}

// metricGroupMetricsToList converts the API representation of a metric group's
// metrics into a Terraform list value, preserving API order.
func metricGroupMetricsToList(metrics []ldapi.MetricInGroupRep) (types.List, error) {
	objType := types.ObjectType{AttrTypes: metricGroupMetricAttrTypes}
	elems := make([]attr.Value, 0, len(metrics))
	for _, m := range metrics {
		nameInGroup := types.StringNull()
		if m.NameInGroup != nil && *m.NameInGroup != "" {
			nameInGroup = types.StringValue(*m.NameInGroup)
		}
		obj, diags := types.ObjectValue(metricGroupMetricAttrTypes, map[string]attr.Value{
			KEY:           types.StringValue(m.Key),
			NAME_IN_GROUP: nameInGroup,
		})
		if diags.HasError() {
			return types.ListNull(objType), fmt.Errorf("failed to build metric group metric object for key %q", m.Key)
		}
		elems = append(elems, obj)
	}
	list, diags := types.ListValue(objType, elems)
	if diags.HasError() {
		return types.ListNull(objType), fmt.Errorf("failed to build metric group metrics list")
	}
	return list, nil
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const fixtureSpec = `{
  "paths": {
    "/api/v2/projects": {
      "get": {"tags": ["Projects"], "operationId": "getProjects"},
      "post": {"tags": ["Projects"], "operationId": "postProject"},
      "parameters": [{"name": "ignored"}]
    },
    "/api/v2/projects/{projectKey}": {
      "get": {"tags": ["Projects"], "operationId": "getProject"},
      "delete": {"tags": ["Projects"], "operationId": "deleteProject"}
    },
    "/api/v2/shiny/{key}": {
      "get": {"tags": ["Shiny new feature"], "operationId": "getShiny"},
      "post": {"tags": ["Shiny new feature"], "operationId": "postShiny"}
    },
    "/api/v2/widgets": {
      "get": {"tags": ["Widgets"], "operationId": "getWidgets"},
      "post": {"tags": ["Widgets"], "operationId": "postWidget"}
    },
    "/api/v2/widgets/{key}": {
      "delete": {"tags": ["Widgets"], "operationId": "deleteWidget"}
    },
    "/api/v2/untagged-thing": {
      "get": {}
    }
  }
}`

func fixtureMapping(t *testing.T, families ...MappingFamily) *Mapping {
	t.Helper()
	m := &Mapping{Version: 1, Families: families}
	if err := m.validate(); err != nil {
		t.Fatalf("fixture mapping invalid: %v", err)
	}
	return m
}

func fixtureOperations(t *testing.T) map[string][]SpecOperation {
	t.Helper()
	families, err := specOperations([]byte(fixtureSpec))
	if err != nil {
		t.Fatal(err)
	}
	return families
}

func TestSpecOperations(t *testing.T) {
	families := fixtureOperations(t)
	if got := len(families["Projects"]); got != 4 {
		t.Errorf("Projects operations = %d, want 4", got)
	}
	if got := len(uniquePaths(families["Projects"])); got != 2 {
		t.Errorf("Projects paths = %d, want 2", got)
	}
	if got := len(families["<untagged>"]); got != 1 {
		t.Errorf("<untagged> operations = %d, want 1", got)
	}
	if _, ok := families["parameters"]; ok {
		t.Error("non-method path-item keys must not become families")
	}
	first := families["Projects"][0]
	if first.Method != "GET" || first.Path != "/api/v2/projects" || first.OperationID != "getProjects" {
		t.Errorf("first Projects op = %+v, want GET /api/v2/projects getProjects", first)
	}
	// Untagged fixture op has no operationId; the claim key falls back to method+path.
	if got := families["<untagged>"][0].key(); got != "GET /api/v2/untagged-thing" {
		t.Errorf("untagged op key = %q, want method+path fallback", got)
	}
}

func TestSpecOperationsEmptySpec(t *testing.T) {
	if _, err := specOperations([]byte(`{"paths": {}}`)); err == nil {
		t.Error("empty spec should error, not report full coverage")
	}
}

func TestBuildReportDetectsDrift(t *testing.T) {
	families := fixtureOperations(t)
	mapping := fixtureMapping(t,
		MappingFamily{Tag: "Projects", Status: statusCovered, Resources: []ResourceEntry{{Name: "launchdarkly_project"}}},
		MappingFamily{Tag: "Widgets", Status: statusIgnored, Reason: "fixture filler"},
		MappingFamily{Tag: "<untagged>", Status: statusIgnored, Reason: "spec hygiene bucket"},
		MappingFamily{Tag: "Renamed family", Status: statusIgnored, Reason: "was removed from spec"},
	)
	report := buildReport(families, mapping,
		[]string{"launchdarkly_project", "launchdarkly_orphan"},
		[]string{"launchdarkly_project"}, // data source sharing a family entry
		"fixture")

	if !report.HasDrift() {
		t.Fatal("expected drift")
	}
	if len(report.NewFamilies) != 1 || report.NewFamilies[0].Tag != "Shiny new feature" {
		t.Errorf("NewFamilies = %+v, want exactly [Shiny new feature]", report.NewFamilies)
	}
	if len(report.StaleFamilies) != 1 || report.StaleFamilies[0] != "Renamed family" {
		t.Errorf("StaleFamilies = %v, want [Renamed family]", report.StaleFamilies)
	}
	if len(report.UnmappedResources) != 1 || report.UnmappedResources[0] != "launchdarkly_orphan" {
		t.Errorf("UnmappedResources = %v, want [launchdarkly_orphan]", report.UnmappedResources)
	}
}

func TestBuildReportClean(t *testing.T) {
	families := fixtureOperations(t)
	mapping := fixtureMapping(t,
		MappingFamily{Tag: "Projects", Status: statusCovered, Resources: []ResourceEntry{{Name: "launchdarkly_project"}}},
		MappingFamily{Tag: "Shiny new feature", Status: statusTriage},
		MappingFamily{Tag: "Widgets", Status: statusIgnored, Reason: "fixture filler"},
		MappingFamily{Tag: "<untagged>", Status: statusIgnored, Reason: "spec hygiene bucket"},
	)
	report := buildReport(families, mapping, []string{"launchdarkly_project"}, nil, "fixture")
	if report.HasDrift() {
		t.Fatalf("expected clean report, got %+v", report)
	}
	if len(report.TriageFamilies) != 1 {
		t.Errorf("TriageFamilies = %v, want 1 entry", report.TriageFamilies)
	}

	var buf bytes.Buffer
	if err := renderMarkdown(&buf, report); err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	if !strings.Contains(buf.String(), "No drift detected") {
		t.Error("markdown should state no drift")
	}

	// Empty drift lists must serialize as [] (not null) for jq consumers.
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"new_families", "stale_families", "unmapped_resources", "unclaimed_operations", "stale_operations", "triage_operations"} {
		if strings.Contains(string(raw), fmt.Sprintf("%q:null", key)) {
			t.Errorf("%s serializes as null, want []", key)
		}
	}
}

// partialWidgets returns a partial-family entry claiming the given operations
// on a fixture resource, with optional ignored ops.
func partialWidgets(claims []string, ignored ...IgnoredOperation) MappingFamily {
	return MappingFamily{
		Tag:               "Widgets",
		Status:            statusPartial,
		Resources:         []ResourceEntry{{Name: "launchdarkly_widget", Operations: claims}},
		IgnoredOperations: ignored,
	}
}

func TestBuildReportUnclaimedOperations(t *testing.T) {
	families := fixtureOperations(t)
	mapping := fixtureMapping(t,
		MappingFamily{Tag: "Projects", Status: statusCovered, Resources: []ResourceEntry{{Name: "launchdarkly_project"}}},
		MappingFamily{Tag: "Shiny new feature", Status: statusTriage},
		MappingFamily{Tag: "<untagged>", Status: statusIgnored, Reason: "spec hygiene bucket"},
		partialWidgets([]string{"getWidgets", "postWidget"}),
	)
	report := buildReport(families, mapping, []string{"launchdarkly_project", "launchdarkly_widget"}, nil, "fixture")

	if !report.HasDrift() {
		t.Fatal("expected drift from unclaimed deleteWidget")
	}
	if len(report.UnclaimedOperations) != 1 {
		t.Fatalf("UnclaimedOperations = %+v, want exactly 1", report.UnclaimedOperations)
	}
	got := report.UnclaimedOperations[0]
	want := OperationDetail{Tag: "Widgets", OperationID: "deleteWidget", Method: "DELETE", Path: "/api/v2/widgets/{key}"}
	if got != want {
		t.Errorf("UnclaimedOperations[0] = %+v, want %+v", got, want)
	}

	var buf bytes.Buffer
	if err := renderMarkdown(&buf, report); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Unclaimed operations in partial families") {
		t.Error("markdown should include the unclaimed-operations section")
	}
}

func TestBuildReportTriageOperations(t *testing.T) {
	families := fixtureOperations(t)
	widgets := partialWidgets([]string{"getWidgets", "postWidget"})
	widgets.TriageOperations = []string{"deleteWidget"}
	mapping := fixtureMapping(t,
		MappingFamily{Tag: "Projects", Status: statusCovered, Resources: []ResourceEntry{{Name: "launchdarkly_project"}}},
		MappingFamily{Tag: "Shiny new feature", Status: statusTriage},
		MappingFamily{Tag: "<untagged>", Status: statusIgnored, Reason: "spec hygiene bucket"},
		widgets,
	)
	report := buildReport(families, mapping, []string{"launchdarkly_project", "launchdarkly_widget"}, nil, "fixture")

	// Triage ops are informational: listed, but never drift.
	if report.HasDrift() {
		t.Fatalf("triage operation must not drift, got %+v", report)
	}
	if len(report.TriageOperations) != 1 || report.TriageOperations[0].OperationID != "deleteWidget" {
		t.Errorf("TriageOperations = %+v, want exactly [deleteWidget]", report.TriageOperations)
	}
	if report.TriageOperations[0].Method != "DELETE" || report.TriageOperations[0].Path != "/api/v2/widgets/{key}" {
		t.Errorf("TriageOperations[0] = %+v, want method+path resolved from spec", report.TriageOperations[0])
	}

	var buf bytes.Buffer
	if err := renderMarkdown(&buf, report); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Operations pending triage in partial families") {
		t.Error("markdown should include the triage-operations section")
	}

	// A triage opId that vanishes from the spec is stale, same as a claim.
	widgets.TriageOperations = []string{"deleteWidget", "ghostTriageOp"}
	report = buildReport(families, fixtureMapping(t,
		MappingFamily{Tag: "Projects", Status: statusCovered, Resources: []ResourceEntry{{Name: "launchdarkly_project"}}},
		MappingFamily{Tag: "Shiny new feature", Status: statusTriage},
		MappingFamily{Tag: "<untagged>", Status: statusIgnored, Reason: "spec hygiene bucket"},
		widgets,
	), []string{"launchdarkly_project", "launchdarkly_widget"}, nil, "fixture")
	if len(report.StaleOperations) != 1 || report.StaleOperations[0].OperationID != "ghostTriageOp" {
		t.Errorf("StaleOperations = %+v, want [ghostTriageOp]", report.StaleOperations)
	}
}

func TestBuildReportIgnoredOperationSuppression(t *testing.T) {
	families := fixtureOperations(t)
	mapping := fixtureMapping(t,
		MappingFamily{Tag: "Projects", Status: statusCovered, Resources: []ResourceEntry{{Name: "launchdarkly_project"}}},
		MappingFamily{Tag: "Shiny new feature", Status: statusTriage},
		MappingFamily{Tag: "<untagged>", Status: statusIgnored, Reason: "spec hygiene bucket"},
		partialWidgets([]string{"getWidgets", "postWidget"},
			IgnoredOperation{ID: "deleteWidget", Reason: "bulk delete is UI-only"}),
	)
	report := buildReport(families, mapping, []string{"launchdarkly_project", "launchdarkly_widget"}, nil, "fixture")
	if report.HasDrift() {
		t.Fatalf("ignored operation must suppress drift, got %+v", report)
	}
}

func TestBuildReportStaleOperations(t *testing.T) {
	families := fixtureOperations(t)
	mapping := fixtureMapping(t,
		MappingFamily{Tag: "Projects", Status: statusCovered, Resources: []ResourceEntry{{Name: "launchdarkly_project"}}},
		MappingFamily{Tag: "Shiny new feature", Status: statusTriage},
		MappingFamily{Tag: "<untagged>", Status: statusIgnored, Reason: "spec hygiene bucket"},
		partialWidgets([]string{"getWidgets", "postWidget", "ghostOperation"},
			IgnoredOperation{ID: "deleteWidget", Reason: "bulk delete is UI-only"}),
	)
	report := buildReport(families, mapping, []string{"launchdarkly_project", "launchdarkly_widget"}, nil, "fixture")

	if !report.HasDrift() {
		t.Fatal("expected drift from stale ghostOperation claim")
	}
	if len(report.StaleOperations) != 1 || report.StaleOperations[0].OperationID != "ghostOperation" {
		t.Errorf("StaleOperations = %+v, want exactly [ghostOperation]", report.StaleOperations)
	}
	if len(report.UnclaimedOperations) != 0 {
		t.Errorf("UnclaimedOperations = %+v, want none", report.UnclaimedOperations)
	}
}

func TestBuildReportSkipsOperationChecksForStaleFamily(t *testing.T) {
	families := fixtureOperations(t)
	mapping := fixtureMapping(t,
		MappingFamily{Tag: "Projects", Status: statusCovered, Resources: []ResourceEntry{{Name: "launchdarkly_project"}}},
		MappingFamily{Tag: "Shiny new feature", Status: statusTriage},
		MappingFamily{Tag: "Widgets", Status: statusIgnored, Reason: "fixture filler"},
		MappingFamily{Tag: "<untagged>", Status: statusIgnored, Reason: "spec hygiene bucket"},
		MappingFamily{Tag: "Vanished", Status: statusPartial,
			Resources: []ResourceEntry{{Name: "launchdarkly_project", Operations: []string{"goneOp"}}}},
	)
	report := buildReport(families, mapping, []string{"launchdarkly_project"}, nil, "fixture")
	if len(report.StaleFamilies) != 1 || report.StaleFamilies[0] != "Vanished" {
		t.Fatalf("StaleFamilies = %v, want [Vanished]", report.StaleFamilies)
	}
	// The stale-family signal covers the whole entry; its ops must not be
	// double-reported as stale operations.
	if len(report.StaleOperations) != 0 {
		t.Errorf("StaleOperations = %+v, want none for a stale family", report.StaleOperations)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("disk full")
}

func TestRenderMarkdownSurfacesWriteErrors(t *testing.T) {
	report := buildReport(nil, &Mapping{}, nil, nil, "fixture")
	if err := renderMarkdown(failingWriter{}, report); err == nil {
		t.Fatal("renderMarkdown should return the write error")
	}
}

func TestMappingValidation(t *testing.T) {
	cases := []struct {
		name string
		fam  MappingFamily
	}{
		{"bad status", MappingFamily{Tag: "X", Status: "wat"}},
		{"ignored without reason", MappingFamily{Tag: "X", Status: statusIgnored}},
		{"covered without resources", MappingFamily{Tag: "X", Status: statusCovered}},
		{"empty tag", MappingFamily{Status: statusTriage}},
		{"partial resource without operations", MappingFamily{Tag: "X", Status: statusPartial,
			Resources: []ResourceEntry{{Name: "launchdarkly_x"}}}},
		{"operations on covered family", MappingFamily{Tag: "X", Status: statusCovered,
			Resources: []ResourceEntry{{Name: "launchdarkly_x", Operations: []string{"getX"}}}}},
		{"ignored_operations on covered family", MappingFamily{Tag: "X", Status: statusCovered,
			Resources:         []ResourceEntry{{Name: "launchdarkly_x"}},
			IgnoredOperations: []IgnoredOperation{{ID: "getX", Reason: "r"}}}},
		{"ignored operation without reason", MappingFamily{Tag: "X", Status: statusPartial,
			Resources:         []ResourceEntry{{Name: "launchdarkly_x", Operations: []string{"getX"}}},
			IgnoredOperations: []IgnoredOperation{{ID: "putX"}}}},
		{"operation claimed twice across resources", MappingFamily{Tag: "X", Status: statusPartial,
			Resources: []ResourceEntry{
				{Name: "launchdarkly_x", Operations: []string{"getX"}},
				{Name: "launchdarkly_y", Operations: []string{"getX"}}}}},
		{"operation both claimed and ignored", MappingFamily{Tag: "X", Status: statusPartial,
			Resources:         []ResourceEntry{{Name: "launchdarkly_x", Operations: []string{"getX"}}},
			IgnoredOperations: []IgnoredOperation{{ID: "getX", Reason: "r"}}}},
		{"triage_operations on covered family", MappingFamily{Tag: "X", Status: statusCovered,
			Resources:        []ResourceEntry{{Name: "launchdarkly_x"}},
			TriageOperations: []string{"getX"}}},
		{"operation both claimed and triaged", MappingFamily{Tag: "X", Status: statusPartial,
			Resources:        []ResourceEntry{{Name: "launchdarkly_x", Operations: []string{"getX"}}},
			TriageOperations: []string{"getX"}}},
		{"empty triage operationId", MappingFamily{Tag: "X", Status: statusPartial,
			Resources:        []ResourceEntry{{Name: "launchdarkly_x", Operations: []string{"getX"}}},
			TriageOperations: []string{""}}},
	}
	for _, tc := range cases {
		m := &Mapping{Version: 1, Families: []MappingFamily{tc.fam}}
		if err := m.validate(); err == nil {
			t.Errorf("%s: expected validation error", tc.name)
		}
	}
	dup := &Mapping{Version: 1, Families: []MappingFamily{
		{Tag: "X", Status: statusTriage}, {Tag: "X", Status: statusTriage},
	}}
	if err := dup.validate(); err == nil {
		t.Error("duplicate tags: expected validation error")
	}
}

func TestResourceEntryYAMLParsing(t *testing.T) {
	doc := `
version: 1
families:
  - tag: Tag-level
    status: covered
    resources:
      - launchdarkly_plain
  - tag: Op-level
    status: partial
    resources:
      - name: launchdarkly_rich
        operations:
          - getThing
          - putThing
    ignored_operations:
      - id: searchThings
        reason: runtime search, not declarative
    triage_operations:
      - postThingBulk
`
	var m Mapping
	if err := yaml.Unmarshal([]byte(doc), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := m.validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	plain := m.Families[0].Resources[0]
	if plain.Name != "launchdarkly_plain" || len(plain.Operations) != 0 {
		t.Errorf("bare-string entry = %+v, want name-only", plain)
	}
	rich := m.Families[1].Resources[0]
	if rich.Name != "launchdarkly_rich" || len(rich.Operations) != 2 {
		t.Errorf("object entry = %+v, want name + 2 operations", rich)
	}
	ig := m.Families[1].IgnoredOperations[0]
	if ig.ID != "searchThings" || ig.Reason == "" {
		t.Errorf("ignored_operations entry = %+v, want id + reason", ig)
	}
	if tr := m.Families[1].TriageOperations; len(tr) != 1 || tr[0] != "postThingBulk" {
		t.Errorf("triage_operations = %v, want [postThingBulk]", tr)
	}
}

func TestFamilySlice(t *testing.T) {
	slice, err := familySlice([]byte(fixtureSpec), "Projects")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Tag   string `json:"tag"`
		Paths map[string][]struct {
			Method string `json:"method"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(slice, &got); err != nil {
		t.Fatal(err)
	}
	if got.Tag != "Projects" || len(got.Paths) != 2 {
		t.Errorf("slice = tag %q with %d paths, want Projects with 2", got.Tag, len(got.Paths))
	}
	if ops := got.Paths["/api/v2/projects"]; len(ops) != 2 {
		t.Errorf("/api/v2/projects ops = %d, want 2 (GET+POST)", len(ops))
	}

	if _, err := familySlice([]byte(fixtureSpec), "No such family"); err == nil {
		t.Error("unknown family should error")
	}

	// The synthetic <untagged> family must resolve the same untagged paths
	// specOperations reports, not look for a literal tag named "<untagged>".
	untagged, err := familySlice([]byte(fixtureSpec), untaggedFamily)
	if err != nil {
		t.Fatalf("familySlice(%q) = %v, want untagged paths", untaggedFamily, err)
	}
	if err := json.Unmarshal(untagged, &got); err != nil {
		t.Fatal(err)
	}
	if got.Tag != untaggedFamily || len(got.Paths["/api/v2/untagged-thing"]) != 1 {
		t.Errorf("untagged slice = tag %q with %d ops for untagged-thing, want %q with 1",
			got.Tag, len(got.Paths["/api/v2/untagged-thing"]), untaggedFamily)
	}
}

func TestRegisteredTypes(t *testing.T) {
	resources, dataSources := registeredTypes()
	if len(resources) < 20 {
		t.Errorf("resources = %d, expected at least 20", len(resources))
	}
	if len(dataSources) < 15 {
		t.Errorf("dataSources = %d, expected at least 15", len(dataSources))
	}
	found := false
	for _, r := range resources {
		if r == "launchdarkly_project" {
			found = true
		}
	}
	if !found {
		t.Error("launchdarkly_project missing from registered resources")
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

const fixtureSpec = `{
  "paths": {
    "/api/v2/projects": {
      "get": {"tags": ["Projects"]},
      "post": {"tags": ["Projects"]},
      "parameters": [{"name": "ignored"}]
    },
    "/api/v2/projects/{projectKey}": {
      "get": {"tags": ["Projects"]},
      "delete": {"tags": ["Projects"]}
    },
    "/api/v2/shiny/{key}": {
      "get": {"tags": ["Shiny new feature"]},
      "post": {"tags": ["Shiny new feature"]}
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

func TestSpecFamilies(t *testing.T) {
	families, err := specFamilies([]byte(fixtureSpec))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(families["Projects"]); got != 2 {
		t.Errorf("Projects paths = %d, want 2", got)
	}
	if got := len(families["<untagged>"]); got != 1 {
		t.Errorf("<untagged> paths = %d, want 1", got)
	}
	if _, ok := families["parameters"]; ok {
		t.Error("non-method path-item keys must not become families")
	}
}

func TestSpecFamiliesEmptySpec(t *testing.T) {
	if _, err := specFamilies([]byte(`{"paths": {}}`)); err == nil {
		t.Error("empty spec should error, not report full coverage")
	}
}

func TestBuildReportDetectsDrift(t *testing.T) {
	families, err := specFamilies([]byte(fixtureSpec))
	if err != nil {
		t.Fatal(err)
	}
	mapping := fixtureMapping(t,
		MappingFamily{Tag: "Projects", Status: statusCovered, Resources: []string{"launchdarkly_project"}},
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
	families, err := specFamilies([]byte(fixtureSpec))
	if err != nil {
		t.Fatal(err)
	}
	mapping := fixtureMapping(t,
		MappingFamily{Tag: "Projects", Status: statusCovered, Resources: []string{"launchdarkly_project"}},
		MappingFamily{Tag: "Shiny new feature", Status: statusTriage},
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
	for _, key := range []string{"new_families", "stale_families", "unmapped_resources"} {
		if strings.Contains(string(raw), fmt.Sprintf("%q:null", key)) {
			t.Errorf("%s serializes as null, want []", key)
		}
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

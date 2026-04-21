package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateOperations(t *testing.T) {
	t.Parallel()

	doc := openAPIDocument{Paths: map[string]map[string]json.RawMessage{
		"/api/v2/teams": {
			"post": json.RawMessage(`{"operationId":"createTeam"}`),
		},
		"/api/v2/teams/{teamKey}": {
			"get":    json.RawMessage(`{"operationId":"getTeam"}`),
			"patch":  json.RawMessage(`{"operationId":"patchTeam"}`),
			"delete": json.RawMessage(`{"operationId":"deleteTeam"}`),
		},
	}}

	resources := []frameworkResourceConfig{
		{
			TerraformName: "team",
			Operations: operationMappings{
				Create: operationRef{Path: "/api/v2/teams", Method: "POST"},
				Read:   operationRef{Path: "/api/v2/teams/{teamKey}", Method: "GET"},
				Update: operationRef{Path: "/api/v2/teams/{teamKey}", Method: "PATCH"},
				Delete: operationRef{Path: "/api/v2/teams/{teamKey}", Method: "DELETE"},
			},
		},
	}

	if err := validateOperations(resources, doc); err != nil {
		t.Fatalf("expected operations to validate, got error: %v", err)
	}
}

func TestRunGeneratesFiles(t *testing.T) {
	t.Parallel()

	openAPI := map[string]interface{}{
		"paths": map[string]interface{}{
			"/api/v2/teams": map[string]interface{}{"post": map[string]interface{}{}},
			"/api/v2/teams/{teamKey}": map[string]interface{}{
				"get":    map[string]interface{}{},
				"patch":  map[string]interface{}{},
				"delete": map[string]interface{}{},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(openAPI)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	templateDir := filepath.Join(tempDir, "templates")
	outDir := filepath.Join(tempDir, "launchdarkly")
	testsOutDir := filepath.Join(outDir, "tests")

	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("failed to create template dir: %v", err)
	}

	config := generationConfig{
		Version:  configVersion,
		Provider: providerConfig{Name: "launchdarkly", OpenAPIURL: server.URL},
		Framework: frameworkConfig{Resources: []frameworkResourceConfig{
			{
				TerraformName:     "team",
				FrameworkTypeName: "generated_team",
				Constructor:       "NewGeneratedTeamResource",
				Implementation:    "team",
				RegisterFramework: true,
				IdentityFields:    []string{"key"},
				MutableFields:     []string{"name"},
				Operations: operationMappings{
					Create: operationRef{Path: "/api/v2/teams", Method: "POST"},
					Read:   operationRef{Path: "/api/v2/teams/{teamKey}", Method: "GET"},
					Update: operationRef{Path: "/api/v2/teams/{teamKey}", Method: "PATCH"},
					Delete: operationRef{Path: "/api/v2/teams/{teamKey}", Method: "DELETE"},
				},
				Test: acceptanceTestConfig{Enabled: true, Scenario: "team"},
			},
		}},
	}

	configRaw, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, configRaw, 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	templates := map[string]string{
		"plugin_provider_gen.gotmpl":               "package launchdarkly\n\nimport \"github.com/hashicorp/terraform-plugin-framework/resource\"\n\nfunc generatedFrameworkResources() []func() resource.Resource { return nil }\n",
		"openapi_provider_metadata_gen.gotmpl":     "package launchdarkly\n\nvar _ = \"metadata\"\n",
		"generated_openapi_resources.gotmpl":       "package launchdarkly\n\nvar _ = \"generated-resources\"\n",
		"generated_openapi_acceptance_test.gotmpl": "package tests\n\nimport \"testing\"\n\nfunc TestGeneratedTemplate(t *testing.T) {}\n",
	}
	for fileName, content := range templates {
		if err := os.WriteFile(filepath.Join(templateDir, fileName), []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write template %s: %v", fileName, err)
		}
	}

	g := newGenerator()
	if err := g.run(context.Background(), configPath, templateDir, outDir, testsOutDir); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	requiredFiles := []string{
		filepath.Join(outDir, "plugin_provider_gen.go"),
		filepath.Join(outDir, "openapi_provider_metadata_gen.go"),
		filepath.Join(outDir, "openapi_generated_resources_gen.go"),
		filepath.Join(testsOutDir, "generated_openapi_acceptance_test.go"),
	}
	for _, filePath := range requiredFiles {
		if _, err := os.Stat(filePath); err != nil {
			t.Fatalf("expected output file %s to exist: %v", filePath, err)
		}
	}
}

func TestDiscoverResources(t *testing.T) {
	t.Parallel()

	doc := openAPIDocument{
		Paths: map[string]map[string]json.RawMessage{
			"/api/v2/projects": {
				"post": json.RawMessage(`{}`),
			},
			"/api/v2/projects/{projectKey}": {
				"get":    json.RawMessage(`{}`),
				"patch":  json.RawMessage(`{}`),
				"delete": json.RawMessage(`{}`),
			},
		},
	}

	resources := discoverResources(doc)
	if len(resources) == 0 {
		t.Fatalf("expected discovered resources, got none")
	}

	var discovered frameworkResourceConfig
	found := false
	for _, resource := range resources {
		if resource.FrameworkTypeName == "generated_project" {
			discovered = resource
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected generated_project in discovered resources: %#v", resources)
	}

	if discovered.Implementation != "generic" {
		t.Fatalf("expected generic implementation, got %q", discovered.Implementation)
	}
	if discovered.Operations.Create.Path != "/api/v2/projects" || discovered.Operations.Create.Method != http.MethodPost {
		t.Fatalf("unexpected create operation: %#v", discovered.Operations.Create)
	}
	if discovered.Operations.Read.Path != "/api/v2/projects/{projectKey}" || discovered.Operations.Read.Method != http.MethodGet {
		t.Fatalf("unexpected read operation: %#v", discovered.Operations.Read)
	}
	if boolDefault(discovered.Enabled, true) {
		t.Fatalf("expected discovered resource to default disabled")
	}
	if !boolDefault(discovered.Experimental, false) {
		t.Fatalf("expected discovered resource to be experimental")
	}
	if discovered.Test.Scenario != "generic" || !discovered.Test.Enabled {
		t.Fatalf("unexpected test configuration: %#v", discovered.Test)
	}
}

func TestMergeResourcesOverlayOverridesCatalog(t *testing.T) {
	t.Parallel()

	catalog := []frameworkResourceConfig{
		{
			TerraformName:     "project",
			FrameworkTypeName: "generated_project",
			Constructor:       "NewGeneratedProjectResource",
			Implementation:    "generic",
			RegisterFramework: true,
			Operations: operationMappings{
				Create: operationRef{Path: "/api/v2/projects", Method: "POST"},
				Read:   operationRef{Path: "/api/v2/projects/{projectKey}", Method: "GET"},
			},
			Enabled:      boolPtr(false),
			Experimental: boolPtr(true),
			Test: acceptanceTestConfig{
				Enabled:  true,
				Scenario: "generic",
			},
		},
	}
	overlay := []frameworkResourceConfig{
		{
			TerraformName:     "project",
			FrameworkTypeName: "generated_project",
			Constructor:       "NewGeneratedProjectResource",
			Implementation:    "project_basic",
			RegisterFramework: true,
			Operations: operationMappings{
				Create: operationRef{Path: "/api/v2/projects", Method: "POST"},
				Read:   operationRef{Path: "/api/v2/projects/{projectKey}", Method: "GET"},
				Update: operationRef{Path: "/api/v2/projects/{projectKey}", Method: "PATCH"},
				Delete: operationRef{Path: "/api/v2/projects/{projectKey}", Method: "DELETE"},
			},
			Test: acceptanceTestConfig{
				Enabled:  true,
				Scenario: "project",
			},
		},
	}

	merged := mergeResources(catalog, overlay)
	if len(merged) != 1 {
		t.Fatalf("expected one merged resource, got %d", len(merged))
	}
	if merged[0].Implementation != "project_basic" {
		t.Fatalf("expected overlay implementation to win, got %q", merged[0].Implementation)
	}
	if !boolDefault(merged[0].Enabled, false) {
		t.Fatalf("expected merged resource enabled by overlay default")
	}
	if boolDefault(merged[0].Experimental, true) {
		t.Fatalf("expected merged resource to default non-experimental for overlay")
	}
	if merged[0].Test.Scenario != "project" {
		t.Fatalf("expected overlay test scenario to win, got %q", merged[0].Test.Scenario)
	}
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"
	"unicode"
)

const configVersion = "v1"

type operationRef struct {
	Path   string `json:"path"`
	Method string `json:"method"`
}

type operationMappings struct {
	Create operationRef `json:"create"`
	Read   operationRef `json:"read"`
	Update operationRef `json:"update"`
	Delete operationRef `json:"delete"`
}

type acceptanceTestConfig struct {
	Enabled  bool   `json:"enabled"`
	Scenario string `json:"scenario"`
	Fixture  string `json:"fixture"`
}

type frameworkResourceConfig struct {
	TerraformName     string               `json:"terraform_name"`
	FrameworkTypeName string               `json:"framework_type_name"`
	Constructor       string               `json:"constructor"`
	Implementation    string               `json:"implementation"`
	RegisterFramework bool                 `json:"register_framework"`
	Enabled           *bool                `json:"enabled,omitempty"`
	Experimental      *bool                `json:"experimental,omitempty"`
	RolloutPhase      string               `json:"rollout_phase"`
	ModifyPlanHook    string               `json:"modify_plan_hook"`
	IdentityFields    []string             `json:"identity_fields"`
	MutableFields     []string             `json:"mutable_fields"`
	ImportIgnore      []string             `json:"import_ignore"`
	Operations        operationMappings    `json:"operations"`
	Test              acceptanceTestConfig `json:"test"`
}

type providerConfig struct {
	Name       string `json:"name"`
	OpenAPIURL string `json:"openapi_url"`
}

type frameworkConfig struct {
	Resources []frameworkResourceConfig `json:"resources"`
}

type generationConfig struct {
	Version   string          `json:"version"`
	Provider  providerConfig  `json:"provider"`
	Framework frameworkConfig `json:"framework"`
}

type openAPIDocument struct {
	Paths map[string]map[string]json.RawMessage `json:"paths"`
}

type templateResourceData struct {
	TerraformName     string
	FrameworkTypeName string
	GoName            string
	Constructor       string
	Implementation    string
	RegisterFramework bool
	Enabled           bool
	Experimental      bool
	RolloutPhase      string
	ModifyPlanHook    string
	IdentityFields    []string
	MutableFields     []string
	ImportIgnore      []string
	Operations        operationMappings
	Scenario          string
	TestFixture       string
	PrimaryIdentity   string
}

type templateData struct {
	ProviderName string
	Resources    []templateResourceData
}

type runOptions struct {
	OverlayPath string
	CatalogPath string
	CatalogOut  string
	DiscoverOnly bool
	TemplateDir string
	OutDir      string
	TestsOutDir string
}

type generator struct {
	httpClient *http.Client
	readFile   func(string) ([]byte, error)
	writeFile  func(string, []byte, os.FileMode) error
	mkdirAll   func(string, os.FileMode) error
}

func newGenerator() *generator {
	return &generator{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		readFile:   os.ReadFile,
		writeFile:  os.WriteFile,
		mkdirAll:   os.MkdirAll,
	}
}

func (g *generator) run(ctx context.Context, configPath string, templateDir string, outDir string, testsOutDir string) error {
	return g.runWithOptions(ctx, runOptions{
		OverlayPath: configPath,
		TemplateDir: templateDir,
		OutDir:      outDir,
		TestsOutDir: testsOutDir,
	})
}

func (g *generator) runWithOptions(ctx context.Context, opts runOptions) error {
	overlayCfg, err := g.loadConfig(opts.OverlayPath)
	if err != nil {
		return err
	}

	opDoc, err := g.fetchOpenAPI(ctx, overlayCfg.Provider.OpenAPIURL)
	if err != nil {
		return err
	}

	discoveredResources := discoverResources(opDoc)
	if opts.CatalogOut != "" {
		if err := g.writeCatalog(opts.CatalogOut, overlayCfg.Provider, discoveredResources); err != nil {
			return err
		}
	}
	if opts.DiscoverOnly {
		return nil
	}

	resources := []frameworkResourceConfig{}
	if opts.CatalogPath != "" {
		catalogCfg, loadCatalogErr := g.loadConfig(opts.CatalogPath)
		if loadCatalogErr != nil {
			return loadCatalogErr
		}
		resources = mergeResources(catalogCfg.Framework.Resources, overlayCfg.Framework.Resources)
	} else {
		resources = normalizeResources(overlayCfg.Framework.Resources, false)
	}

	if err := validateOperations(resources, opDoc); err != nil {
		return err
	}

	data := buildTemplateData(overlayCfg.Provider.Name, resources)

	renderTargets := []struct {
		TemplateName string
		OutputPath   string
	}{
		{TemplateName: "plugin_provider_gen.gotmpl", OutputPath: filepath.Join(opts.OutDir, "plugin_provider_gen.go")},
		{TemplateName: "openapi_provider_metadata_gen.gotmpl", OutputPath: filepath.Join(opts.OutDir, "openapi_provider_metadata_gen.go")},
		{TemplateName: "generated_openapi_resources.gotmpl", OutputPath: filepath.Join(opts.OutDir, "openapi_generated_resources_gen.go")},
		{TemplateName: "generated_openapi_acceptance_test.gotmpl", OutputPath: filepath.Join(opts.TestsOutDir, "generated_openapi_acceptance_test.go")},
	}

	for _, target := range renderTargets {
		rendered, renderErr := g.renderTemplate(filepath.Join(opts.TemplateDir, target.TemplateName), data)
		if renderErr != nil {
			return renderErr
		}

		formatted, formatErr := format.Source(rendered)
		if formatErr != nil {
			return fmt.Errorf("failed to format %s: %w", target.OutputPath, formatErr)
		}

		if err := g.mkdirAll(filepath.Dir(target.OutputPath), 0o755); err != nil {
			return err
		}

		if err := g.writeFile(target.OutputPath, formatted, 0o644); err != nil {
			return err
		}
	}

	return nil
}

func (g *generator) loadConfig(path string) (generationConfig, error) {
	var cfg generationConfig

	raw, err := g.readFile(path)
	if err != nil {
		return cfg, fmt.Errorf("failed to read config %s: %w", path, err)
	}

	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config %s: %w", path, err)
	}

	if cfg.Version != configVersion {
		return cfg, fmt.Errorf("unsupported config version %q, expected %q", cfg.Version, configVersion)
	}
	if cfg.Provider.Name == "" {
		return cfg, errors.New("provider.name is required")
	}
	if cfg.Provider.OpenAPIURL == "" {
		return cfg, errors.New("provider.openapi_url is required")
	}

	for i := range cfg.Framework.Resources {
		resource := &cfg.Framework.Resources[i]
		if resource.TerraformName == "" {
			return cfg, fmt.Errorf("framework.resources[%d].terraform_name is required", i)
		}
		if resource.FrameworkTypeName == "" {
			return cfg, fmt.Errorf("framework.resources[%d].framework_type_name is required", i)
		}
		if resource.RegisterFramework && resource.Constructor == "" {
			return cfg, fmt.Errorf("framework.resources[%d].constructor is required", i)
		}
		if resource.Test.Enabled && resource.Test.Scenario == "" {
			return cfg, fmt.Errorf("framework.resources[%d].test.scenario is required when test.enabled is true", i)
		}
	}

	return cfg, nil
}

func (g *generator) fetchOpenAPI(ctx context.Context, url string) (openAPIDocument, error) {
	var doc openAPIDocument

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return doc, fmt.Errorf("failed to create openapi request: %w", err)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return doc, fmt.Errorf("failed to download openapi spec from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return doc, fmt.Errorf("failed to download openapi spec from %s: status %d", url, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return doc, fmt.Errorf("failed to decode openapi spec from %s: %w", url, err)
	}

	if len(doc.Paths) == 0 {
		return doc, errors.New("openapi spec missing paths")
	}

	return doc, nil
}

func (g *generator) writeCatalog(path string, provider providerConfig, resources []frameworkResourceConfig) error {
	cfg := generationConfig{
		Version:  configVersion,
		Provider: provider,
		Framework: frameworkConfig{
			Resources: normalizeResources(resources, true),
		},
	}

	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal catalog %s: %w", path, err)
	}

	if err := g.mkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	if err := g.writeFile(path, append(raw, '\n'), 0o644); err != nil {
		return fmt.Errorf("failed to write catalog %s: %w", path, err)
	}

	return nil
}

func discoverResources(doc openAPIDocument) []frameworkResourceConfig {
	methodsByPath := make(map[string]map[string]struct{}, len(doc.Paths))
	paths := make([]string, 0, len(doc.Paths))
	for path, operations := range doc.Paths {
		paths = append(paths, path)
		methodSet := make(map[string]struct{}, len(operations))
		for method := range operations {
			methodSet[strings.ToUpper(method)] = struct{}{}
		}
		methodsByPath[path] = methodSet
	}
	sort.Strings(paths)

	resources := make([]frameworkResourceConfig, 0)
	usedTypeNames := make(map[string]struct{})
	consumedItemPaths := make(map[string]struct{})

	addResource := func(resource frameworkResourceConfig) {
		resource = normalizeResource(resource, true)
		baseType := resource.FrameworkTypeName
		candidateType := baseType
		suffix := 2
		for {
			if _, exists := usedTypeNames[candidateType]; !exists {
				break
			}
			candidateType = fmt.Sprintf("%s_%d", baseType, suffix)
			suffix++
		}
		resource.FrameworkTypeName = candidateType
		if resource.RegisterFramework {
			resource.Constructor = fmt.Sprintf("New%sResource", snakeToCamel(resource.FrameworkTypeName))
		}
		usedTypeNames[candidateType] = struct{}{}
		resources = append(resources, resource)
	}

	for _, path := range paths {
		if !hasMethod(methodsByPath[path], http.MethodPost) {
			continue
		}
		resource, itemPath, ok := discoverFromCollection(path, methodsByPath)
		if !ok {
			continue
		}
		if itemPath != "" {
			consumedItemPaths[itemPath] = struct{}{}
		}
		addResource(resource)
	}

	for _, path := range paths {
		if _, consumed := consumedItemPaths[path]; consumed {
			continue
		}
		if hasMethod(methodsByPath[path], http.MethodPost) {
			continue
		}
		resource, ok := discoverFromItem(path, methodsByPath[path])
		if !ok {
			continue
		}
		addResource(resource)
	}

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].FrameworkTypeName < resources[j].FrameworkTypeName
	})
	return resources
}

func discoverFromCollection(path string, methodsByPath map[string]map[string]struct{}) (frameworkResourceConfig, string, bool) {
	itemPath, itemParam := findPrimaryItemPath(path, methodsByPath)

	read := operationRef{}
	update := operationRef{}
	deleteOp := operationRef{}
	identityFields := []string{}

	if itemPath != "" {
		if hasMethod(methodsByPath[itemPath], http.MethodGet) {
			read = operationRef{Path: itemPath, Method: http.MethodGet}
		}
		if hasMethod(methodsByPath[itemPath], http.MethodPatch) {
			update = operationRef{Path: itemPath, Method: http.MethodPatch}
		} else if hasMethod(methodsByPath[itemPath], http.MethodPut) {
			update = operationRef{Path: itemPath, Method: http.MethodPut}
		}
		if hasMethod(methodsByPath[itemPath], http.MethodDelete) {
			deleteOp = operationRef{Path: itemPath, Method: http.MethodDelete}
		}
		if itemParam != "" {
			identityFields = append(identityFields, normalizeIdentityField(itemParam))
		}
	} else {
		if hasMethod(methodsByPath[path], http.MethodGet) {
			read = operationRef{Path: path, Method: http.MethodGet}
		}
		if hasMethod(methodsByPath[path], http.MethodPatch) {
			update = operationRef{Path: path, Method: http.MethodPatch}
		} else if hasMethod(methodsByPath[path], http.MethodPut) {
			update = operationRef{Path: path, Method: http.MethodPut}
		}
		if hasMethod(methodsByPath[path], http.MethodDelete) {
			deleteOp = operationRef{Path: path, Method: http.MethodDelete}
		}
		identityFields = extractPathParams(path)
	}

	if read.Path == "" {
		return frameworkResourceConfig{}, "", false
	}

	resourceName := resourceNameFromPath(path)
	resource := frameworkResourceConfig{
		TerraformName:     resourceName,
		FrameworkTypeName: "generated_" + resourceName,
		RegisterFramework: true,
		Implementation:    "generic",
		IdentityFields:    identityFields,
		MutableFields:     []string{},
		ImportIgnore:      []string{},
		Operations: operationMappings{
			Create: operationRef{Path: path, Method: http.MethodPost},
			Read:   read,
			Update: update,
			Delete: deleteOp,
		},
		Test: acceptanceTestConfig{
			Enabled:  true,
			Scenario: "generic",
		},
		Enabled:      boolPtr(false),
		Experimental: boolPtr(true),
		RolloutPhase: "read_import",
	}

	if resource.Operations.Update.Path == "" && resource.Operations.Delete.Path == "" {
		resource.RolloutPhase = "read_import"
	}
	return resource, itemPath, true
}

func discoverFromItem(path string, methods map[string]struct{}) (frameworkResourceConfig, bool) {
	if !hasMethod(methods, http.MethodGet) {
		return frameworkResourceConfig{}, false
	}
	if !hasMethod(methods, http.MethodPatch) && !hasMethod(methods, http.MethodPut) && !hasMethod(methods, http.MethodDelete) {
		return frameworkResourceConfig{}, false
	}
	identityFields := extractPathParams(path)
	if len(identityFields) == 0 {
		return frameworkResourceConfig{}, false
	}

	update := operationRef{}
	if hasMethod(methods, http.MethodPatch) {
		update = operationRef{Path: path, Method: http.MethodPatch}
	} else if hasMethod(methods, http.MethodPut) {
		update = operationRef{Path: path, Method: http.MethodPut}
	}
	deleteOp := operationRef{}
	if hasMethod(methods, http.MethodDelete) {
		deleteOp = operationRef{Path: path, Method: http.MethodDelete}
	}
	create := operationRef{}
	if hasMethod(methods, http.MethodPost) {
		create = operationRef{Path: path, Method: http.MethodPost}
	} else {
		create = update
	}

	resourceName := resourceNameFromPath(path)
	resource := frameworkResourceConfig{
		TerraformName:     resourceName,
		FrameworkTypeName: "generated_" + resourceName,
		RegisterFramework: true,
		Implementation:    "generic",
		IdentityFields:    identityFields,
		MutableFields:     []string{},
		ImportIgnore:      []string{},
		Operations: operationMappings{
			Create: create,
			Read:   operationRef{Path: path, Method: http.MethodGet},
			Update: update,
			Delete: deleteOp,
		},
		Test: acceptanceTestConfig{
			Enabled:  true,
			Scenario: "generic",
		},
		Enabled:      boolPtr(false),
		Experimental: boolPtr(true),
		RolloutPhase: "read_import",
	}
	return resource, true
}

func findPrimaryItemPath(collectionPath string, methodsByPath map[string]map[string]struct{}) (string, string) {
	prefix := collectionPath + "/"
	bestPath := ""
	bestParam := ""
	bestScore := -1

	for path, methods := range methodsByPath {
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		remainder := strings.TrimPrefix(path, prefix)
		if strings.Contains(remainder, "/") {
			continue
		}
		if !isPathParam(remainder) {
			continue
		}

		score := 0
		if hasMethod(methods, http.MethodGet) {
			score += 4
		}
		if hasMethod(methods, http.MethodPatch) || hasMethod(methods, http.MethodPut) {
			score += 2
		}
		if hasMethod(methods, http.MethodDelete) {
			score++
		}
		if score > bestScore {
			bestScore = score
			bestPath = path
			bestParam = strings.TrimSuffix(strings.TrimPrefix(remainder, "{"), "}")
		}
	}

	if bestPath == "" {
		return "", ""
	}
	return bestPath, bestParam
}

func hasMethod(methods map[string]struct{}, method string) bool {
	if len(methods) == 0 {
		return false
	}
	_, ok := methods[strings.ToUpper(method)]
	return ok
}

func resourceNameFromPath(path string) string {
	trimmed := strings.TrimPrefix(path, "/")
	if strings.HasPrefix(trimmed, "api/v2/") {
		trimmed = strings.TrimPrefix(trimmed, "api/v2/")
	}
	segments := strings.Split(trimmed, "/")
	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment == "" || isPathParam(segment) {
			continue
		}
		normalized := sanitizeToken(segment)
		if normalized == "" {
			continue
		}
		parts = append(parts, singularize(normalized))
	}
	if len(parts) == 0 {
		return "root"
	}
	return strings.Join(parts, "_")
}

func sanitizeToken(token string) string {
	token = strings.TrimSpace(token)
	token = strings.ReplaceAll(token, "-", "_")
	token = strings.ReplaceAll(token, ".", "_")
	token = toSnakeCase(token)

	runes := make([]rune, 0, len(token))
	for _, r := range token {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			runes = append(runes, r)
		}
	}
	return strings.Trim(strings.ReplaceAll(string(runes), "__", "_"), "_")
}

func singularize(token string) string {
	switch {
	case strings.HasSuffix(token, "ies") && len(token) > 3:
		return strings.TrimSuffix(token, "ies") + "y"
	case strings.HasSuffix(token, "ses") && len(token) > 3:
		return strings.TrimSuffix(token, "es")
	case strings.HasSuffix(token, "s") && len(token) > 1:
		return strings.TrimSuffix(token, "s")
	default:
		return token
	}
}

func extractPathParams(path string) []string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	params := make([]string, 0)
	seen := make(map[string]struct{})
	for _, segment := range segments {
		if !isPathParam(segment) {
			continue
		}
		paramName := strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
		identity := normalizeIdentityField(paramName)
		if identity == "" {
			continue
		}
		if _, exists := seen[identity]; exists {
			continue
		}
		seen[identity] = struct{}{}
		params = append(params, identity)
	}
	return params
}

func normalizeIdentityField(param string) string {
	normalized := sanitizeToken(param)
	if normalized == "" {
		return normalized
	}
	return normalized
}

func isPathParam(segment string) bool {
	return strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}")
}

func mergeResources(catalogResources, overlayResources []frameworkResourceConfig) []frameworkResourceConfig {
	merged := make(map[string]frameworkResourceConfig, len(catalogResources)+len(overlayResources))

	for _, resource := range normalizeResources(catalogResources, true) {
		merged[resource.FrameworkTypeName] = resource
	}
	for _, resource := range normalizeResources(overlayResources, false) {
		merged[resource.FrameworkTypeName] = resource
	}

	out := make([]frameworkResourceConfig, 0, len(merged))
	for _, resource := range merged {
		out = append(out, resource)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].FrameworkTypeName < out[j].FrameworkTypeName
	})
	return out
}

func normalizeResources(resources []frameworkResourceConfig, fromCatalog bool) []frameworkResourceConfig {
	normalized := make([]frameworkResourceConfig, 0, len(resources))
	for _, resource := range resources {
		normalized = append(normalized, normalizeResource(resource, fromCatalog))
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].FrameworkTypeName < normalized[j].FrameworkTypeName
	})
	return normalized
}

func normalizeResource(resource frameworkResourceConfig, fromCatalog bool) frameworkResourceConfig {
	if resource.FrameworkTypeName == "" {
		resource.FrameworkTypeName = "generated_" + sanitizeToken(resource.TerraformName)
	}
	if resource.RegisterFramework && resource.Constructor == "" {
		resource.Constructor = fmt.Sprintf("New%sResource", snakeToCamel(resource.FrameworkTypeName))
	}
	if resource.Enabled == nil {
		resource.Enabled = boolPtr(!fromCatalog)
	}
	if resource.Experimental == nil {
		resource.Experimental = boolPtr(fromCatalog)
	}
	if resource.RolloutPhase == "" {
		if resource.Implementation == "generic" {
			resource.RolloutPhase = "read_import"
		} else {
			resource.RolloutPhase = "full"
		}
	}
	if len(resource.IdentityFields) == 0 {
		if resource.Operations.Read.Path != "" {
			resource.IdentityFields = extractPathParams(resource.Operations.Read.Path)
		} else if resource.Operations.Create.Path != "" {
			resource.IdentityFields = extractPathParams(resource.Operations.Create.Path)
		}
	}
	resource.IdentityFields = uniqueSortedStrings(resource.IdentityFields)
	resource.MutableFields = uniqueSortedStrings(resource.MutableFields)
	resource.ImportIgnore = uniqueSortedStrings(resource.ImportIgnore)
	if resource.Test.Enabled && resource.Test.Scenario == "" {
		resource.Test.Scenario = "generic"
	}
	return resource
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func validateOperations(resources []frameworkResourceConfig, doc openAPIDocument) error {
	var errs []string

	for _, res := range resources {
		checks := map[string]operationRef{
			"create": res.Operations.Create,
			"read":   res.Operations.Read,
			"update": res.Operations.Update,
			"delete": res.Operations.Delete,
		}

		for operationType, op := range checks {
			if op.Path == "" || op.Method == "" {
				continue
			}
			pathItem, ok := doc.Paths[op.Path]
			if !ok {
				errs = append(errs, fmt.Sprintf("resource %q %s path %q not found in openapi spec", res.FrameworkTypeName, operationType, op.Path))
				continue
			}
			if _, ok := pathItem[strings.ToLower(op.Method)]; !ok {
				errs = append(errs, fmt.Sprintf("resource %q %s method %q not found for path %q", res.FrameworkTypeName, operationType, strings.ToUpper(op.Method), op.Path))
			}
		}
	}

	if len(errs) > 0 {
		sort.Strings(errs)
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

func buildTemplateData(providerName string, resources []frameworkResourceConfig) templateData {
	dataResources := make([]templateResourceData, 0, len(resources))
	for _, res := range resources {
		primaryIdentity := ""
		if len(res.IdentityFields) > 0 {
			primaryIdentity = res.IdentityFields[0]
		}
		dataResources = append(dataResources, templateResourceData{
			TerraformName:     res.TerraformName,
			FrameworkTypeName: res.FrameworkTypeName,
			GoName:            snakeToCamel(res.FrameworkTypeName),
			Constructor:       res.Constructor,
			Implementation:    res.Implementation,
			RegisterFramework: res.RegisterFramework,
			Enabled:           boolDefault(res.Enabled, true),
			Experimental:      boolDefault(res.Experimental, false),
			RolloutPhase:      res.RolloutPhase,
			ModifyPlanHook:    res.ModifyPlanHook,
			IdentityFields:    append([]string(nil), res.IdentityFields...),
			MutableFields:     append([]string(nil), res.MutableFields...),
			ImportIgnore:      append([]string(nil), res.ImportIgnore...),
			Operations:        res.Operations,
			Scenario:          res.Test.Scenario,
			TestFixture:       res.Test.Fixture,
			PrimaryIdentity:   primaryIdentity,
		})
	}

	sort.Slice(dataResources, func(i, j int) bool {
		return dataResources[i].FrameworkTypeName < dataResources[j].FrameworkTypeName
	})

	return templateData{
		ProviderName: providerName,
		Resources:    dataResources,
	}
}

func (g *generator) renderTemplate(path string, data templateData) ([]byte, error) {
	rawTemplate, err := g.readFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read template %s: %w", path, err)
	}

	tmpl, err := template.New(filepath.Base(path)).Funcs(template.FuncMap{
		"quoteSlice": quoteSlice,
		"isScenario": func(resource templateResourceData, scenario string) bool {
			return resource.Scenario == scenario
		},
		"isImplementation": func(resource templateResourceData, implementation string) bool {
			return resource.Implementation == implementation
		},
		"fieldName": snakeToCamel,
		"contains": func(values []string, target string) bool {
			for _, value := range values {
				if value == target {
					return true
				}
			}
			return false
		},
		"hasEnabledExperimental": func(resources []templateResourceData) bool {
			for _, resource := range resources {
				if resource.RegisterFramework && resource.Enabled && resource.Experimental {
					return true
				}
			}
			return false
		},
	}).Parse(string(rawTemplate))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", path, err)
	}

	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, data); err != nil {
		return nil, fmt.Errorf("failed to render template %s: %w", path, err)
	}

	return buf.Bytes(), nil
}

func quoteSlice(values []string) string {
	if len(values) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return strings.Join(quoted, ", ")
}

func snakeToCamel(input string) string {
	parts := strings.Split(input, "_")
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

func toSnakeCase(input string) string {
	if input == "" {
		return input
	}
	var out []rune
	var prev rune
	for i, r := range input {
		if r == '-' || r == ' ' || r == '.' {
			out = append(out, '_')
			prev = '_'
			continue
		}
		if unicode.IsUpper(r) {
			if i > 0 && prev != '_' {
				out = append(out, '_')
			}
			out = append(out, unicode.ToLower(r))
			prev = unicode.ToLower(r)
			continue
		}
		out = append(out, unicode.ToLower(r))
		prev = unicode.ToLower(r)
	}
	return strings.Trim(strings.ReplaceAll(string(out), "__", "_"), "_")
}

func boolPtr(value bool) *bool {
	return &value
}

func boolDefault(value *bool, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	return *value
}

package launchdarkly

// obsFake is an in-memory implementation of the beta observability filter
// REST contract (GET/PUT filter-settings, CRUD filter-rules), used by the unit
// tests in place of LaunchDarkly. It deliberately depends only on the
// standard library.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var obsFakeProducts = []string{"sessions", "errors", "logs", "traces", "metrics"}

var obsFakeKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,255}$`)

type obsFakeRule struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	ProductType string   `json:"productType"`
	Enabled     bool     `json:"enabled"`
	Query       string   `json:"query"`
	Operation   string   `json:"operation"`
	Rate        float64  `json:"rate"`
	Attributes  []string `json:"attributes"`
	Version     int64    `json:"version"`
	CreatedAt   string   `json:"createdAt"` // RFC 3339
	UpdatedAt   string   `json:"updatedAt"`
}

type obsFakeProject struct {
	rules   map[string]*obsFakeRule
	order   map[string][]string
	limits  map[string]*int64
	version int64
}

type obsFake struct {
	mu       sync.Mutex
	projects map[string]*obsFakeProject
	// missingProjects answer 404 to every request.
	missingProjects map[string]bool
	// beforeRulePut, when set, runs once before the next rule PUT is
	// handled (with the lock held), to simulate a concurrent UI edit.
	beforeRulePut func(p *obsFakeProject, key string)
	// logf, when set, receives one line per request.
	logf func(format string, args ...interface{})
}

func newObsFake() *obsFake {
	return &obsFake{projects: map[string]*obsFakeProject{}, missingProjects: map[string]bool{}}
}

func (f *obsFake) project(key string) *obsFakeProject {
	p, ok := f.projects[key]
	if !ok {
		p = &obsFakeProject{rules: map[string]*obsFakeRule{}, order: map[string][]string{}, limits: map[string]*int64{}, version: 1}
		f.projects[key] = p
	}
	return p
}

// uiCreateRule creates a rule the way the LaunchDarkly UI would: outside
// Terraform, appended to its product's order.
func (f *obsFake) uiCreateRule(projectKey string, r obsFakeRule) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := f.project(projectKey)
	now := time.Now().UTC().Format(time.RFC3339)
	r.Version, r.CreatedAt, r.UpdatedAt = 1, now, now
	if r.Attributes == nil {
		r.Attributes = []string{}
	}
	p.rules[r.Key] = &r
	p.order[r.ProductType] = append(p.order[r.ProductType], r.Key)
	p.version++
}

func (f *obsFake) uiDeleteRule(projectKey, key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := f.project(projectKey)
	if r, ok := p.rules[key]; ok {
		p.order[r.ProductType] = obsFakeRemove(p.order[r.ProductType], key)
		delete(p.rules, key)
		p.version++
	}
}

func (f *obsFake) snapshot(projectKey string) (rules map[string]obsFakeRule, order map[string][]string, limits map[string]*int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := f.project(projectKey)
	rules = map[string]obsFakeRule{}
	for k, r := range p.rules {
		rules[k] = *r
	}
	order = map[string][]string{}
	for k, v := range p.order {
		order[k] = append([]string{}, v...)
	}
	limits = map[string]*int64{}
	for k, v := range p.limits {
		limits[k] = v
	}
	return rules, order, limits
}

func obsFakeRemove(list []string, key string) []string {
	out := []string{}
	for _, k := range list {
		if k != key {
			out = append(out, k)
		}
	}
	return out
}

type obsFakeResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *obsFakeResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (f *obsFake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rw := &obsFakeResponseWriter{ResponseWriter: w, status: http.StatusOK}
	body, _ := io.ReadAll(r.Body)
	f.serve(rw, r, body)
	if f.logf != nil {
		line := fmt.Sprintf("%s %s -> %d", r.Method, r.URL.RequestURI(), rw.status)
		if len(body) > 0 {
			line += "  body=" + string(body)
		}
		f.logf("%s", line)
	}
}

func obsFakeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": msg})
}

func obsFakeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (f *obsFake) settingsRep(p *obsFakeProject) map[string]interface{} {
	limits := map[string]*int64{}
	order := map[string][]string{}
	for _, prod := range obsFakeProducts {
		limits[prod] = p.limits[prod]
		order[prod] = append([]string{}, p.order[prod]...)
	}
	return map[string]interface{}{"version": p.version, "minuteRateLimits": limits, "ruleOrder": order}
}

func obsFakeValidProduct(p string) bool {
	for _, x := range obsFakeProducts {
		if x == p {
			return true
		}
	}
	return false
}

func obsFakeValidateRule(productType, name, operation string, rate float64) string {
	switch {
	case !obsFakeValidProduct(productType):
		return fmt.Sprintf("invalid productType %q", productType)
	case name == "":
		return "name is required"
	case operation != "include" && operation != "exclude":
		return fmt.Sprintf("invalid operation %q", operation)
	case rate < 0 || rate > 1:
		return "rate must be between 0 and 1"
	case productType == "metrics" && rate != 1:
		return "metrics rules must use rate 1"
	}
	return ""
}

func (f *obsFake) serve(w http.ResponseWriter, r *http.Request, body []byte) {
	if r.Header.Get("Authorization") == "" {
		obsFakeError(w, http.StatusUnauthorized, "unauthorized", "missing Authorization header")
		return
	}
	if r.Header.Get("LD-API-Version") != "beta" {
		// gonfalon's beta gate answers 403, not 400.
		obsFakeError(w, http.StatusForbidden, "forbidden", "this endpoint requires LD-API-Version: beta")
		return
	}
	// /api/v2/projects/{projectKey}/observability/{filter-settings|filter-rules}[/{ruleKey}]
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v2/"), "/")
	if len(parts) < 4 || parts[0] != "projects" || parts[2] != "observability" {
		obsFakeError(w, http.StatusNotFound, "not_found", "unknown route")
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	projectKey := parts[1]
	if f.missingProjects[projectKey] {
		obsFakeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("project %q not found", projectKey))
		return
	}
	p := f.project(projectKey)
	switch {
	case parts[3] == "filter-settings" && len(parts) == 4:
		f.serveSettings(w, r, body, p)
	case parts[3] == "filter-rules" && len(parts) == 4:
		f.serveRules(w, r, body, p)
	case parts[3] == "filter-rules" && len(parts) == 5:
		f.serveRule(w, r, body, p, parts[4])
	default:
		obsFakeError(w, http.StatusNotFound, "not_found", "unknown route")
	}
}

func (f *obsFake) serveSettings(w http.ResponseWriter, r *http.Request, body []byte, p *obsFakeProject) {
	switch r.Method {
	case http.MethodGet:
		obsFakeJSON(w, http.StatusOK, f.settingsRep(p))
	case http.MethodPut:
		var in struct {
			Version          *int64              `json:"version"`
			MinuteRateLimits map[string]*int64   `json:"minuteRateLimits"`
			RuleOrder        map[string][]string `json:"ruleOrder"`
		}
		if err := json.Unmarshal(body, &in); err != nil {
			obsFakeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		if in.Version != nil && *in.Version != p.version {
			obsFakeError(w, http.StatusConflict, "optimistic_locking_error", fmt.Sprintf("settings version is %d, request has %d", p.version, *in.Version))
			return
		}
		for prod, v := range in.MinuteRateLimits {
			if !obsFakeValidProduct(prod) {
				obsFakeError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("unknown product %q", prod))
				return
			}
			if v != nil && *v < 0 {
				obsFakeError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("minuteRateLimits.%s must be >= 0", prod))
				return
			}
		}
		for prod, keys := range in.RuleOrder {
			if !obsFakeValidProduct(prod) {
				obsFakeError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("unknown product %q", prod))
				return
			}
			seen := map[string]bool{}
			for _, k := range keys {
				rule, ok := p.rules[k]
				switch {
				case !ok:
					obsFakeError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("ruleOrder.%s: unknown rule key %q", prod, k))
					return
				case rule.ProductType != prod:
					obsFakeError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("ruleOrder.%s: rule %q belongs to %s", prod, k, rule.ProductType))
					return
				case seen[k]:
					obsFakeError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("ruleOrder.%s: duplicate rule key %q", prod, k))
					return
				}
				seen[k] = true
			}
		}
		p.limits = map[string]*int64{}
		for prod, v := range in.MinuteRateLimits {
			p.limits[prod] = v
		}
		for prod, keys := range in.RuleOrder {
			p.order[prod] = append([]string{}, keys...)
		}
		p.version++
		obsFakeJSON(w, http.StatusOK, f.settingsRep(p))
	default:
		obsFakeError(w, http.StatusMethodNotAllowed, "method_not_allowed", r.Method)
	}
}

func (f *obsFake) serveRules(w http.ResponseWriter, r *http.Request, body []byte, p *obsFakeProject) {
	switch r.Method {
	case http.MethodGet:
		productType := r.URL.Query().Get("productType")
		if productType != "" && !obsFakeValidProduct(productType) {
			obsFakeError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("invalid productType %q", productType))
			return
		}
		keys := make([]string, 0, len(p.rules))
		for k, rule := range p.rules {
			if productType == "" || rule.ProductType == productType {
				keys = append(keys, k)
			}
		}
		// Grouped by product (sessions, errors, logs, traces, metrics), then by key.
		rank := map[string]int{}
		for i, prod := range obsFakeProducts {
			rank[prod] = i
		}
		sort.Slice(keys, func(i, j int) bool {
			ri, rj := rank[p.rules[keys[i]].ProductType], rank[p.rules[keys[j]].ProductType]
			if ri != rj {
				return ri < rj
			}
			return keys[i] < keys[j]
		})
		items := make([]*obsFakeRule, 0, len(keys))
		for _, k := range keys {
			items = append(items, p.rules[k])
		}
		obsFakeJSON(w, http.StatusOK, map[string]interface{}{"items": items})
	case http.MethodPost:
		var in obsFakeRule
		if err := json.Unmarshal(body, &in); err != nil {
			obsFakeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		if !obsFakeKeyPattern.MatchString(in.Key) {
			obsFakeError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("invalid key %q", in.Key))
			return
		}
		if msg := obsFakeValidateRule(in.ProductType, in.Name, in.Operation, in.Rate); msg != "" {
			obsFakeError(w, http.StatusBadRequest, "invalid_request", msg)
			return
		}
		if _, taken := p.rules[in.Key]; taken {
			obsFakeError(w, http.StatusConflict, "conflict", fmt.Sprintf("a filter rule with key %q already exists", in.Key))
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)
		in.Version, in.CreatedAt, in.UpdatedAt = 1, now, now
		if in.Attributes == nil {
			in.Attributes = []string{}
		}
		p.rules[in.Key] = &in
		p.order[in.ProductType] = append(p.order[in.ProductType], in.Key)
		p.version++
		obsFakeJSON(w, http.StatusCreated, in)
	default:
		obsFakeError(w, http.StatusMethodNotAllowed, "method_not_allowed", r.Method)
	}
}

func (f *obsFake) serveRule(w http.ResponseWriter, r *http.Request, body []byte, p *obsFakeProject, key string) {
	rule, ok := p.rules[key]
	if !ok {
		obsFakeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("filter rule %q not found", key))
		return
	}
	switch r.Method {
	case http.MethodGet:
		obsFakeJSON(w, http.StatusOK, rule)
	case http.MethodPut:
		if f.beforeRulePut != nil {
			hook := f.beforeRulePut
			f.beforeRulePut = nil
			hook(p, key)
		}
		var in struct {
			Key         *string  `json:"key"`
			ProductType *string  `json:"productType"`
			Name        string   `json:"name"`
			Enabled     bool     `json:"enabled"`
			Query       string   `json:"query"`
			Operation   string   `json:"operation"`
			Rate        float64  `json:"rate"`
			Attributes  []string `json:"attributes"`
			Version     *int64   `json:"version"`
		}
		if err := json.Unmarshal(body, &in); err != nil {
			obsFakeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		if (in.Key != nil && *in.Key != rule.Key) || (in.ProductType != nil && *in.ProductType != rule.ProductType) {
			obsFakeError(w, http.StatusBadRequest, "invalid_request", "key and productType are immutable")
			return
		}
		if in.Version != nil && *in.Version != rule.Version {
			obsFakeError(w, http.StatusConflict, "optimistic_locking_error", fmt.Sprintf("rule %q is at version %d, request has %d", key, rule.Version, *in.Version))
			return
		}
		if msg := obsFakeValidateRule(rule.ProductType, in.Name, in.Operation, in.Rate); msg != "" {
			obsFakeError(w, http.StatusBadRequest, "invalid_request", msg)
			return
		}
		if in.Attributes == nil {
			in.Attributes = []string{}
		}
		rule.Name, rule.Enabled, rule.Query, rule.Operation, rule.Rate, rule.Attributes = in.Name, in.Enabled, in.Query, in.Operation, in.Rate, in.Attributes
		rule.Version++
		rule.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		obsFakeJSON(w, http.StatusOK, rule)
	case http.MethodDelete:
		p.order[rule.ProductType] = obsFakeRemove(p.order[rule.ProductType], key)
		delete(p.rules, key)
		p.version++
		w.WriteHeader(http.StatusNoContent)
	default:
		obsFakeError(w, http.StatusMethodNotAllowed, "method_not_allowed", r.Method)
	}
}

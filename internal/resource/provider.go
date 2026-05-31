package resource

import (
	"os"
	"sort"

	"github.com/rotemmiz/forge/internal/credstore"
	"github.com/rotemmiz/forge/internal/engine/catalog"
)

// Provider is the wire shape served in GET /provider's `all` array (openapi
// Provider). Env, Options, and Models are always serialized (required).
type Provider struct {
	ID      string                   `json:"id"`
	Name    string                   `json:"name"`
	Source  string                   `json:"source"`
	Env     []string                 `json:"env"`
	Options map[string]any           `json:"options"`
	Models  map[string]catalog.Model `json:"models"`
}

// ProviderList is the GET /provider response (provider/provider.ts ListResult).
type ProviderList struct {
	All       []Provider        `json:"all"`
	Default   map[string]string `json:"default"`
	Connected []string          `json:"connected"`
}

// BuildProviderList assembles the /provider response from the models.dev catalog,
// the merged config (disabled_providers/enabled_providers/provider overlay), and
// credential detection (env vars + opencode's auth.json). It mirrors opencode's
// provider list handler: `all` is every (filtered) catalog provider, `connected`
// is the subset with resolvable credentials, `default` maps each provider to its
// lowest-id model.
func BuildProviderList(cat catalog.Catalog, cfg map[string]any) ProviderList {
	disabled := stringSet(cfg["disabled_providers"])
	enabled := stringSet(cfg["enabled_providers"])
	hasEnabled := enabled != nil

	auth := credstore.Load()
	configProviders := configProviderKeys(cfg)

	result := ProviderList{All: []Provider{}, Default: map[string]string{}, Connected: []string{}}
	connectedSet := map[string]bool{}

	for id, p := range cat {
		if disabled[id] || (hasEnabled && !enabled[id]) {
			continue
		}
		source := "env"
		conn := false
		if envAny(p.Env) {
			conn = true
		}
		if _, ok := auth[id]; ok {
			conn, source = true, authSource(auth[id])
		}
		if configProviders[id] {
			conn, source = true, "config"
		}

		result.All = append(result.All, Provider{
			ID: id, Name: p.Name, Source: source,
			Env: nonNil(p.Env), Options: map[string]any{}, Models: p.Models,
		})
		if def := lowestModelID(p.Models); def != "" {
			result.Default[id] = def
		}
		if conn {
			connectedSet[id] = true
		}
	}

	sort.Slice(result.All, func(i, j int) bool { return result.All[i].ID < result.All[j].ID })
	for id := range connectedSet {
		result.Connected = append(result.Connected, id)
	}
	sort.Strings(result.Connected)
	return result
}

// authSource maps a stored credential's type to the provider's source label.
func authSource(r credstore.Record) string {
	if credstore.TypeOf(r) == "api" {
		return "api"
	}
	return "env"
}

// envAny reports whether any of the named env vars is set (non-empty).
func envAny(names []string) bool {
	for _, n := range names {
		if os.Getenv(n) != "" {
			return true
		}
	}
	return false
}

// lowestModelID returns the alphabetically-first model id (opencode's default
// model per provider; provider.ts defaultModelIDs).
func lowestModelID(models map[string]catalog.Model) string {
	ids := make([]string, 0, len(models))
	for id := range models {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return ""
	}
	sort.Strings(ids)
	return ids[0]
}

// configProviderKeys returns the provider IDs declared in the config `provider`
// map (those are configured/connected regardless of env).
func configProviderKeys(cfg map[string]any) map[string]bool {
	out := map[string]bool{}
	if raw, ok := cfg["provider"].(map[string]any); ok {
		for id := range raw {
			out[id] = true
		}
	}
	return out
}

// stringSet converts a config string array (any) to a set; nil when absent.
func stringSet(v any) map[string]bool {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	set := map[string]bool{}
	for _, item := range arr {
		if s, ok := item.(string); ok {
			set[s] = true
		}
	}
	return set
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

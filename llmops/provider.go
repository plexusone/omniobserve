package llmops

import (
	"fmt"
	"sort"
	"sync"
)

var (
	providersMu sync.RWMutex
	providers   = make(map[string]ProviderFactory)
)

// ProviderFactory creates a new Provider instance with the given options.
type ProviderFactory func(opts ...ClientOption) (Provider, error)

// Register makes a provider available by the provided name.
// If Register is called twice with the same name or if factory is nil,
// it panics.
func Register(name string, factory ProviderFactory) {
	providersMu.Lock()
	defer providersMu.Unlock()

	if factory == nil {
		panic("llmops: Register factory is nil")
	}
	if _, dup := providers[name]; dup {
		panic("llmops: Register called twice for provider " + name)
	}
	providers[name] = factory
}

// Providers returns a sorted list of the names of the registered providers.
func Providers() []string {
	providersMu.RLock()
	defer providersMu.RUnlock()

	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Open opens a provider specified by its name.
//
// Most users will use a specific provider package import like:
//
//	import _ "github.com/plexusone/omniobserve/llmops/opik"
//
// And then open it with:
//
//	provider, err := llmops.Open("opik", llmops.WithAPIKey("..."))
func Open(name string, opts ...ClientOption) (Provider, error) {
	providersMu.RLock()
	factory, ok := providers[name]
	providersMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("llmops: unknown provider %q (forgotten import?)", name)
	}
	return factory(opts...)
}

// MustOpen is like Open but panics on error.
func MustOpen(name string, opts ...ClientOption) Provider {
	provider, err := Open(name, opts...)
	if err != nil {
		panic(err)
	}
	return provider
}

// ProviderInfo contains metadata about a registered provider.
type ProviderInfo struct {
	Name         string
	Description  string
	Website      string
	OpenSource   bool
	SelfHosted   bool
	Capabilities []Capability
}

var providerInfos = make(map[string]ProviderInfo)

// RegisterInfo registers metadata about a provider.
// This is optional and used for discovery/documentation.
func RegisterInfo(info ProviderInfo) {
	providersMu.Lock()
	defer providersMu.Unlock()
	providerInfos[info.Name] = info
}

// GetProviderInfo returns metadata about a registered provider.
func GetProviderInfo(name string) (ProviderInfo, bool) {
	providersMu.RLock()
	defer providersMu.RUnlock()
	info, ok := providerInfos[name]
	return info, ok
}

// AllProviderInfo returns metadata for all registered providers.
func AllProviderInfo() []ProviderInfo {
	providersMu.RLock()
	defer providersMu.RUnlock()

	infos := make([]ProviderInfo, 0, len(providerInfos))
	for _, info := range providerInfos {
		infos = append(infos, info)
	}
	return infos
}

// Unregister removes a provider from the registry.
// This is primarily useful for testing.
func Unregister(name string) {
	providersMu.Lock()
	defer providersMu.Unlock()
	delete(providers, name)
	delete(providerInfos, name)
}

// UnregisterAll removes all providers from the registry.
// This is primarily useful for testing.
func UnregisterAll() {
	providersMu.Lock()
	defer providersMu.Unlock()
	providers = make(map[string]ProviderFactory)
	providerInfos = make(map[string]ProviderInfo)
}

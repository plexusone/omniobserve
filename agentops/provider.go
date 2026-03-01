package agentops

import (
	"fmt"
	"sort"
	"sync"
)

var (
	providersMu sync.RWMutex
	providers   = make(map[string]StoreFactory)
)

// StoreFactory creates a new Store instance with the given options.
type StoreFactory func(opts ...ClientOption) (Store, error)

// Register makes a store provider available by the provided name.
// If Register is called twice with the same name or if factory is nil,
// it panics.
func Register(name string, factory StoreFactory) {
	providersMu.Lock()
	defer providersMu.Unlock()

	if factory == nil {
		panic("agentops: Register factory is nil")
	}
	if _, dup := providers[name]; dup {
		panic("agentops: Register called twice for provider " + name)
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

// Open opens a store specified by its provider name.
//
// Most users will use a specific provider package import like:
//
//	import _ "github.com/plexusone/omniobserve/agentops/postgres"
//
// And then open it with:
//
//	store, err := agentops.Open("postgres",
//		agentops.WithDSN("postgres://user:pass@localhost/db"),
//	)
func Open(name string, opts ...ClientOption) (Store, error) {
	providersMu.RLock()
	factory, ok := providers[name]
	providersMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("agentops: unknown provider %q (forgotten import?)", name)
	}
	return factory(opts...)
}

// MustOpen is like Open but panics on error.
func MustOpen(name string, opts ...ClientOption) Store {
	store, err := Open(name, opts...)
	if err != nil {
		panic(err)
	}
	return store
}

// ProviderInfo contains metadata about a registered provider.
type ProviderInfo struct {
	Name        string
	Description string
	Features    []string
}

var providerInfos = make(map[string]ProviderInfo)

// RegisterInfo registers metadata about a provider.
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

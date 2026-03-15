# OmniObserve Tasks

## Unified slog + Observability Implementation

Reference: [TRD](docs/design/TRD_UNIFIED_SLOG_OBSERVABILITY.md) | [Plan](docs/design/PLAN_UNIFIED_OBSERVABILITY.md)

### Phase 1: sloghandler Package Foundation
- [x] Create `sloghandler/handler.go` - core slog.Handler implementation
- [x] Create `sloghandler/trace.go` - trace context extraction
- [x] Create `sloghandler/options.go` - configuration options
- [x] Create `sloghandler/fanout.go` - multi-destination handler
- [x] Create `sloghandler/handler_test.go` - tests and benchmarks

### Phase 2: observops Provider Integration
- [x] Add `SlogHandler()` to `observops.Provider` interface
- [x] Add `SlogOption` and `SlogConfig` types
- [x] Create `observops/slog.go` - LoggerSlogHandler adapter
- [x] Update OTLP provider with SlogHandler
- [x] Update Datadog provider with SlogHandler
- [x] Update New Relic provider with SlogHandler
- [ ] Integration tests with OTLP collector

### Phase 3: Additional Backend Providers
- [x] Create `observops/dynatrace/dynatrace.go`
- [x] Create `observops/dynatrace/noop.go`
- [ ] Create `observops/prometheus/prometheus.go` (metrics-only)
- [x] Provider documentation in `docs/providers/`
  - [x] `docs/providers/README.md`
  - [x] `docs/providers/otlp.md`
  - [x] `docs/providers/datadog.md`
  - [x] `docs/providers/dynatrace.md`
  - [x] `docs/providers/newrelic.md`
  - [x] `docs/providers/sloghandler.md`

### Phase 4: Unified Entry Point
- [ ] Create `omniobserve/unified.go`
- [ ] Create `omniobserve/options.go`
- [ ] Context utilities (ContextWithLogger, LoggerFromContext)
- [ ] HTTP middleware (Chi/Echo/Gin adapters)
- [ ] Example applications

### Phase 5: llmops + observops Correlation
- [ ] Context bridge between llmops and observops
- [ ] Unified span types
- [ ] Combined provider

---

## Status

**Phases 1-3 Complete** - Core slog + observability integration is ready for use.

Remaining phases (4-5) are optional enhancements that can be done later:

- Phase 4: Convenience APIs and middleware
- Phase 5: LLM-specific correlation (only needed for LLM applications)

---

## Deferred Tasks

The following are not blocking and can be done as needed:

- [ ] Integration tests with OTLP collector (Phase 2)
- [ ] Prometheus provider - metrics-only (Phase 3)
- [ ] Unified entry point API (Phase 4)
- [ ] HTTP middleware adapters (Phase 4)
- [ ] llmops + observops correlation (Phase 5)

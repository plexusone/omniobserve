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
- [x] Create `omniobserve/unified.go`
- [x] Create `omniobserve/options.go`
- [x] Context utilities (ContextWithLogger, LoggerFromContext)
- [x] HTTP middleware (standard http.Handler)
- [x] Example applications

### Phase 5: llmops + observops Correlation
- [ ] Context bridge between llmops and observops
- [ ] Unified span types
- [ ] Combined provider

---

## Status

**Phases 1-4 Complete** - Unified observability with HTTP middleware is ready for use.

Remaining phase (5) is an optional enhancement:

- Phase 5: LLM-specific correlation (only needed for LLM applications)

---

## Deferred Tasks

The following are not blocking and can be done as needed:

- [ ] Integration tests with OTLP collector (Phase 2)
- [ ] Prometheus provider - metrics-only (Phase 3)
- [ ] Chi/Echo/Gin-specific middleware adapters (Phase 4 enhancement)
- [ ] llmops + observops correlation (Phase 5)

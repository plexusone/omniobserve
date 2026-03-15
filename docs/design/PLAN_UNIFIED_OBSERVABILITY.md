# Implementation Plan: Unified slog + Observability

**Related TRD**: [TRD_UNIFIED_SLOG_OBSERVABILITY.md](./TRD_UNIFIED_SLOG_OBSERVABILITY.md)
**Target Version**: v0.3.0
**Date**: 2026-03-03

## Overview

This plan outlines the implementation phases for integrating Go's `*slog.Logger` with OmniObserve's observability stack, enabling unified logging with automatic trace correlation and multi-backend export.

---

## Phase 1: sloghandler Package Foundation

**Goal**: Create core slog.Handler implementation with trace correlation

### Tasks

- [ ] **1.1** Create `sloghandler/handler.go`
  - Implement `slog.Handler` interface
  - Support local + remote output
  - Configure minimum levels per destination
  - Handle `WithAttrs()` and `WithGroup()`

- [ ] **1.2** Create `sloghandler/trace.go`
  - Extract trace context from `context.Context`
  - Support both observops.Span and OTel span
  - Add `trace_id` and `span_id` to log records

- [ ] **1.3** Create `sloghandler/options.go`
  - `WithLocalHandler(slog.Handler)`
  - `WithRemoteLevel(slog.Level)`
  - `WithoutTraceContext()`
  - `WithAttributeProcessor(...)`

- [ ] **1.4** Create `sloghandler/fanout.go`
  - Fan-out handler for multiple destinations
  - Async option for non-blocking writes
  - Error handling strategy (continue on error)

- [ ] **1.5** Create `sloghandler/handler_test.go`
  - Test level filtering
  - Test trace context extraction
  - Test WithAttrs/WithGroup chaining
  - Benchmark handler overhead

### Deliverables
- `sloghandler` package with full test coverage
- Benchmark showing <100ns overhead per log call

---

## Phase 2: observops Provider Integration

**Goal**: Add slog integration to observops.Provider interface

### Tasks

- [ ] **2.1** Update `observops/observops.go`
  - Add `SlogHandler(opts ...SlogOption) slog.Handler` to Provider interface
  - Add `Logger(opts ...SlogOption) *slog.Logger` convenience method
  - Define SlogOption type and common options

- [ ] **2.2** Update `observops/otlp/otlp.go`
  - Implement `SlogHandler()` using internal Logger
  - Wire up trace context from OTel SDK
  - Test with OTel Collector

- [ ] **2.3** Create `observops/noop/noop.go`
  - No-op implementations for disabled mode
  - `NoopSlogHandler()` returns `slog.DiscardHandler`

- [ ] **2.4** Integration tests
  - End-to-end test with OTLP collector (docker-compose)
  - Verify trace_id correlation in exported logs
  - Test multi-level scenarios

### Deliverables
- Updated `observops.Provider` interface
- OTLP provider with slog integration
- Integration test suite

---

## Phase 3: Additional Backend Providers

**Goal**: Implement Dynatrace, complete Datadog, add Prometheus

### Tasks

- [ ] **3.1** Complete `observops/datadog/datadog.go`
  - Implement full Provider interface
  - Add Datadog-specific configuration (site, API key)
  - Test with Datadog agent

- [ ] **3.2** Create `observops/dynatrace/dynatrace.go`
  - Implement Provider wrapping OTLP
  - Tenant/environment configuration
  - Region selection (US, EU, Managed)
  - Test with Dynatrace

- [ ] **3.3** Create `observops/prometheus/prometheus.go`
  - Metrics-only provider
  - HTTP handler for `/metrics` endpoint
  - No-op for Tracer and Logger

- [ ] **3.4** Documentation
  - Provider setup guides in `docs/providers/`
  - Configuration reference
  - Troubleshooting common issues

### Deliverables
- Three additional providers (Datadog, Dynatrace, Prometheus)
- Provider documentation

---

## Phase 4: Unified Entry Point

**Goal**: Create high-level API combining observops + llmops + slog

### Tasks

- [ ] **4.1** Create `omniobserve/unified.go`
  - `UnifiedProvider` struct combining capabilities
  - `Open(name, opts...)` for single backend
  - `OpenMulti(backends...)` for fan-out

- [ ] **4.2** Create `omniobserve/options.go`
  - Unified configuration options
  - Backend-specific option helpers
  - Slog integration options

- [ ] **4.3** Context utilities
  - `ContextWithLogger(ctx, *slog.Logger)`
  - `LoggerFromContext(ctx, fallback)`
  - Consider re-exporting from mogo/slogutil

- [ ] **4.4** HTTP middleware
  - Request tracing middleware
  - Logger injection middleware
  - Chi/Echo/Gin adapters

- [ ] **4.5** Examples
  - Basic usage example
  - HTTP service example
  - Multi-backend example
  - LLM + APM combined example

### Deliverables
- Unified `omniobserve.Open()` API
- HTTP middleware
- Example applications

---

## Phase 5: llmops + observops Correlation

**Goal**: Enable correlation between LLM traces and service traces

### Tasks

- [ ] **5.1** Context bridge
  - Share trace context between llmops and observops
  - Option to link llmops trace as child of observops span
  - Automatic span linking

- [ ] **5.2** Unified span types
  - Map llmops.SpanType to observops semantic conventions
  - Add LLM-specific attributes to observops spans
  - Support agent semantic conventions (semconv)

- [ ] **5.3** Combined provider
  - Single provider for both llmops and observops
  - Unified configuration
  - Coherent shutdown/flush

### Deliverables
- Correlated traces across LLM and service layers
- Combined provider option

---

## File Structure (Final)

```
omniobserve/
├── sloghandler/
│   ├── handler.go
│   ├── handler_test.go
│   ├── trace.go
│   ├── options.go
│   └── fanout.go
├── observops/
│   ├── observops.go          # Updated with SlogHandler
│   ├── otlp/
│   │   └── otlp.go           # Updated
│   ├── datadog/
│   │   └── datadog.go        # Completed
│   ├── dynatrace/
│   │   └── dynatrace.go      # New
│   └── prometheus/
│       └── prometheus.go     # New
├── llmops/
│   └── (existing)
├── unified.go                # New unified entry
├── options.go                # Unified options
├── middleware/
│   ├── http.go
│   ├── chi.go
│   └── echo.go
└── examples/
    ├── basic/
    ├── http-service/
    ├── multi-backend/
    └── llm-with-apm/
```

---

## Success Criteria

### Functional
- [ ] slog.Logger works with trace correlation
- [ ] Logs appear in OTLP collector with trace_id
- [ ] Multiple backends receive logs simultaneously
- [ ] Context-based logger injection works

### Performance
- [ ] <100ns overhead per log call (vs plain slog)
- [ ] <1MB additional memory for handler state
- [ ] Async fan-out doesn't block application

### Quality
- [ ] >80% test coverage for new packages
- [ ] All providers have integration tests
- [ ] Documentation for each provider

---

## Timeline

| Phase | Duration | Dependencies |
|-------|----------|--------------|
| Phase 1 | 1 week | None |
| Phase 2 | 1 week | Phase 1 |
| Phase 3 | 2 weeks | Phase 2 |
| Phase 4 | 1 week | Phase 2 |
| Phase 5 | 1 week | Phase 4 |

**Total**: ~6 weeks

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| OTel SDK breaking changes | High | Pin specific versions, test with multiple |
| Performance regression | Medium | Benchmark suite, profile hot paths |
| Vendor API changes | Medium | Abstract behind interfaces, version tests |
| slog API changes | Low | Go 1.21+ has stable slog |

---

## Next Steps

1. Review and approve this plan
2. Create GitHub issues for Phase 1 tasks
3. Set up CI for new packages
4. Begin Phase 1 implementation

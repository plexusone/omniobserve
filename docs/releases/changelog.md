# Changelog

All notable changes to OmniObserve are documented here.

This changelog follows [Semantic Versioning](https://semver.org/) and is generated from [CHANGELOG.json](https://github.com/plexusone/omniobserve/blob/main/CHANGELOG.json) using [schangelog](https://github.com/grokify/structured-changelog).

## [v0.8.0](v0.8.0.md) - 2026-03

### Highlights

- New `sloghandler` package with dual output and trace context injection
- Dynatrace provider for full Dynatrace observability via OTLP
- SlogHandler integration for all observops providers
- Observability specs package with OpenSLO and RED metrics support

### Added

- `sloghandler` package with dual output (local + remote) and automatic trace context injection
- `observops/dynatrace` provider for Dynatrace via OTLP/HTTP
- `SlogHandler()` method on OTLP, Datadog, New Relic, and Dynatrace providers
- `SlogOption` and `SlogConfig` types for slog handler configuration
- `specs` package with OpenSLO, RED metrics, and service class templates
- `cmd/genspecs` CLI for JSON Schema generation from Go types
- Provider documentation in `docs/providers/`

### Dependencies

- Added OTLP HTTP exporters for metrics and traces
- Added OpenSLO Go SDK
- Added jsonschema for schema generation
- Updated Go to 1.25.5

## [v0.7.0](v0.7.0.md) - 2026-03-01

### Highlights

- Organization rename from agentplexus to plexusone

### Changed

- **BREAKING**: Module path changed from `github.com/agentplexus/omniobserve` to `github.com/plexusone/omniobserve`

### Dependencies

- Upgraded `github.com/plexusone/omnillm` to v0.13.0
- Upgraded `github.com/plexusone/structured-evaluation` to v0.3.0

## [v0.6.0](v0.6.0.md) - 2026-02-22

### Highlights

- Built-in slog provider for local trace logging during development and debugging
- Structured evaluation integration for connecting llmops with sevaluation workflows

### Added

- `llmops/slog` provider for local structured logging of trace events
- `WithLogger` option in `ClientOptions` for custom slog.Logger configuration
- `integrations/sevaluation` package for structured-evaluation integration

### Fixed

- gosec warnings for APIKey struct fields (G117) and HTTP client SSRF (G704)

### Documentation

- Observability integration guide for stats-agent-team
- README updated with slog provider in supported providers table

## [v0.5.1](https://github.com/plexusone/omniobserve/releases/tag/v0.5.1) - 2026-01-19

### Highlights

- Compatibility update for omnillm v0.11.0 with new multi-provider configuration API

### Changed

- Updated `ClientConfig` usage to new `Providers` slice API for omnillm v0.11.0 compatibility

### Dependencies

- Upgraded `github.com/plexusone/omnillm` from v0.10.0 to v0.11.0

## [v0.5.0](v0.5.0.md) - 2026-01-03

### Highlights

- Comprehensive evaluation metrics for assessing LLM outputs with both code-based and LLM-based metrics
- Streamlined provider architecture with Opik and Phoenix adapters moved to standalone SDKs

### Added

- `AnnotationManager` interface for span/trace annotations with `CreateAnnotation` and `ListAnnotations` methods
- `DatasetManager` methods: `GetDatasetByID` and `DeleteDataset`
- Prompt model/provider options: `WithPromptModel`, `WithPromptProvider`, `ModelName`, `ModelProvider` fields
- OmniLLM hook auto-creates traces when none exists in context
- Trace context helpers: `contextWithTrace` and `traceFromContext`
- `llmops/metrics` package with LLM-based metrics: `HallucinationMetric`, `RelevanceMetric`, `QACorrectnessMetric`, `ToxicityMetric`
- `llmops/metrics` package with code-based metrics: `ExactMatchMetric`, `RegexMetric`, `ContainsMetric`
- `examples/evaluation` demonstrating metrics usage

### Changed

- **BREAKING**: Provider adapters moved to standalone SDKs: Opik to `github.com/agentplexus/go-opik/llmops`, Phoenix to `github.com/agentplexus/go-phoenix/llmops`

### Removed

- `llmops/opik` adapter (moved to go-opik)
- `llmops/phoenix` adapter (moved to go-phoenix)
- `sdk/phoenix` package (use go-phoenix directly)

## [v0.4.0](https://github.com/plexusone/omniobserve/releases/tag/v0.4.0) - 2025-12-27

### Highlights

- Full-stack observability with new agentops and observops packages for monitoring agentic AI systems
- Semantic conventions aligned with OpenTelemetry for standardized telemetry across LLM applications

### Added

- `agentops` package for agent operations monitoring
- `observops` package for unified observability operations
- `semconv` package with semantic conventions for agentic AI

### Changed

- `llmops` refactored to use `github.com/plexusone/omnillm`

## [v0.3.0](https://github.com/plexusone/omniobserve/releases/tag/v0.3.0) - 2025-12-21

### Highlights

- Project renamed from observai to omniobserve for consistency with omnillm ecosystem naming

### Changed

- **BREAKING**: Project renamed from `observai` to `omniobserve` for consistency with `omnillm` naming conventions

## [v0.2.0](https://github.com/plexusone/omniobserve/releases/tag/v0.2.0) - 2025-12-21

### Highlights

- FluxLLM integration enables observability for flux-based LLM workflows

### Added

- `integrations/fluxllm` package for FluxLLM observability

## [v0.1.0](https://github.com/plexusone/omniobserve/releases/tag/v0.1.0) - 2025-12-20

### Highlights

- Initial release providing unified observability primitives for LLM applications with OpenTelemetry integration

### Added

- Core observability framework for LLM applications
- GitHub Actions CI workflow
- GitHub Pages presentation
- MIT License

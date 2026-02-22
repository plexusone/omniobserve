# Installation

## Requirements

- Go 1.21 or later

## Install Core Library

```bash
go get github.com/agentplexus/omniobserve
```

## Install Provider Adapters

Provider adapters are imported as blank imports to register themselves:

```bash
# Opik (external module)
go get github.com/agentplexus/go-opik

# Phoenix (external module)
go get github.com/agentplexus/go-phoenix

# Langfuse (included in omniobserve)
# No additional install needed

# slog (included in omniobserve)
# No additional install needed
```

## OmniLLM Integration (Optional)

To auto-instrument OmniLLM calls:

```bash
go get github.com/agentplexus/omniobserve/integrations/omnillm
```

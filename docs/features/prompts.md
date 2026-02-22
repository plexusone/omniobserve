# Prompts

OmniObserve supports prompt template management with versioning (provider-dependent).

## Creating Prompts

```go
prompt, err := provider.CreatePrompt(ctx, "chat-template",
    `You are a helpful assistant.

User: {{.query}}
Assistant:`,
    llmops.WithPromptDescription("Main chat template"),
    llmops.WithPromptModel("gpt-4"),
    llmops.WithPromptProvider("openai"),
)
```

## Retrieving Prompts

```go
// Get latest version
prompt, err := provider.GetPrompt(ctx, "chat-template")

// Get specific version
prompt, err := provider.GetPrompt(ctx, "chat-template", "v2")
```

## Rendering Prompts

```go
prompt, _ := provider.GetPrompt(ctx, "chat-template")

rendered := prompt.Render(map[string]any{
    "query": "What is the meaning of life?",
})

// Result:
// You are a helpful assistant.
//
// User: What is the meaning of life?
// Assistant:
```

## Listing Prompts

```go
prompts, err := provider.ListPrompts(ctx)

for _, p := range prompts {
    fmt.Printf("%s (v%s): %s\n", p.Name, p.Version, p.Description)
}
```

## Prompt Structure

```go
type Prompt struct {
    ID          string
    Name        string
    Template    string
    Description string
    Version     string
    Model       string
    Provider    string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

## Provider Support

| Provider | Prompts |
|----------|:-------:|
| Opik | :white_check_mark: Full |
| Langfuse | Partial |
| Phoenix | :x: |
| slog | :x: |

!!! note
    Prompt management features vary by provider. Opik has the most complete implementation.

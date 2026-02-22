# Datasets

OmniObserve provides dataset management for evaluation and testing.

## Creating Datasets

```go
dataset, err := provider.CreateDataset(ctx, "test-cases",
    llmops.WithDatasetDescription("Test cases for RAG evaluation"),
)
```

## Adding Items

```go
err := provider.AddDatasetItems(ctx, "test-cases", []llmops.DatasetItem{
    {
        Input:    map[string]any{"query": "What is Go?"},
        Expected: map[string]any{"answer": "Go is a programming language..."},
    },
    {
        Input:    map[string]any{"query": "What is Python?"},
        Expected: map[string]any{"answer": "Python is a programming language..."},
    },
})
```

## Retrieving Datasets

```go
// Get by name
dataset, err := provider.GetDataset(ctx, "test-cases")

// Get by ID
dataset, err := provider.GetDatasetByID(ctx, "dataset-uuid")

// List all datasets
datasets, err := provider.ListDatasets(ctx)
```

## Dataset Item Structure

```go
type DatasetItem struct {
    ID       string         // Auto-generated if empty
    Input    map[string]any // Input to the system
    Expected map[string]any // Expected output
    Metadata map[string]any // Additional metadata
}
```

## Deleting Datasets

```go
err := provider.DeleteDataset(ctx, "dataset-uuid")
```

## Provider Support

| Provider | Datasets |
|----------|:--------:|
| Opik | :white_check_mark: |
| Langfuse | :white_check_mark: |
| Phoenix | Partial |
| slog | :x: |

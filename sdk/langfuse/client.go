// Package langfuse provides a Go SDK for Langfuse, an open-source LLM observability platform.
//
// Langfuse uses a batch ingestion API and supports OpenTelemetry-compatible trace export.
//
// Usage:
//
//	client, err := langfuse.NewClient(
//		langfuse.WithPublicKey("pk-..."),
//		langfuse.WithSecretKey("sk-..."),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer client.Close()
//
//	ctx, trace, _ := client.StartTrace(ctx, "my-trace")
//	defer trace.End(ctx)
package langfuse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Version is the SDK version.
const Version = "0.1.0"

// Default endpoints
const (
	DefaultEndpoint = "https://cloud.langfuse.com"
	USEndpoint      = "https://us.cloud.langfuse.com"
	HIPAAEndpoint   = "https://hipaa.cloud.langfuse.com"
	LocalEndpoint   = "http://localhost:3000"
)

// Client is the main Langfuse client.
type Client struct {
	publicKey  string
	secretKey  string
	endpoint   string
	httpClient *http.Client
	timeout    time.Duration

	// Batching
	mu          sync.Mutex
	batch       []Event
	batchSize   int
	flushPeriod time.Duration
	stopCh      chan struct{}
	doneCh      chan struct{}

	// State
	disabled bool
	debug    bool
}

// NewClient creates a new Langfuse client.
func NewClient(opts ...Option) (*Client, error) {
	c := &Client{
		endpoint:    DefaultEndpoint,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		timeout:     30 * time.Second,
		batch:       make([]Event, 0, 100),
		batchSize:   100,
		flushPeriod: 5 * time.Second,
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.publicKey == "" {
		return nil, ErrMissingPublicKey
	}
	if c.secretKey == "" {
		return nil, ErrMissingSecretKey
	}

	// Start background flusher
	if !c.disabled {
		go c.backgroundFlusher()
	}

	return c, nil
}

// Close flushes pending events and closes the client.
func (c *Client) Close() error {
	if c.disabled {
		return nil
	}

	close(c.stopCh)
	<-c.doneCh

	// Final flush
	return c.Flush(context.Background())
}

// Flush sends all pending events to Langfuse.
func (c *Client) Flush(ctx context.Context) error {
	c.mu.Lock()
	if len(c.batch) == 0 {
		c.mu.Unlock()
		return nil
	}

	events := c.batch
	c.batch = make([]Event, 0, c.batchSize)
	c.mu.Unlock()

	return c.sendBatch(ctx, events)
}

// backgroundFlusher periodically flushes events.
func (c *Client) backgroundFlusher() {
	defer close(c.doneCh)

	ticker := time.NewTicker(c.flushPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = c.Flush(context.Background())
		case <-c.stopCh:
			return
		}
	}
}

// enqueue adds an event to the batch.
func (c *Client) enqueue(event Event) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.batch = append(c.batch, event)

	// Flush if batch is full
	if len(c.batch) >= c.batchSize {
		events := c.batch
		c.batch = make([]Event, 0, c.batchSize)
		go func() {
			_ = c.sendBatch(context.Background(), events)
		}()
	}
}

// sendBatch sends a batch of events to the ingestion API.
func (c *Client) sendBatch(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return nil
	}

	payload := BatchIngestionRequest{
		Batch: events,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal batch: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/api/public/ingestion", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "langfuse-go-sdk/"+Version)

	resp, err := c.httpClient.Do(req) //nolint:gosec // G704: URL is constructed from configured endpoint, not user input
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	return nil
}

// StartTrace creates a new trace.
func (c *Client) StartTrace(ctx context.Context, name string, opts ...TraceOption) (context.Context, *Trace, error) {
	if c.disabled {
		return ctx, &Trace{disabled: true}, nil
	}

	cfg := &traceConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	trace := &Trace{
		client:    c,
		id:        uuid.New().String(),
		name:      name,
		startTime: time.Now(),
		metadata:  cfg.metadata,
		tags:      cfg.tags,
		userId:    cfg.userId,
		sessionId: cfg.sessionId,
		input:     cfg.input,
	}

	// Enqueue trace creation event
	c.enqueue(Event{
		ID:        uuid.New().String(),
		Type:      EventTypeTraceCreate,
		Timestamp: time.Now(),
		Body: TraceBody{
			ID:        trace.id,
			Name:      name,
			Timestamp: trace.startTime,
			Metadata:  cfg.metadata,
			Tags:      cfg.tags,
			UserID:    cfg.userId,
			SessionID: cfg.sessionId,
			Input:     cfg.input,
			Public:    cfg.public,
		},
	})

	newCtx := ContextWithTrace(ctx, trace)
	newCtx = ContextWithClient(newCtx, c)
	return newCtx, trace, nil
}

// doRequest performs an HTTP request with authentication.
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, body)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "langfuse-go-sdk/"+Version)

	return c.httpClient.Do(req) //nolint:gosec // G704: URL is constructed from configured endpoint, not user input
}

// doGet performs a GET request.
func (c *Client) doGet(ctx context.Context, path string, result any) error {
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// doPost performs a POST request.
func (c *Client) doPost(ctx context.Context, path string, body, result any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}

	resp, err := c.doRequest(ctx, "POST", path, bodyReader)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

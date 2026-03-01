package omnillm

import (
	"io"
	"strings"

	"github.com/plexusone/omnillm"
	"github.com/plexusone/omnillm/provider"

	"github.com/plexusone/omniobserve/llmops"
)

// observedStream wraps a provider.ChatCompletionStream to capture
// streaming content and record it when the stream ends.
type observedStream struct {
	stream        provider.ChatCompletionStream
	span          llmops.Span
	trace         llmops.Trace // trace we created (may be nil)
	info          omnillm.LLMCallInfo
	contentBuffer strings.Builder
	ended         bool
}

// Recv receives the next chunk from the stream.
// It buffers the content and ends the span when the stream completes.
func (s *observedStream) Recv() (*provider.ChatCompletionChunk, error) {
	chunk, err := s.stream.Recv()

	if err == io.EOF {
		// Stream complete - finalize span
		s.finalizeSpan(nil)
		return chunk, err
	}

	if err != nil {
		s.finalizeSpan(err)
		return chunk, err
	}

	// Buffer content from chunk
	if chunk != nil && len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
		s.contentBuffer.WriteString(chunk.Choices[0].Delta.Content)
	}

	return chunk, nil
}

// Close closes the underlying stream.
func (s *observedStream) Close() error {
	// Ensure span is ended if Close is called before EOF
	s.finalizeSpan(nil)
	return s.stream.Close()
}

// finalizeSpan ends the span and trace with the buffered content.
func (s *observedStream) finalizeSpan(err error) {
	if s.ended {
		return
	}
	s.ended = true

	output := s.contentBuffer.String()

	// End span first
	if err != nil {
		_ = s.span.End(llmops.WithEndError(err))
	} else {
		// Set the buffered output
		if len(output) > 0 {
			_ = s.span.SetOutput(output)
		}
		_ = s.span.End()
	}

	// End trace if we created one
	if s.trace != nil {
		if len(output) > 0 {
			_ = s.trace.SetOutput(output)
		}
		_ = s.trace.End()
	}
}

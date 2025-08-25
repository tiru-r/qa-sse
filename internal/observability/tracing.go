package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

type Span struct {
	TraceID   string            `json:"trace_id"`
	SpanID    string            `json:"span_id"`
	ParentID  string            `json:"parent_id,omitempty"`
	Operation string            `json:"operation"`
	StartTime time.Time         `json:"start_time"`
	EndTime   *time.Time        `json:"end_time,omitempty"`
	Duration  *time.Duration    `json:"duration,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
	Status    SpanStatus        `json:"status"`
	Error     string            `json:"error,omitempty"`
}

type SpanStatus string

const (
	SpanStatusOK    SpanStatus = "OK"
	SpanStatusError SpanStatus = "ERROR"
)

type spanContextKey struct{}

func StartSpan(ctx context.Context, operation string) (context.Context, *Span) {
	span := &Span{
		TraceID:   generateTraceID(ctx),
		SpanID:    generateID(),
		Operation: operation,
		StartTime: time.Now(),
		Status:    SpanStatusOK,
		Tags:      make(map[string]string),
	}

	if parentSpan := GetSpan(ctx); parentSpan != nil {
		span.ParentID = parentSpan.SpanID
		span.TraceID = parentSpan.TraceID
	}

	return context.WithValue(ctx, spanContextKey{}, span), span
}

func (s *Span) Finish() {
	now := time.Now()
	s.EndTime = &now
	duration := now.Sub(s.StartTime)
	s.Duration = &duration
}

func (s *Span) SetTag(key, value string) {
	if s.Tags == nil {
		s.Tags = make(map[string]string)
	}
	s.Tags[key] = value
}

func (s *Span) SetError(err error) {
	s.Status = SpanStatusError
	if err != nil {
		s.Error = err.Error()
	}
}

func GetSpan(ctx context.Context) *Span {
	if span, ok := ctx.Value(spanContextKey{}).(*Span); ok {
		return span
	}
	return nil
}

func generateTraceID(ctx context.Context) string {
	if existingSpan := GetSpan(ctx); existingSpan != nil {
		return existingSpan.TraceID
	}
	return generateID()
}

func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

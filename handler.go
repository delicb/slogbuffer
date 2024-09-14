// Package slogbuffer provides [slog.Handler] implementation that is capable of
// buffering log records until controlling applications flushes them to real handler.
//
// This can be helpful in situations like:
//   - configuration of real handler is not known until later (e.g. CLI applications that provide
//     user with flags to control log output)
//   - logs are streamed over network and sink is still not ready to accept log records
//   - logs should be flushed only if some condition is met (e.g. panic)
//
// This handler handles those cases but buffering log records and re-emits them on command by
// implementing [slog.Handler] that stores logs in memory until it receives Handler to wrap,
// after which  it is just a simple wrapping handler.
package slogbuffer

import (
	"context"
	"go.uber.org/multierr"
	"log/slog"
	"slices"
)

// BufferLogHandler is [pkg/log/slog.Handler] that buffers records in memory until real log handler
// is provided, at which point it drains memory buffer and uses real handler from that point.
// Zero value is useful.
type BufferLogHandler struct {
	// level is minimal level that this handler will consider storing
	level slog.Level
	// real is handler to which all calls will be sent to and where memory buffer of records.
	// will be drained, when provided
	real slog.Handler

	// buffer is place where records are stored.
	buffer *buffer[record]

	// attrs serve as part of implementation of [slog.Handler.WithAttrs].
	attrs []slog.Attr
	// groups serve as part of implementation of [slog.Handler.WithGroup],
	groups []string

	// parent is reference to handler from which this logger was created
	parent *BufferLogHandler
}

// NewBufferLogHandler returns unbound instance of log handler that stores log records
// until such time when SetRealHandler is called, at which point messages get flushed
// and all subsequent calls are just proxy calls to real handler.
func NewBufferLogHandler(level slog.Level) *BufferLogHandler {
	return NewBoundBufferLogHandler(level, 0)
}

// NewBoundBufferLogHandler creates instance of log handler that stores log records with
// upper limit on number of records, thus providing some level of memory consumption control.
func NewBoundBufferLogHandler(level slog.Level, maxRecords int) *BufferLogHandler {
	return &BufferLogHandler{
		level:  level,
		real:   nil,
		buffer: newBuffer[record](maxRecords),
		attrs:  nil,
		groups: nil,
	}
}

// Implementation of slog.Handler interface.

// compile time check that BufferLogHandler implements slog.Handler interface.
var _ slog.Handler = &BufferLogHandler{}

func (h *BufferLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	rHandler := h.getRealHandler()
	if rHandler == nil {
		return level >= h.level
	}
	return rHandler.Enabled(ctx, level)
}

func (h *BufferLogHandler) Handle(ctx context.Context, r slog.Record) error {
	rHandler := h.getRealHandler()
	if rHandler != nil {
		rh := rHandler
		for _, group := range h.groups {
			rh = rh.WithGroup(group)
		}
		if h.attrs != nil {
			rh = rh.WithAttrs(h.attrs)
		}
		return rh.Handle(ctx, r)
	}
	h.buffer.Add(record{
		Record: r,
		attrs:  h.attrs,
		groups: h.groups,
	})

	return nil
}

func (h *BufferLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	rHandler := h.getRealHandler()
	if rHandler != nil {
		return &BufferLogHandler{
			real: rHandler.WithAttrs(attrs),
		}
	}

	c := h.clone()
	if c.attrs != nil {
		c.attrs = append(c.attrs, attrs...)
	} else {
		c.attrs = attrs
	}
	return c
}

func (h *BufferLogHandler) WithGroup(name string) slog.Handler {
	if len(name) == 0 {
		return h
	}

	rHandler := h.getRealHandler()

	if rHandler != nil {
		return &BufferLogHandler{
			real: rHandler.WithGroup(name),
		}
	}

	child := h.clone()
	if len(h.groups) > 0 {
		child.groups = append(child.groups, name)
	} else {
		child.groups = []string{name}
	}
	return child
}

// Discard removers all stored records.
func (h *BufferLogHandler) Discard() {
	h.buffer.Clear()
}

// SetRealHandler set real slog.Handler for this buffer handler.
// This will cause all buffered log records to be emitted to provided handler.
// Also, from this point on, current handler behaves as simple wrapper and all
// handling is passed to real handler (thus this instance is still usable,
// in case reference to it is held somewhere).
func (h *BufferLogHandler) SetRealHandler(ctx context.Context, real slog.Handler) error {
	var flushErr error
	for rec := range h.buffer.Values() {
		handler := real
		for _, g := range rec.groups {
			handler = handler.WithGroup(g)
		}
		if len(rec.attrs) > 0 {
			handler = handler.WithAttrs(rec.attrs)
		}
		multierr.AppendFunc(&flushErr, func() error { return handler.Handle(ctx, rec.Record) })
	}

	// we don't need storage anymore, let GC collect it
	h.buffer.Clear()

	// switch to wrapper mode
	h.real = real
	return flushErr
}

// clone creates a copy of current handler.
// buffer is reused and all other relevant fields are copied.
func (h *BufferLogHandler) clone() *BufferLogHandler {
	// maxRecords is not copied since it is irrelevant, it is only used
	// to create buffer, and buffer is shared, so we don't need it anymore.
	return &BufferLogHandler{
		level:  h.level,
		real:   h.real,
		buffer: h.buffer,
		attrs:  slices.Clone(h.attrs),
		groups: slices.Clone(h.groups),
		parent: h,
	}
}

// getRealHandler returns instance of real handler either from current instance or
// from parent instance (recursively).
func (h *BufferLogHandler) getRealHandler() slog.Handler {
	if h.real != nil {
		return h.real
	}
	if h.parent != nil {
		return h.parent.getRealHandler()
	}
	return nil
}

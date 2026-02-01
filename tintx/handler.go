package tintx

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"

	"golang.org/x/term"
)

// Handler implements slog.Handler for colored console output.
type Handler struct {
	attrsPrefix string   // pre-formatted attributes from WithAttrs
	groupPrefix string   // "group1.group2." prefix for keys
	groups      []string // group names for tracking

	mu   *sync.Mutex // POINTER - shared across all clones
	w    io.Writer
	opts Options
}

// Compile-time interface check.
var _ slog.Handler = (*Handler)(nil)

// NewHandler creates a new colored console handler.
// TTY detection auto-disables colors unless NoColor is explicitly set.
func NewHandler(w io.Writer, opts *Options) *Handler {
	h := &Handler{
		w:  w,
		mu: &sync.Mutex{},
	}
	if opts != nil {
		h.opts = *opts
	}

	// Auto-detect TTY unless NoColor explicitly set via constructor.
	// We only auto-detect if the caller didn't set NoColor.
	if !h.opts.NoColor {
		if f, ok := w.(*os.File); ok {
			h.opts.NoColor = !term.IsTerminal(int(f.Fd()))
		} else {
			h.opts.NoColor = true // Not a file, assume no TTY
		}
	}

	// Set default time format if not specified.
	if h.opts.TimeFormat == "" {
		h.opts.TimeFormat = "15:04:05.000"
	}

	return h
}

// Enabled reports whether the handler handles records at the given level.
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

// clone creates a copy of the handler, sharing the mutex.
func (h *Handler) clone() *Handler {
	return &Handler{
		attrsPrefix: h.attrsPrefix,
		groupPrefix: h.groupPrefix,
		groups:      h.groups,
		mu:          h.mu, // CRITICAL: shared across clones
		w:           h.w,
		opts:        h.opts,
	}
}

// WithAttrs returns a new Handler with the given attributes pre-formatted.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := h.clone()

	buf := newBuffer()
	defer buf.Free()

	// Pre-format attributes with current group context.
	for _, attr := range attrs {
		h.appendAttr(buf, attr, h.groupPrefix, h.groups)
	}
	h2.attrsPrefix = h.attrsPrefix + string(*buf)
	return h2
}

// WithGroup returns a new Handler with the given group name.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := h.clone()
	h2.groupPrefix += name + "."
	h2.groups = append(h2.groups, name)
	return h2
}

// Handle outputs the log record. Implemented in Plan 02.
func (h *Handler) Handle(_ context.Context, _ slog.Record) error {
	// TODO: Implement in Plan 02
	return nil
}

// appendAttr formats an attribute. Stub for WithAttrs to compile.
func (h *Handler) appendAttr(buf *buffer, a slog.Attr, groupPrefix string, _ []string) {
	// Resolve any LogValuer types.
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}
	buf.WriteString(groupPrefix)
	buf.WriteString(a.Key)
	buf.WriteByte('=')
	buf.WriteString(a.Value.String())
	buf.WriteByte(' ')
}

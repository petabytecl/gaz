package tint

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

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

// Handle outputs the log record with colorized formatting.
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	buf := newBuffer()
	defer buf.Free()

	// 1. Time (if not zero)
	if !r.Time.IsZero() {
		h.appendTime(buf, r.Time)
		buf.WriteByte(' ')
	}

	// 2. Level (colorized)
	h.appendLevel(buf, r.Level)
	buf.WriteByte(' ')

	// 3. Source (if AddSource enabled and PC available)
	if h.opts.AddSource && r.PC != 0 {
		h.appendSource(buf, r.PC)
		buf.WriteByte(' ')
	}

	// 4. Message
	buf.WriteString(r.Message)

	// 5. Pre-formatted attrs from WithAttrs (if any)
	if len(h.attrsPrefix) > 0 {
		buf.WriteByte(' ')
		buf.WriteString(h.attrsPrefix)
	}

	// 6. Record attrs
	if r.NumAttrs() > 0 {
		buf.WriteByte(' ')
		r.Attrs(func(a slog.Attr) bool {
			h.appendAttr(buf, a, h.groupPrefix, h.groups)
			return true
		})
	}

	// Remove trailing space if present, add newline
	if len(*buf) > 0 && (*buf)[len(*buf)-1] == ' ' {
		(*buf)[len(*buf)-1] = '\n'
	} else {
		buf.WriteByte('\n')
	}

	// Thread-safe write
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(*buf)
	return err
}

// appendLevel writes the colorized log level.
func (h *Handler) appendLevel(buf *buffer, level slog.Level) {
	if !h.opts.NoColor {
		switch {
		case level < slog.LevelInfo:
			buf.WriteString(ansiBrightBlue) // DEBUG
		case level < slog.LevelWarn:
			buf.WriteString(ansiBrightGreen) // INFO
		case level < slog.LevelError:
			buf.WriteString(ansiBrightYellow) // WARN
		default:
			buf.WriteString(ansiBrightRed) // ERROR
		}
	}

	// Write level string (3-char for alignment)
	switch {
	case level < slog.LevelInfo:
		buf.WriteString("DBG")
	case level < slog.LevelWarn:
		buf.WriteString("INF")
	case level < slog.LevelError:
		buf.WriteString("WRN")
	default:
		buf.WriteString("ERR")
	}

	if !h.opts.NoColor {
		buf.WriteString(ansiReset)
	}
}

// appendTime writes the formatted timestamp.
func (h *Handler) appendTime(buf *buffer, t time.Time) {
	if !h.opts.NoColor {
		buf.WriteString(ansiFaint)
	}
	*buf = t.AppendFormat(*buf, h.opts.TimeFormat)
	if !h.opts.NoColor {
		buf.WriteString(ansiReset)
	}
}

// appendSource writes the source file:line.
func (h *Handler) appendSource(buf *buffer, pc uintptr) {
	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()
	if f.File == "" {
		return
	}

	if !h.opts.NoColor {
		buf.WriteString(ansiFaint)
	}

	// Write dir/file.go:line format
	dir := filepath.Base(filepath.Dir(f.File))
	file := filepath.Base(f.File)
	buf.WriteString(dir)
	buf.WriteByte('/')
	buf.WriteString(file)
	buf.WriteByte(':')

	// Use strconv for zero-allocation int formatting
	*buf = strconv.AppendInt(*buf, int64(f.Line), 10)

	if !h.opts.NoColor {
		buf.WriteString(ansiReset)
	}
}

// appendAttr formats an attribute with proper value resolution.
func (h *Handler) appendAttr(buf *buffer, a slog.Attr, groupPrefix string, groups []string) {
	// Resolve LogValuer types
	a.Value = a.Value.Resolve()

	// Skip empty attributes
	if a.Equal(slog.Attr{}) {
		return
	}

	// Handle group type recursively
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		// Empty group with name should be ignored
		if len(attrs) == 0 {
			return
		}

		// Update prefix for nested group
		prefix := groupPrefix
		if a.Key != "" {
			prefix += a.Key + "."
		}

		for _, ga := range attrs {
			h.appendAttr(buf, ga, prefix, groups)
		}
		return
	}

	// Key with group prefix
	if !h.opts.NoColor {
		buf.WriteString(ansiFaint)
	}
	buf.WriteString(groupPrefix)
	buf.WriteString(a.Key)
	buf.WriteByte('=')
	if !h.opts.NoColor {
		buf.WriteString(ansiReset)
	}

	// Value formatting
	h.appendValue(buf, a.Value)
	buf.WriteByte(' ')
}

// appendValue formats the value based on kind.
func (h *Handler) appendValue(buf *buffer, v slog.Value) {
	switch v.Kind() {
	case slog.KindString:
		buf.WriteString(v.String())
	case slog.KindInt64:
		*buf = strconv.AppendInt(*buf, v.Int64(), 10)
	case slog.KindUint64:
		*buf = strconv.AppendUint(*buf, v.Uint64(), 10)
	case slog.KindFloat64:
		*buf = strconv.AppendFloat(*buf, v.Float64(), 'f', -1, 64)
	case slog.KindBool:
		*buf = strconv.AppendBool(*buf, v.Bool())
	case slog.KindDuration:
		buf.WriteString(v.Duration().String())
	case slog.KindTime:
		*buf = v.Time().AppendFormat(*buf, time.RFC3339)
	case slog.KindAny:
		fmt.Fprint(buf, v.Any())
	default:
		buf.WriteString(v.String())
	}
}

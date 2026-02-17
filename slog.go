package bolt

import (
	"context"
	"io"
	"log/slog"
	"runtime"
	"sync"
)

// SlogHandler implements [slog.Handler] backed by Bolt's zero-allocation engine.
// This allows use of the standard [slog] API with Bolt's high-performance output.
//
// Usage:
//
//	handler := bolt.NewSlogHandler(os.Stdout, nil)
//	logger := slog.New(handler)
//	logger.Info("request handled", "method", "GET", "status", 200)
type SlogHandler struct {
	out    io.Writer
	level  slog.Level
	groups []string
	attrs  []slog.Attr
	mu     *sync.Mutex
}

// SlogHandlerOptions configures a [SlogHandler].
type SlogHandlerOptions struct {
	// Level sets the minimum log level. Defaults to slog.LevelInfo.
	Level slog.Leveler

	// AddSource adds source file information to log entries.
	AddSource bool
}

// NewSlogHandler creates a new [slog.Handler] that writes JSON logs using Bolt's
// zero-allocation serialization. If opts is nil, defaults are used.
func NewSlogHandler(out io.Writer, opts *SlogHandlerOptions) *SlogHandler {
	h := &SlogHandler{
		out: out,
		mu:  &sync.Mutex{},
	}
	if opts != nil {
		if opts.Level != nil {
			h.level = opts.Level.Level()
		}
	}
	return h
}

// Enabled reports whether the handler handles records at the given level.
func (h *SlogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle writes the [slog.Record] as JSON to the output writer.
func (h *SlogHandler) Handle(_ context.Context, r slog.Record) error {
	buf := bufPool.Get().(*[]byte)
	b := (*buf)[:0]

	// Open JSON object with level
	b = append(b, `{"level":"`...)
	b = appendSlogLevel(b, r.Level)
	b = append(b, '"')

	// Add timestamp
	if !r.Time.IsZero() {
		b = append(b, `,"time":"`...)
		b = appendRFC3339(b, r.Time)
		b = append(b, '"')
	}

	// Add source if the record has PC info
	if r.PC != 0 {
		frame, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		if frame.File != "" {
			b = append(b, `,"source":{"function":"`...)
			b = appendJSONString(b, frame.Function)
			b = append(b, `","file":"`...)
			b = appendJSONString(b, frame.File)
			b = append(b, `","line":`...)
			b = appendInt(b, frame.Line)
			b = append(b, '}')
		}
	}

	// Add pre-computed group prefix and attrs
	prefix := h.groupPrefix()

	// Add pre-set attrs
	for _, a := range h.attrs {
		b = appendSlogAttr(b, a, prefix)
	}

	// Add record attrs
	r.Attrs(func(a slog.Attr) bool {
		b = appendSlogAttr(b, a, prefix)
		return true
	})

	// Add message
	if r.Message != "" {
		b = append(b, `,"message":"`...)
		b = appendJSONString(b, r.Message)
		b = append(b, '"')
	}

	// Close JSON and add newline
	b = append(b, '}', '\n')

	h.mu.Lock()
	_, err := h.out.Write(b)
	h.mu.Unlock()

	*buf = b
	bufPool.Put(buf)

	return err
}

// WithAttrs returns a new handler with the given attributes pre-set.
func (h *SlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := h.clone()
	h2.attrs = append(h2.attrs, attrs...)
	return h2
}

// WithGroup returns a new handler that nests subsequent attributes under the given key.
func (h *SlogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := h.clone()
	h2.groups = append(h2.groups, name)
	return h2
}

func (h *SlogHandler) clone() *SlogHandler {
	return &SlogHandler{
		out:    h.out,
		level:  h.level,
		groups: append([]string(nil), h.groups...),
		attrs:  append([]slog.Attr(nil), h.attrs...),
		mu:     h.mu,
	}
}

func (h *SlogHandler) groupPrefix() string {
	if len(h.groups) == 0 {
		return ""
	}
	prefix := ""
	for _, g := range h.groups {
		prefix += g + "."
	}
	return prefix
}

// bufPool is a pool for slog handler buffers.
var bufPool = &sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 1024)
		return &b
	},
}

func appendSlogLevel(b []byte, l slog.Level) []byte {
	switch {
	case l < slog.LevelInfo:
		return append(b, debugStr...)
	case l < slog.LevelWarn:
		return append(b, infoStr...)
	case l < slog.LevelError:
		return append(b, warnStr...)
	default:
		return append(b, errorStr...)
	}
}

func appendSlogAttr(b []byte, a slog.Attr, prefix string) []byte {
	a.Value = a.Value.Resolve()

	// Skip empty attrs
	if a.Equal(slog.Attr{}) {
		return b
	}

	// Handle groups inline
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return b
		}
		// If the attr has a key, it becomes a nested prefix
		newPrefix := prefix
		if a.Key != "" {
			newPrefix = prefix + a.Key + "."
		}
		for _, ga := range attrs {
			b = appendSlogAttr(b, ga, newPrefix)
		}
		return b
	}

	b = append(b, ',', '"')
	b = appendJSONString(b, prefix+a.Key)
	b = append(b, `":`...)

	switch a.Value.Kind() {
	case slog.KindString:
		b = append(b, '"')
		b = appendJSONString(b, a.Value.String())
		b = append(b, '"')
	case slog.KindInt64:
		b = appendInt(b, int(a.Value.Int64()))
	case slog.KindUint64:
		b = appendUint(b, a.Value.Uint64())
	case slog.KindFloat64:
		b = appendFloat64(b, a.Value.Float64())
	case slog.KindBool:
		b = appendBool(b, a.Value.Bool())
	case slog.KindDuration:
		b = append(b, '"')
		b = appendJSONString(b, a.Value.Duration().String())
		b = append(b, '"')
	case slog.KindTime:
		b = append(b, '"')
		b = appendRFC3339(b, a.Value.Time())
		b = append(b, '"')
	default:
		b = append(b, '"')
		b = appendJSONString(b, a.Value.String())
		b = append(b, '"')
	}
	return b
}

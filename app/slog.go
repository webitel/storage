package app

import (
	"context"
	"log/slog"

	"github.com/webitel/wlog"
)

// newSlogLogger returns a *slog.Logger that forwards records to log, for
// components that speak log/slog (e.g. webitel-go-kit modules).
func newSlogLogger(log *wlog.Logger) *slog.Logger {
	return slog.New(&slogHandler{log: log})
}

// slogHandler adapts *wlog.Logger to the slog.Handler interface. Group names
// are flattened into dotted field keys ("group.key"): zap namespaces would
// nest every subsequent sibling group instead.
type slogHandler struct {
	log *wlog.Logger
	// prefix is the accumulated open-group prefix, "" or "g1.g2." with a
	// trailing dot.
	prefix string
}

func (h *slogHandler) Enabled(context.Context, slog.Level) bool {
	// wlog exposes no level probe; its zap cores filter by level themselves.
	return true
}

func (h *slogHandler) Handle(_ context.Context, r slog.Record) error {
	fields := make([]wlog.Field, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		fields = appendAttr(fields, a, h.prefix)
		return true
	})

	switch {
	case r.Level < slog.LevelInfo:
		h.log.Debug(r.Message, fields...)
	case r.Level < slog.LevelWarn:
		h.log.Info(r.Message, fields...)
	case r.Level < slog.LevelError:
		h.log.Warn(r.Message, fields...)
	default:
		h.log.Error(r.Message, fields...)
	}

	return nil
}

func (h *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	fields := make([]wlog.Field, 0, len(attrs))
	for _, a := range attrs {
		fields = appendAttr(fields, a, h.prefix)
	}

	return &slogHandler{log: h.log.With(fields...), prefix: h.prefix}
}

func (h *slogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return &slogHandler{log: h.log, prefix: h.prefix + name + "."}
}

func appendAttr(fields []wlog.Field, a slog.Attr, prefix string) []wlog.Field {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return fields
	}

	if a.Value.Kind() == slog.KindGroup {
		if a.Key != "" {
			prefix += a.Key + "."
		}

		for _, ga := range a.Value.Group() {
			fields = appendAttr(fields, ga, prefix)
		}

		return fields
	}

	return append(fields, wlog.Any(prefix+a.Key, a.Value.Any()))
}

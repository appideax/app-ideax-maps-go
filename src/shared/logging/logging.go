// Package logging arma el logger de la app.
package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
)

// bootHandler escribe los logs del arranque como texto de una linea, para leerlos
// en la terminal, y el resto en JSON. El cambio
// lo hace la funcion que devuelve New, cuando el arranque termina.
type bootHandler struct {
	booting *atomic.Bool
	boot    slog.Handler
	app     slog.Handler
}

func (h *bootHandler) current() slog.Handler {
	if h.booting.Load() {
		return h.boot
	}
	return h.app
}

func (h *bootHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.current().Enabled(ctx, level)
}

// Handle escribe el registro. En el arranque, un mensaje de varias lineas sale
// con la primera como registro normal y el resto debajo, tal cual, para que se
// lea como una lista (el formato de texto las pondria en una sola linea con \n).
func (h *bootHandler) Handle(ctx context.Context, r slog.Record) error {
	if !h.booting.Load() {
		return h.app.Handle(ctx, r)
	}
	first, rest, multiline := strings.Cut(r.Message, "\n")
	if !multiline {
		return h.boot.Handle(ctx, r)
	}

	head := slog.NewRecord(r.Time, r.Level, first, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		head.AddAttrs(a)
		return true
	})
	if err := h.boot.Handle(ctx, head); err != nil {
		return err
	}
	_, err := fmt.Fprintln(os.Stdout, rest)
	return err
}

func (h *bootHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &bootHandler{booting: h.booting, boot: h.boot.WithAttrs(attrs), app: h.app.WithAttrs(attrs)}
}

func (h *bootHandler) WithGroup(name string) slog.Handler {
	return &bootHandler{booting: h.booting, boot: h.boot.WithGroup(name), app: h.app.WithGroup(name)}
}

// New arma el logger de la app y devuelve la funcion que da por terminado el
// arranque: desde ahi los logs salen en JSON.
func New(level slog.Level) (*slog.Logger, func()) {
	// La hora va corta (HH:MM:SS) en los dos formatos.
	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 && a.Key == slog.TimeKey && a.Value.Kind() == slog.KindTime {
				a.Value = slog.StringValue(a.Value.Time().Format("15:04:05"))
			}
			return a
		},
	}

	booting := new(atomic.Bool)
	booting.Store(true)
	handler := &bootHandler{
		booting: booting,
		boot:    slog.NewTextHandler(os.Stdout, opts),
		app:     slog.NewJSONHandler(os.Stdout, opts),
	}
	return slog.New(handler), func() { booting.Store(false) }
}

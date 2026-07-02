package logger

import (
	"context"
	"io"
	"os"

	"github.com/rs/zerolog"
)

type contextKey struct{}

type Logger struct {
	zl zerolog.Logger
}

func New(env string) Logger {
	return NewWithWriter(env, os.Stdout)
}

func NewWithWriter(env string, w io.Writer) Logger {
	if env == "development" {
		console := zerolog.ConsoleWriter{Out: w}
		return Logger{zl: zerolog.New(console).With().Timestamp().Logger()}
	}

	return Logger{zl: zerolog.New(w).With().Timestamp().Logger()}
}

func WithCtx(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

func FromCtx(ctx context.Context) Logger {
	if l, ok := ctx.Value(contextKey{}).(Logger); ok {
		return l
	}

	return New(os.Getenv("ENVIRONMENT"))
}

func (l Logger) With(keysAndValues ...any) Logger {
	ctx := l.zl.With()
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		ctx = ctx.Interface(key, keysAndValues[i+1])
	}
	return Logger{zl: ctx.Logger()}
}

func (l Logger) Info(msg string, keysAndValues ...any) {
	l.logEvent(l.zl.Info(), msg, keysAndValues...)
}

func (l Logger) Error(msg string, keysAndValues ...any) {
	l.logEvent(l.zl.Error(), msg, keysAndValues...)
}

func (l Logger) Debug(msg string, keysAndValues ...any) {
	l.logEvent(l.zl.Debug(), msg, keysAndValues...)
}

func (l Logger) Warn(msg string, keysAndValues ...any) {
	l.logEvent(l.zl.Warn(), msg, keysAndValues...)
}

func (l Logger) logEvent(event *zerolog.Event, msg string, keysAndValues ...any) {
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}

		event = event.Interface(key, keysAndValues[i+1])
	}

	event.Msg(msg)
}

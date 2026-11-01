package i18n

import (
	"context"
	"fmt"
)

type Language struct {
	Tag  string
	Name string
}

var Supported = []Language{
	{Tag: "en", Name: "English"},
	{Tag: "pt-BR", Name: "Português (Brasil)"},
}

const CookieName = "lang"

type contextKey struct{}

func WithLang(ctx context.Context, tag string) context.Context {
	return context.WithValue(ctx, contextKey{}, tag)
}

func Lang(ctx context.Context) string {
	if tag, ok := ctx.Value(contextKey{}).(string); ok {
		return tag
	}
	return Supported[0].Tag
}

func T(ctx context.Context, key string, args ...any) string {
	text, ok := catalogs[Lang(ctx)][key]
	if !ok {
		text = key
	}
	if len(args) > 0 {
		return fmt.Sprintf(text, args...)
	}
	return text
}

func IsSupported(tag string) bool {
	for _, l := range Supported {
		if l.Tag == tag {
			return true
		}
	}
	return false
}

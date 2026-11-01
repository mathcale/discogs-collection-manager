package i18n_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mathcale/discogs-collection-manager/internal/pkg/i18n"
)

func TestT(t *testing.T) {
	en := i18n.WithLang(context.Background(), "en")
	ptBR := i18n.WithLang(context.Background(), "pt-BR")

	tests := []struct {
		name string
		ctx  context.Context
		key  string
		args []any
		want string
	}{
		{"english returns the key", en, "Log in", nil, "Log in"},
		{"no language returns the key", context.Background(), "Log in", nil, "Log in"},
		{"portuguese returns the translation", ptBR, "Log in", nil, "Entrar"},
		{"missing translation falls back to the key", ptBR, "Not in any catalog", nil, "Not in any catalog"},
		{"applies format args", ptBR, "Your verification code is: <strong>%s</strong>. It expires in 10 minutes.", []any{"123456"}, "Seu código de verificação é: <strong>123456</strong>. Ele expira em 10 minutos."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, i18n.T(tt.ctx, tt.key, tt.args...))
		})
	}
}

func TestLang_DefaultsToEnglish(t *testing.T) {
	assert.Equal(t, "en", i18n.Lang(context.Background()))
}

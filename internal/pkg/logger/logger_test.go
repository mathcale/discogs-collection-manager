package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mathcale/discogs-collection-manager/internal/pkg/logger"
)

func TestNew_JSONOutput(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewWithWriter("production", &buf)

	l.Info("test message", "key", "value")

	var result map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "test message", result["message"])
	assert.Equal(t, "info", result["level"])
	assert.Equal(t, "value", result["key"])
}

func TestNew_DevelopmentOutput(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewWithWriter("development", &buf)

	l.Info("hello world")

	out := buf.String()
	assert.Contains(t, out, "hello world")
}

func TestLogger_Levels(t *testing.T) {
	tests := []struct {
		name      string
		log       func(l logger.Logger)
		wantLevel string
	}{
		{"info", func(l logger.Logger) { l.Info("msg") }, "info"},
		{"error", func(l logger.Logger) { l.Error("msg") }, "error"},
		{"debug", func(l logger.Logger) { l.Debug("msg") }, "debug"},
		{"warn", func(l logger.Logger) { l.Warn("msg") }, "warn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := logger.NewWithWriter("production", &buf)
			tt.log(l)

			var result map[string]any
			require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
			assert.Equal(t, tt.wantLevel, result["level"])
		})
	}
}

func TestLogger_KeyValuePairs(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewWithWriter("production", &buf)

	l.Info("msg", "stringKey", "stringVal", "intKey", 42)

	var result map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "stringVal", result["stringKey"])
	assert.InDelta(t, 42, result["intKey"], 0.01)
}

func TestWithCtx_FromCtx(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewWithWriter("production", &buf)

	ctx := logger.WithCtx(context.Background(), l)
	retrieved := logger.FromCtx(ctx)

	retrieved.Info("from context", "source", "ctx")

	var result map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "from context", result["message"])
}

func TestFromCtx_FallsBackToDefault(t *testing.T) {
	l := logger.FromCtx(context.Background())
	assert.NotNil(t, l)
}

func TestNew_DefaultEnvIsJSON(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewWithWriter("", &buf)

	l.Info("msg")

	out := strings.TrimSpace(buf.String())
	assert.True(t, strings.HasPrefix(out, "{"), "expected JSON output for non-development env")
}

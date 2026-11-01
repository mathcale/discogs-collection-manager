package web_test

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var translationKey = regexp.MustCompile(`i18n\.T\((?:ctx|r\.Context\(\)),\s*("(?:[^"\\]|\\.)*")`)

func usedTranslationKeys(t *testing.T) map[string]struct{} {
	used := map[string]struct{}{}
	err := filepath.WalkDir(filepath.Join(repoRoot, "internal"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasSuffix(path, "_templ.go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		if !strings.HasSuffix(path, ".templ") && !strings.HasSuffix(path, ".go") {
			return nil
		}
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		for _, match := range translationKey.FindAllStringSubmatch(string(content), -1) {
			key, err := strconv.Unquote(match[1])
			require.NoError(t, err, "%s: %s", path, match[1])
			used[key] = struct{}{}
		}
		return nil
	})
	require.NoError(t, err)
	return used
}

func portugueseCatalog(t *testing.T) map[string]string {
	content, err := os.ReadFile(filepath.Join(repoRoot, "internal", "pkg", "i18n", "locales", "pt-BR.json"))
	require.NoError(t, err)
	var catalog map[string]string
	require.NoError(t, json.Unmarshal(content, &catalog))
	return catalog
}

func TestI18nCatalog_HasEveryUsedKey(t *testing.T) {
	catalog := portugueseCatalog(t)

	var missing []string
	for key := range usedTranslationKeys(t) {
		if _, ok := catalog[key]; !ok {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)

	assert.Empty(t, missing, "add these keys to internal/pkg/i18n/locales/pt-BR.json")
}

func TestI18nCatalog_HasNoUnusedKeys(t *testing.T) {
	used := usedTranslationKeys(t)

	var unused, empty []string
	for key, value := range portugueseCatalog(t) {
		if _, ok := used[key]; !ok {
			unused = append(unused, key)
		}
		if value == "" {
			empty = append(empty, key)
		}
	}
	sort.Strings(unused)
	sort.Strings(empty)

	assert.Empty(t, unused, "remove these stale keys from internal/pkg/i18n/locales/pt-BR.json")
	assert.Empty(t, empty, "these keys have empty translations in internal/pkg/i18n/locales/pt-BR.json")
}

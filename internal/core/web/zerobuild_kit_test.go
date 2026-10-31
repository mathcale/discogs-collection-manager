package web_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const repoRoot = "../../.."

var (
	templateClass   = regexp.MustCompile(`(?:^|[^a-zA-Z0-9_-])(z-[a-zA-Z0-9_-]+)`)
	stylesheetClass = regexp.MustCompile(`\.(z-[a-zA-Z0-9_-]+)`)
)

func TestVendoredZerobuildKit_HasEveryUsedClass(t *testing.T) {
	used := map[string]struct{}{}
	collect := func(path string, pattern *regexp.Regexp) {
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		for _, match := range pattern.FindAllStringSubmatch(string(content), -1) {
			used[match[1]] = struct{}{}
		}
	}

	collect(filepath.Join(repoRoot, "static", "app.css"), stylesheetClass)
	err := filepath.WalkDir(filepath.Join(repoRoot, "internal"), func(path string, d fs.DirEntry, err error) error {
		if err == nil && strings.HasSuffix(path, ".templ") {
			collect(path, templateClass)
		}
		return err
	})
	require.NoError(t, err)

	kit, err := os.ReadFile(filepath.Join(repoRoot, "static", "vendor", "0build", "kit.min.css"))
	require.NoError(t, err)

	var missing []string
	for class := range used {
		if !regexp.MustCompile(`\.` + regexp.QuoteMeta(class) + `([^a-zA-Z0-9_-]|$)`).Match(kit) {
			missing = append(missing, class)
		}
	}
	sort.Strings(missing)

	assert.Empty(t, missing, "static/vendor/0build/kit.min.css is stale: run `make 0build` and commit static/vendor/0build/")
}

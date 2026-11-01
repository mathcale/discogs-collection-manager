package i18n

import (
	"embed"
	"encoding/json"
	"path"
	"strings"
)

//go:embed locales/*.json
var localeFiles embed.FS

var catalogs = loadCatalogs()

func loadCatalogs() map[string]map[string]string {
	entries, err := localeFiles.ReadDir("locales")
	if err != nil {
		panic(err)
	}

	loaded := map[string]map[string]string{}
	for _, entry := range entries {
		content, err := localeFiles.ReadFile(path.Join("locales", entry.Name()))
		if err != nil {
			panic(err)
		}
		var catalog map[string]string
		if err := json.Unmarshal(content, &catalog); err != nil {
			panic("i18n: " + entry.Name() + ": " + err.Error())
		}
		loaded[strings.TrimSuffix(entry.Name(), ".json")] = catalog
	}
	return loaded
}

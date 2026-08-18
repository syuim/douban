package handler

import (
	"net/http"
	"strings"

	"stremio-addon-douban/internal/collection"
)

// CatalogResourceHandler parses catalog URLs manually because
// chi cannot match path segments containing dots (e.g. "movie_hot_gaia.json").
// Format: /catalog/{type}/{catalogID}.json（{type} 为兼容保留，实际由 catalogID 决定）
func CatalogResourceHandler(w http.ResponseWriter, r *http.Request) {
	rest, ok := parseCatalogPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	dispatchCatalog(w, r, rest)
}

func parseCatalogPath(path string) ([]string, bool) {
	path = strings.TrimSuffix(path, ".json")
	path = strings.TrimPrefix(path, "/")
	segments := strings.Split(path, "/")
	if len(segments) >= 3 && segments[0] == "catalog" {
		return segments[2:], true
	}
	return nil, false
}

func dispatchCatalog(w http.ResponseWriter, r *http.Request, rest []string) {
	if len(rest) == 0 {
		http.NotFound(w, r)
		return
	}
	catalogID := rest[0]
	if len(catalogID) > 64 || strings.ContainsAny(catalogID, "/\\") {
		http.NotFound(w, r)
		return
	}
	extraStr := ""
	if len(rest) > 1 {
		extraStr = strings.Join(rest[1:], "/")
	}

	extra := parseExtra(extraStr, r)

	if catalogID == "__random__" {
		handleRandomCatalog(w, r, extra)
		return
	}

	if collection.FindTheaterConfig(catalogID) != nil {
		handleTheaterCatalog(w, r, catalogID, extra)
		return
	}

	handleCatalogRequest(w, r, catalogID, extra)
}

package handler

import (
	"reflect"
	"testing"
)

func TestParseCatalogPath(t *testing.T) {
	cases := []struct {
		path string
		rest []string
		ok   bool
	}{
		{"/catalog/movie/movie_hot_gaia.json", []string{"movie_hot_gaia"}, true},
		{"/catalog/series/tv_hot/skip=20&genre=喜剧.json", []string{"tv_hot", "skip=20&genre=喜剧"}, true},
		{"/catalog/series/__random__.json", []string{"__random__"}, true},
		{"/catalog", nil, false},
		{"/catalog/movie", nil, false},
		{"/meta/movie/douban:37450627.json", nil, false},
		{"/suyu/catalog/movie/movie_hot_gaia.json", nil, false},
		{"/", nil, false},
		{"/random/path/here.json", nil, false},
	}
	for _, c := range cases {
		rest, ok := parseCatalogPath(c.path)
		if ok != c.ok {
			t.Errorf("parseCatalogPath(%q) ok = %v, want %v", c.path, ok, c.ok)
			continue
		}
		if ok && !reflect.DeepEqual(rest, c.rest) {
			t.Errorf("parseCatalogPath(%q) = %v, want %v", c.path, rest, c.rest)
		}
	}
}

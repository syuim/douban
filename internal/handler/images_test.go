package handler

import (
	"os"
	"testing"
)

func TestDirectProxyURL(t *testing.T) {
	proxyURL := "https://proxy.laoz.org/url"
	t.Setenv("PROXY_URL", proxyURL)

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"absolute image URL", "https://image.tmdb.org/t/p/original/abc.jpg", proxyURL + "?url=https%3A%2F%2Fimage.tmdb.org%2Ft%2Fp%2Foriginal%2Fabc.jpg"},
		{"douban cover URL", "https://img9.doubanio.com/view/photo/m/public/p123.jpg", proxyURL + "?url=https%3A%2F%2Fimg9.doubanio.com%2Fview%2Fphoto%2Fm%2Fpublic%2Fp123.jpg"},
		{"empty URL", "", ""},
		{"URL with Chinese chars", "https://example.com/图片.jpg", proxyURL + "?url=https%3A%2F%2Fexample.com%2F%E5%9B%BE%E7%89%87.jpg"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := directProxyURL(c.input)
			if got != c.want {
				t.Errorf("directProxyURL(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

func TestDirectProxyURLWithCustomProxy(t *testing.T) {
	customURL := "https://custom-proxy.example.com/url"
	t.Setenv("PROXY_URL", customURL)

	got := directProxyURL("https://example.com/img.jpg")
	want := customURL + "?url=https%3A%2F%2Fexample.com%2Fimg.jpg"
	if got != want {
		t.Errorf("directProxyURL with custom PROXY_URL = %q, want %q", got, want)
	}
}

func TestDirectProxyURLWithoutEnv(t *testing.T) {
	os.Unsetenv("PROXY_URL")

	got := directProxyURL("https://example.com/img.jpg")
	if got == "" {
		t.Error("directProxyURL returned empty without env, expected fallback to default")
	}
}
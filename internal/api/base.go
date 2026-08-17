package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"stremio-addon-douban/internal/db"
)

// DefaultProxyURL 是统一代理入口（emby-proxy /url，与 /img 同一实现），
// 图片与内部 API 都经它转发：POST JSON {url, method, headers, body}，
// GET 目标请求可命中 CF 边缘缓存。
const DefaultProxyURL = "https://proxy.laoz.org/url"

type CacheConfig struct {
	Key string
	TTL int // seconds
}

type BaseAPI struct {
	baseURL string
	headers map[string]string

	requestMap sync.Map // key -> chan struct{}
}

func NewBaseAPI(baseURL string, headers map[string]string) *BaseAPI {
	return &BaseAPI{
		baseURL: baseURL,
		headers: headers,
	}
}

type ProxyRequest struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    io.Reader
}

type proxyPayload struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// ProxyFetch 把请求经统一代理转发（emby-proxy /url：POST JSON，无鉴权），
// 网络错误与 429/418/5xx 指数退避重试，兼容 GET 目标命中 CF 边缘缓存。
func ProxyFetch(ctx context.Context, proxyURL string, req ProxyRequest) (*http.Response, error) {
	return proxyFetchWithHTTPClient(ctx, proxyHTTPClient, proxyURL, req)
}

func proxyFetchWithHTTPClient(ctx context.Context, client *http.Client, proxyURL string, req ProxyRequest) (*http.Response, error) {
	var bodyStr string
	if req.Body != nil {
		data, _ := io.ReadAll(req.Body)
		bodyStr = string(data)
	}

	allHeaders := make(map[string]string)
	for k, v := range req.Headers {
		allHeaders[k] = v
	}
	if strings.EqualFold(req.Method, "GET") {
		for k := range allHeaders {
			if strings.EqualFold(k, "content-type") {
				delete(allHeaders, k)
			}
		}
	}

	payload, err := json.Marshal(proxyPayload{
		URL:     req.URL,
		Method:  req.Method,
		Headers: allHeaders,
		Body:    bodyStr,
	})
	if err != nil {
		return nil, err
	}

	var resp *http.Response
	var lastErr error
	for attempt := 0; attempt <= 3; attempt++ {
		if attempt > 0 {
			wait := time.Duration(1<<(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		}

		resp, lastErr = client.Post(proxyURL, "application/json", bytes.NewReader(payload))
		if lastErr != nil {
			if attempt == 3 {
				return nil, lastErr
			}
			continue
		}
		if resp.StatusCode != 429 && resp.StatusCode != 418 && resp.StatusCode < 500 || attempt == 3 {
			return resp, nil
		}
		resp.Body.Close()
	}
	if resp != nil {
		return resp, nil
	}
	return nil, lastErr
}

var proxyHTTPClient = &http.Client{Timeout: 30 * time.Second}

func (b *BaseAPI) doRequest(ctx context.Context, method, url string, headers map[string]string, body io.Reader) (*http.Response, error) {
	allHeaders := make(map[string]string, len(b.headers)+len(headers))
	for k, v := range b.headers {
		allHeaders[k] = v
	}
	for k, v := range headers {
		allHeaders[k] = v
	}

	proxyURL := os.Getenv("PROXY_URL")
	if proxyURL == "" {
		proxyURL = DefaultProxyURL
	}
	return ProxyFetch(ctx, proxyURL, ProxyRequest{URL: url, Method: method, Headers: allHeaders, Body: body})
}

func (b *BaseAPI) Request(ctx context.Context, method, path string, params map[string]string, headers map[string]string, body io.Reader, cacheCfg *CacheConfig) ([]byte, int, http.Header, error) {
	fullURL := b.baseURL + path
	if len(params) > 0 {
		q := url.Values{}
		for k, v := range params {
			q.Set(k, v)
		}
		fullURL += "?" + q.Encode()
	}

	if cacheCfg != nil {
		if data, ok, _ := b.getCache(ctx, cacheCfg.Key); ok {
			return data, 200, nil, nil
		}
	}

	// request dedup
	if cacheCfg != nil {
		ch, data, ok, err := b.acquireRequest(ctx, cacheCfg.Key)
		if err != nil {
			return nil, 0, nil, err
		}
		if ok {
			return data, 200, nil, nil
		}
		defer func() {
			b.requestMap.Delete(cacheCfg.Key)
			close(ch)
		}()
	}

	resp, err := b.doRequest(ctx, method, fullURL, headers, body)
	if err != nil {
		return nil, 0, nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, resp.Header, err
	}

	if resp.StatusCode < 400 && cacheCfg != nil {
		b.setCache(cacheCfg.Key, data, cacheCfg.TTL)
	}

	return data, resp.StatusCode, resp.Header, nil
}

func (b *BaseAPI) acquireRequest(ctx context.Context, key string) (chan struct{}, []byte, bool, error) {
	for {
		ch := make(chan struct{})
		if existing, loaded := b.requestMap.LoadOrStore(key, ch); loaded {
			existingCh := existing.(chan struct{})
			select {
			case <-existingCh:
			case <-ctx.Done():
				return nil, nil, false, ctx.Err()
			}
			if data, ok, _ := b.getCache(ctx, key); ok {
				return nil, data, true, nil
			}
			continue
		}
		return ch, nil, false, nil
	}
}

type APIError struct {
	Status            int
	Body              string
	RetryAfterSeconds int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API returned status %d: %s", e.Status, e.Body)
}

func (b *BaseAPI) RequestJSON(ctx context.Context, method, path string, params map[string]string, headers map[string]string, body io.Reader, cacheCfg *CacheConfig, target any) error {
	data, status, respHeader, err := b.Request(ctx, method, path, params, headers, body, cacheCfg)
	if err != nil {
		return err
	}
	if status >= 400 {
		if !strings.Contains(b.baseURL, "webservice.fanart.tv") {
			log.Printf("❌ %d %s", status, b.baseURL+path)
		}
		apiErr := &APIError{Status: status, Body: string(data)}
		if respHeader != nil {
			if ra := respHeader.Get("Retry-After"); ra != "" {
				if secs, err := strconv.Atoi(ra); err == nil {
					apiErr.RetryAfterSeconds = secs
				}
			}
		}
		return apiErr
	}
	if target != nil {
		return json.Unmarshal(data, target)
	}
	return nil
}

func (b *BaseAPI) getCache(ctx context.Context, key string) ([]byte, bool, bool) {
	database, err := db.GetDB()
	if err != nil {
		return nil, false, false
	}

	var value string
	var expiresAt int64
	err = database.QueryRowContext(ctx, "SELECT value, expires_at FROM api_cache WHERE key = ?", key).
		Scan(&value, &expiresAt)
	if err != nil {
		return nil, false, false
	}

	if expiresAt <= time.Now().UnixMilli() {
		return nil, false, false
	}

	return []byte(value), true, false
}

func (b *BaseAPI) setCache(key string, value []byte, ttl int) {
	database, err := db.GetDB()
	if err != nil {
		return
	}
	expiresAt := time.Now().Add(time.Duration(ttl) * time.Second).UnixMilli()
	createdAt := time.Now().UnixMilli()
	database.Exec(`
		INSERT INTO api_cache (key, value, expires_at, created_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at
	`, key, string(value), expiresAt, createdAt)
}

func CleanExpiredCache() {
	database, err := db.GetDB()
	if err != nil {
		return
	}
	database.Exec("DELETE FROM api_cache WHERE expires_at < ?", time.Now().UnixMilli())
}

func jsonUnmarshal(data []byte, target any) error {
	return json.Unmarshal(data, target)
}

func (b *BaseAPI) setCacheJSON(key string, value any, ttl int) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	b.setCache(key, data, ttl)
}

package middleware

import (
	"log"
	"net"
	"net/http"
	"strings"
)

// RequestLog 记录入口信息：Host 用于区分域名/端口入口，客户端 IP 优先取代理转发头。
func RequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[access] ip=%s host=%s xfh=%s method=%s uri=%s ua=%q",
			clientIP(r), r.Host, r.Header.Get("X-Forwarded-Host"), r.Method,
			r.URL.RequestURI(), r.UserAgent())
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	for _, h := range []string{"CF-Connecting-IP", "X-Real-IP"} {
		if v := strings.TrimSpace(r.Header.Get(h)); v != "" {
			return v
		}
	}
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		first, _, _ := strings.Cut(xff, ",")
		return strings.TrimSpace(first)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

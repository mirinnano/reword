package httpx

import (
	"context"
	"net"
	"net/http"
)

// ClientIPHeader is set by the Next.js Worker from Cloudflare's
// cf-connecting-ip after it strips every client-supplied IP header.
// It is trustworthy only because Caddy rejects requests that lack the
// Worker's X-Origin-Auth secret; nothing else (True-Client-IP, X-Real-IP,
// X-Forwarded-For) is ever honoured.
const ClientIPHeader = "X-Client-IP"

type clientIPKey struct{}

// ClientIP returns the caller's IP as resolved by the clientIP middleware.
func ClientIP(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey{}).(string)
	return ip
}

func clientIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := ""
		if h := r.Header.Get(ClientIPHeader); net.ParseIP(h) != nil {
			ip = h
		} else if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			ip = host
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey{}, ip)))
	})
}

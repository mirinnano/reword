package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{"remote addr fallback", nil, "192.0.2.10"},
		{"worker header", map[string]string{"X-Client-IP": "203.0.113.5"}, "203.0.113.5"},
		{"invalid worker header", map[string]string{"X-Client-IP": "not-an-ip"}, "192.0.2.10"},
		{
			"spoofable headers ignored",
			map[string]string{
				"True-Client-IP":  "6.6.6.6",
				"X-Real-IP":       "7.7.7.7",
				"X-Forwarded-For": "8.8.8.8",
			},
			"192.0.2.10",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			h := clientIP(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = ClientIP(r.Context())
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "192.0.2.10:54321"
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			if got != tt.want {
				t.Fatalf("ClientIP = %q, want %q", got, tt.want)
			}
		})
	}
}

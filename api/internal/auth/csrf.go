package auth

import (
	"net/http"

	"github.com/mirinnano/reword/api/internal/platform/httpx"
)

// SameOrigin rejects state-changing requests that do not come from the app's
// own origin (ADR 0001). Browsers always send Origin on cross-origin and on
// same-origin non-GET fetches; Sec-Fetch-Site is a fallback when it is absent.
// JSON-only bodies (httpx.DecodeJSON) additionally rule out HTML form posts.
func SameOrigin(appOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}
			origin := r.Header.Get("Origin")
			switch {
			case origin == appOrigin:
			case origin == "" && r.Header.Get("Sec-Fetch-Site") == "same-origin":
			default:
				httpx.WriteError(w, http.StatusForbidden, "cross_origin", "cross-origin request rejected")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

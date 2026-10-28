package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHasher(t *testing.T) {
	ctx := context.Background()
	h := NewHasher(1)
	enc, err := h.Hash(ctx, "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(enc, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("unexpected encoding %q", enc)
	}
	for _, tc := range []struct {
		pw, enc string
		want    bool
	}{
		{"correct horse battery", enc, true},
		{"wrong password!!", enc, false},
		{"dummy", "", false}, // unknown user path never matches
		{"anything at all", "", false},
	} {
		got, err := h.Verify(ctx, tc.pw, tc.enc)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("Verify(%q) = %v, want %v", tc.pw, got, tc.want)
		}
	}
	if _, err := h.Verify(ctx, "x", "$argon2id$garbage"); err == nil {
		t.Error("malformed hash accepted")
	}
}

func TestValidatePassword(t *testing.T) {
	for pw, ok := range map[string]bool{
		"short":      false,
		"          ": false,
		"ちょうど十文字のパスワード": true,
		"0123456789":             true,
		strings.Repeat("a", 128): true,
		strings.Repeat("a", 129): false,
	} {
		if got := ValidatePassword(pw) == nil; got != ok {
			t.Errorf("ValidatePassword(%q) ok=%v, want %v", pw, got, ok)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	for in, want := range map[string]string{
		"  Taro@Example.COM ":              "taro@example.com",
		"a+tag@sub.example.jp":             "a+tag@sub.example.jp",
		"Taro <taro@example.com>":          "",
		"no-at-sign":                       "",
		"user@localhost":                   "",
		strings.Repeat("a", 250) + "@x.jp": "",
	} {
		got, ok := NormalizeEmail(in)
		if (want == "") == ok || got != want {
			t.Errorf("NormalizeEmail(%q) = %q,%v want %q", in, got, ok, want)
		}
	}
}

func TestSameOrigin(t *testing.T) {
	const app = "https://app.example"
	h := SameOrigin(app)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }))
	for _, tc := range []struct {
		name, method, origin, site string
		want                       int
	}{
		{"get passes", "GET", "", "", 204},
		{"same origin post", "POST", app, "", 204},
		{"cross origin post", "POST", "https://evil.example", "cross-site", 403},
		{"no origin, same-origin fetch", "POST", "", "same-origin", 204},
		{"no origin, no fetch metadata", "POST", "", "", 403},
		{"null origin", "DELETE", "null", "", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.site != "" {
				req.Header.Set("Sec-Fetch-Site", tc.site)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status %d, want %d", rec.Code, tc.want)
			}
		})
	}
}

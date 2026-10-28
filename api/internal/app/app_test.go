package app_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mirinnano/reword/api/internal/app"
	"github.com/mirinnano/reword/api/internal/mail"
	"github.com/mirinnano/reword/api/internal/platform/db"
)

// outbox records messages instead of queueing them.
type outbox struct {
	mu   sync.Mutex
	msgs []mail.Message
}

func (o *outbox) Enqueue(_ context.Context, m mail.Message) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.msgs = append(o.msgs, m)
	return nil
}

func (o *outbox) last(t *testing.T, to string) mail.Message {
	t.Helper()
	o.mu.Lock()
	defer o.mu.Unlock()
	for i := len(o.msgs) - 1; i >= 0; i-- {
		if o.msgs[i].To == to {
			return o.msgs[i]
		}
	}
	t.Fatalf("no mail to %s", to)
	return mail.Message{}
}

func (o *outbox) count(to string) int {
	o.mu.Lock()
	defer o.mu.Unlock()
	n := 0
	for _, m := range o.msgs {
		if m.To == to {
			n++
		}
	}
	return n
}

var tokenRe = regexp.MustCompile(`#token=([A-Za-z0-9_-]+)`)

func token(t *testing.T, m mail.Message) string {
	t.Helper()
	got := tokenRe.FindStringSubmatch(m.Text)
	if got == nil {
		t.Fatalf("no token in mail %q", m.Subject)
	}
	return got[1]
}

type env struct {
	t      *testing.T
	srv    *httptest.Server
	db     *sql.DB
	outbox *outbox
}

const origin = "https://app.test"

func setup(t *testing.T) *env {
	t.Helper()
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN not set")
	}
	pool, err := db.Open(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close() })
	ob := &outbox{}
	a := app.New(app.Options{
		DB: pool, Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		AppOrigin: origin, CookieSecure: false, Outbox: ob,
	})
	srv := httptest.NewServer(a.Handler)
	t.Cleanup(srv.Close)
	return &env{t: t, srv: srv, db: pool, outbox: ob}
}

// client is one browser: its own cookie jar and client IP.
type client struct {
	e  *env
	hc *http.Client
	ip string
}

func (e *env) client(ip string) *client {
	jar, _ := cookiejar.New(nil)
	return &client{e: e, hc: &http.Client{Jar: jar}, ip: ip}
}

func (c *client) do(method, path string, body any) (int, map[string]any) {
	c.e.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.e.srv.URL+path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Origin", origin)
	req.Header.Set("X-Client-IP", c.ip)
	res, err := c.hc.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func (c *client) must(want int, method, path string, body any) map[string]any {
	c.e.t.Helper()
	got, out := c.do(method, path, body)
	if got != want {
		c.e.t.Fatalf("%s %s: status %d, want %d (%v)", method, path, got, want, out)
	}
	return out
}

func uniq(t *testing.T) (email, ip string) {
	n := time.Now().UnixNano()
	email = fmt.Sprintf("t%d@example.test", n)
	ip = fmt.Sprintf("10.%d.%d.%d", n%250, (n/250)%250, (n/62500)%250+1)
	return
}

func (e *env) cleanup(email string) {
	e.t.Cleanup(func() {
		e.db.Exec("DELETE FROM users WHERE email = ?", email)
		e.db.Exec("DELETE FROM pending_signups WHERE email = ?", email)
	})
}

func TestAccountLifecycle(t *testing.T) {
	e := setup(t)
	email, ip := uniq(t)
	e.cleanup(email)
	const pw = "first-password-123"
	browser := e.client(ip)

	// Unauthenticated
	browser.must(401, "GET", "/api/me", nil)

	// Signup → no account yet, verification mail sent
	browser.must(202, "POST", "/api/auth/signup", map[string]any{
		"email": strings.ToUpper(email), "password": pw, "display_name": "たろう",
		"target_grade_id": 6, "accept_terms": true,
	})
	browser.must(401, "POST", "/api/auth/login", map[string]any{"email": email, "password": pw})

	// Verify → account created and signed in
	me := browser.must(201, "POST", "/api/auth/signup/verify", map[string]any{"token": token(t, e.outbox.last(t, email))})
	if me["user"].(map[string]any)["email"] != email {
		t.Fatalf("me = %v", me)
	}
	profiles := me["profiles"].([]any)
	if len(profiles) != 1 || me["active_profile_id"] == nil {
		t.Fatalf("expected 1 active profile, got %v", me)
	}
	browser.must(200, "GET", "/api/me", nil)

	// Same token cannot be reused
	browser.must(400, "POST", "/api/auth/signup/verify", map[string]any{"token": token(t, e.outbox.last(t, email))})

	// Second signup for a registered address: same 202, "already registered" mail
	before := e.outbox.count(email)
	e.client(ip).must(202, "POST", "/api/auth/signup", map[string]any{
		"email": email, "password": "attacker-password-1", "display_name": "x", "accept_terms": true,
	})
	if e.outbox.count(email) != before+1 || !strings.Contains(e.outbox.last(t, email).Subject, "登録済み") {
		t.Fatal("expected already-registered mail")
	}

	// Profiles: create, rename, select, limit, delete
	p := browser.must(201, "POST", "/api/profiles", map[string]any{"display_name": "はなこ", "target_grade_id": 7})
	pid := int(p["id"].(float64))
	browser.must(200, "PATCH", fmt.Sprintf("/api/profiles/%d", pid), map[string]any{"display_name": "花子"})
	browser.must(200, "POST", fmt.Sprintf("/api/profiles/%d/select", pid), map[string]any{})
	if got := browser.must(200, "GET", "/api/me", nil)["active_profile_id"]; int(got.(float64)) != pid {
		t.Fatalf("active profile = %v, want %d", got, pid)
	}
	browser.must(400, "POST", "/api/profiles", map[string]any{"display_name": "x", "target_grade_id": 1}) // inactive grade
	for range 3 {
		browser.must(201, "POST", "/api/profiles", map[string]any{"display_name": "子"})
	}
	browser.must(409, "POST", "/api/profiles", map[string]any{"display_name": "6人目"})
	browser.must(204, "DELETE", fmt.Sprintf("/api/profiles/%d", pid), nil)
	if got := browser.must(200, "GET", "/api/me", nil)["active_profile_id"]; got != nil {
		t.Fatalf("deleted active profile still selected: %v", got)
	}

	// Another account cannot touch these profiles
	firstID := int(profiles[0].(map[string]any)["id"].(float64))
	other := otherAccount(t, e)
	other.must(404, "PATCH", fmt.Sprintf("/api/profiles/%d", firstID), map[string]any{"display_name": "盗む"})
	other.must(404, "POST", fmt.Sprintf("/api/profiles/%d/select", firstID), map[string]any{})

	// Logout ends the session
	browser.must(204, "POST", "/api/auth/logout", map[string]any{})
	browser.must(401, "GET", "/api/me", nil)

	// Login with a second device, then reset the password: all sessions end
	phone := e.client(ip)
	phone.must(200, "POST", "/api/auth/login", map[string]any{"email": email, "password": pw})
	browser.must(401, "POST", "/api/auth/login", map[string]any{"email": email, "password": "wrong-password-1"})
	browser.must(202, "POST", "/api/auth/password-reset", map[string]any{"email": email})
	reset := token(t, e.outbox.last(t, email))
	browser.must(400, "POST", "/api/auth/password-reset/confirm", map[string]any{"token": reset, "password": "short"})
	browser.must(204, "POST", "/api/auth/password-reset/confirm", map[string]any{"token": reset, "password": "second-password-456"})
	phone.must(401, "GET", "/api/me", nil)
	browser.must(400, "POST", "/api/auth/password-reset/confirm", map[string]any{"token": reset, "password": "third-password-789"})
	browser.must(401, "POST", "/api/auth/login", map[string]any{"email": email, "password": pw})
	browser.must(200, "POST", "/api/auth/login", map[string]any{"email": email, "password": "second-password-456"})
}

func otherAccount(t *testing.T, e *env) *client {
	email, ip := uniq(t)
	e.cleanup(email)
	c := e.client(ip)
	c.must(202, "POST", "/api/auth/signup", map[string]any{
		"email": email, "password": "other-password-1", "display_name": "他人", "accept_terms": true,
	})
	c.must(201, "POST", "/api/auth/signup/verify", map[string]any{"token": token(t, e.outbox.last(t, email))})
	return c
}

// An attacker who signs up first with the victim's address must not end up
// with a working password once the victim verifies their own signup.
func TestPreHijackingBlocked(t *testing.T) {
	e := setup(t)
	victim, ip := uniq(t)
	e.cleanup(victim)
	attacker := e.client(ip)
	attacker.must(202, "POST", "/api/auth/signup", map[string]any{
		"email": victim, "password": "attacker-password-1", "display_name": "x", "accept_terms": true,
	})
	v := e.client(ip)
	v.must(202, "POST", "/api/auth/signup", map[string]any{
		"email": victim, "password": "victim-password-12", "display_name": "本人", "accept_terms": true,
	})
	v.must(201, "POST", "/api/auth/signup/verify", map[string]any{"token": token(t, e.outbox.last(t, victim))})

	attacker.must(401, "POST", "/api/auth/login", map[string]any{"email": victim, "password": "attacker-password-1"})
	v.must(200, "POST", "/api/auth/login", map[string]any{"email": victim, "password": "victim-password-12"})
}

func TestRequestGuards(t *testing.T) {
	e := setup(t)
	_, ip := uniq(t)
	c := e.client(ip)

	// Cross-origin POST is rejected before anything else
	req, _ := http.NewRequest("POST", e.srv.URL+"/api/auth/login", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.test")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 403 {
		t.Fatalf("cross-origin status %d", res.StatusCode)
	}

	// Form-encoded bodies are refused (no CSRF via HTML forms)
	req, _ = http.NewRequest("POST", e.srv.URL+"/api/auth/login", strings.NewReader("email=a&password=b"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", origin)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 415 {
		t.Fatalf("form post status %d", res.StatusCode)
	}

	// Unknown fields are rejected
	c.must(400, "POST", "/api/auth/login", map[string]any{"email": "a@b.jp", "password": "x", "role": "admin"})

	// Login is rate limited per email
	email := fmt.Sprintf("rl%d@example.test", time.Now().UnixNano())
	var last int
	for range 12 {
		last, _ = e.client(ip).do("POST", "/api/auth/login", map[string]any{"email": email, "password": "nope-nope-nope"})
	}
	if last != 429 {
		t.Fatalf("expected 429 after repeated logins, got %d", last)
	}

	// Responses are not cacheable
	res, err = http.Get(e.srv.URL + "/api/grades")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", res.Header.Get("Cache-Control"))
	}
}

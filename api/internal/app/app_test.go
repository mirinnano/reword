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

func items(t *testing.T, session map[string]any) []map[string]any {
	t.Helper()
	raw := session["items"].([]any)
	out := make([]map[string]any, len(raw))
	for i, it := range raw {
		out[i] = it.(map[string]any)
	}
	return out
}

func TestPracticeSession(t *testing.T) {
	e := setup(t)
	c := otherAccount(t, e)

	types := c.must(200, "GET", "/api/practice/types", nil)["types"].([]any)
	if len(types) == 0 {
		t.Skip("no content imported in this database")
	}
	c.must(400, "POST", "/api/practice/sessions", map[string]any{"type": "nope"})
	c.must(400, "POST", "/api/practice/sessions", map[string]any{"type": "g2-q1", "count": 31})

	s := c.must(201, "POST", "/api/practice/sessions", map[string]any{"type": "g2-q1", "count": 5})
	id := int(s["id"].(float64))
	first := items(t, s)
	if len(first) != 5 {
		t.Fatalf("want 5 items, got %d", len(first))
	}
	for _, it := range first {
		if it["feedback"] != nil {
			t.Fatalf("unanswered item leaks feedback: %v", it)
		}
		if len(it["choices"].([]any)) != 4 || !strings.Contains(it["stem"].(string), "( )") {
			t.Fatalf("bad item %v", it)
		}
	}

	path := fmt.Sprintf("/api/practice/sessions/%d", id)
	correct := 0
	wasCorrect := map[string]bool{}
	for i := range 5 {
		res := c.must(200, "POST", path+"/answers", map[string]any{"position": i + 1, "choice": i % 4, "time_ms": 3000})
		fb := res["feedback"].(map[string]any)
		cc := int(fb["correct_choice"].(float64))
		if fb["correct"].(bool) != (cc == i%4) || int(fb["selected"].(float64)) != i%4 {
			t.Fatalf("inconsistent feedback %v", fb)
		}
		if fb["explanation_ja"] == "" || len(fb["vocab"].([]any)) == 0 {
			t.Fatalf("feedback lacks explanation or vocab: %v", fb)
		}
		if fb["correct"].(bool) {
			correct++
		}
		wasCorrect[first[i]["stem"].(string)] = fb["correct"].(bool)
		p := res["progress"].(map[string]any)
		if int(p["answered"].(float64)) != i+1 || p["finished"].(bool) != (i == 4) {
			t.Fatalf("progress after %d answers: %v", i+1, p)
		}
		if i == 0 {
			c.must(409, "POST", path+"/answers", map[string]any{"position": 1, "choice": 0})
		}
	}
	c.must(409, "POST", path+"/answers", map[string]any{"position": 1, "choice": 0}) // finished

	got := c.must(200, "GET", path, nil)
	if p := got["progress"].(map[string]any); int(p["correct"].(float64)) != correct || !p["finished"].(bool) {
		t.Fatalf("summary %v, want %d correct", p, correct)
	}
	for _, it := range items(t, got) {
		if it["feedback"] == nil {
			t.Fatal("answered item missing feedback")
		}
	}

	// Progress counts the five answered questions as seen; missed ones are due now.
	for _, p := range c.must(200, "GET", "/api/practice/progress", nil)["progress"].([]any) {
		pv := p.(map[string]any)
		if pv["type_code"] != "g2-q1" {
			continue
		}
		if int(pv["seen"].(float64)) != 5 || int(pv["mastered"].(float64)) != 0 || int(pv["due"].(float64)) != 5-correct {
			t.Fatalf("progress %v, want seen 5, mastered 0, due %d", pv, 5-correct)
		}
	}

	// Another account cannot see or answer this session
	stranger := otherAccount(t, e)
	stranger.must(404, "GET", path, nil)
	stranger.must(404, "POST", path+"/answers", map[string]any{"position": 1, "choice": 0})

	// The next session never repeats a correctly answered question while
	// unseen ones remain; missed ones may come back, up to 30% (2 of 5).
	repeats := 0
	for _, it := range items(t, c.must(201, "POST", "/api/practice/sessions", map[string]any{"type": "g2-q1", "count": 5})) {
		ok, seen := wasCorrect[it["stem"].(string)]
		if seen && ok {
			t.Fatalf("repeated a correctly answered question: %q", it["stem"])
		}
		if seen {
			repeats++
		}
	}
	if missed := 5 - correct; repeats > min(missed, 2) {
		t.Fatalf("%d missed questions came back, want at most %d", repeats, min(missed, 2))
	}

	// Ending early
	early := c.must(201, "POST", "/api/practice/sessions", map[string]any{"type": "g2-q1", "count": 3})
	epath := fmt.Sprintf("/api/practice/sessions/%d", int(early["id"].(float64)))
	c.must(200, "POST", epath+"/answers", map[string]any{"position": 1, "choice": 2})
	if p := c.must(200, "POST", epath+"/finish", map[string]any{})["progress"].(map[string]any); !p["finished"].(bool) || int(p["answered"].(float64)) != 1 {
		t.Fatalf("early finish progress %v", p)
	}

	if n := len(c.must(200, "GET", "/api/practice/sessions", nil)["sessions"].([]any)); n != 3 {
		t.Fatalf("want 3 recent sessions, got %d", n)
	}
}

// Passage sessions present whole passages in order and count in passages.
func TestPassageSession(t *testing.T) {
	e := setup(t)
	for _, tt := range []struct {
		code      string
		wantBlank bool
	}{{"g2-q2", true}, {"g2-q3", false}} {
		t.Run(tt.code, func(t *testing.T) {
			c := otherAccount(t, e)
			c.must(400, "POST", "/api/practice/sessions", map[string]any{"type": tt.code, "count": 6})
			s := c.must(201, "POST", "/api/practice/sessions", map[string]any{"type": tt.code, "count": 2})
			its := items(t, s)

			groups := map[string]bool{}
			last, seq := "", 0
			for i, it := range its {
				g, _ := it["group"].(string)
				if g == "" || it["passage"] == "" || it["title"] == "" || it["feedback"] != nil {
					t.Fatalf("bad item %v", it)
				}
				if g != last {
					if groups[g] {
						t.Fatalf("passage %s is split in the session", g)
					}
					groups[g], last, seq = true, g, 0
				}
				seq++
				if int(it["seq"].(float64)) != seq || int(it["position"].(float64)) != i+1 {
					t.Fatalf("item out of order: %v", it)
				}
				if stem, _ := it["stem"].(string); (stem == "") != tt.wantBlank {
					t.Fatalf("stem %q for %s", stem, tt.code)
				}
				passage := it["passage"].(string)
				if marker := fmt.Sprintf("( %d )", seq); tt.wantBlank && !strings.Contains(passage, marker) {
					t.Fatalf("passage has no %s", marker)
				}
			}
			if len(groups) != 2 {
				t.Fatalf("want 2 passages, got %d", len(groups))
			}

			path := fmt.Sprintf("/api/practice/sessions/%d", int(s["id"].(float64)))
			res := c.must(200, "POST", path+"/answers", map[string]any{"position": 1, "choice": 0, "time_ms": 1000})
			if res["feedback"].(map[string]any)["explanation_ja"] == "" {
				t.Fatalf("feedback lacks an explanation: %v", res)
			}
		})
	}
}

// Today's routine is one session: starting twice lands on the same one, and
// the home screen's plan is what starting creates.
func TestDailyRoutine(t *testing.T) {
	e := setup(t)
	c := otherAccount(t, e)

	plan := c.must(200, "GET", "/api/practice/daily", nil)
	if plan["session"] != nil || int(plan["number"].(float64)) != 1 {
		t.Fatalf("plan before starting: %v", plan)
	}
	s := c.must(201, "POST", "/api/practice/daily", map[string]any{})
	if s["format"] != "daily" || s["type_code"] != "daily" {
		t.Fatalf("session = %v", s)
	}
	its := items(t, s)
	if len(its) != 12 {
		t.Fatalf("want 12 items, got %d", len(its))
	}
	if got := int(plan["reviews"].(float64)) + int(plan["new"].(float64)); got != 12 {
		t.Fatalf("plan adds up to %d, want 12: %v", got, plan)
	}
	formats := map[string]bool{}
	for _, it := range its {
		f, _ := it["format"].(string)
		if f == "" || f == "vocab_word" {
			t.Fatalf("item format %q: %v", f, it)
		}
		formats[f] = true
	}
	if len(formats) < 2 && int(plan["new"].(float64)) == 12 {
		t.Logf("one format only in this draw: %v", formats)
	}

	again := c.must(200, "POST", "/api/practice/daily", map[string]any{})
	if again["id"] != s["id"] {
		t.Fatalf("second start made session %v, want %v", again["id"], s["id"])
	}
	path := fmt.Sprintf("/api/practice/sessions/%d", int(s["id"].(float64)))
	c.must(200, "POST", path+"/answers", map[string]any{"position": 1, "choice": 0, "time_ms": 1000})

	status := c.must(200, "GET", "/api/practice/daily", nil)
	p := status["session"].(map[string]any)
	if int(p["answered"].(float64)) != 1 || status["session_id"] != s["id"] {
		t.Fatalf("status after one answer: %v", status)
	}

	recent := c.must(200, "GET", "/api/practice/sessions", nil)["sessions"].([]any)
	if r := recent[0].(map[string]any); r["type_name_ja"] != "今日の日課" {
		t.Fatalf("recent session = %v", r)
	}
}

func TestPlacement(t *testing.T) {
	e := setup(t)
	c := otherAccount(t, e)

	status := c.must(200, "GET", "/api/practice/placement", nil)
	if status["stage"] != nil {
		t.Fatalf("stage before the test: %v", status)
	}
	s := c.must(201, "POST", "/api/practice/placement", map[string]any{})
	if s["format"] != "placement" || len(items(t, s)) != 30 {
		t.Fatalf("session = %v items", s["format"])
	}
	again := c.must(200, "POST", "/api/practice/placement", map[string]any{})
	if again["id"] != s["id"] {
		t.Fatalf("second start made session %v, want %v", again["id"], s["id"])
	}
	path := fmt.Sprintf("/api/practice/sessions/%d", int(s["id"].(float64)))

	// "Don't know" all the way: the test stops at the end of the first block.
	var last map[string]any
	for pos := 1; pos <= 10; pos++ {
		last = c.must(200, "POST", path+"/answers", map[string]any{"position": pos, "dont_know": true, "time_ms": 1000})
		if last["feedback"] != nil {
			t.Fatalf("a placement answer must not be marked: %v", last)
		}
	}
	if p := last["progress"].(map[string]any); p["finished"] != true || int(p["answered"].(float64)) != 10 {
		t.Fatalf("progress after block 1: %v", p)
	}
	c.must(409, "POST", path+"/answers", map[string]any{"position": 11, "dont_know": true})

	status = c.must(200, "GET", "/api/practice/placement", nil)
	if status["stage"] != float64(0) || status["session_id"] != nil {
		t.Fatalf("status after the test: %v", status)
	}
	view := c.must(200, "GET", path, nil)
	if b := view["placement"].(map[string]any)["blocks"].([]any)[0].(map[string]any); b["dont_know"] != float64(10) {
		t.Fatalf("first block = %v", b)
	}

	// Stage 0: no passage on any day, new questions from 準2級, and the test
	// itself left no review behind.
	plan := c.must(200, "GET", "/api/practice/daily", nil)
	if plan["reviews"] != float64(0) || plan["passage"] != false {
		t.Fatalf("plan after the test: %v", plan)
	}
	for _, r := range c.must(200, "GET", "/api/practice/sessions", nil)["sessions"].([]any) {
		if r.(map[string]any)["type_code"] == "placement" {
			t.Fatalf("placement listed in history: %v", r)
		}
	}
	daily := c.must(201, "POST", "/api/practice/daily", map[string]any{})
	if got := len(items(t, daily)); got != 12 {
		t.Fatalf("routine has %d items", got)
	}

	// Other sessions reject "don't know".
	dpath := fmt.Sprintf("/api/practice/sessions/%d", int(daily["id"].(float64)))
	c.must(400, "POST", dpath+"/answers", map[string]any{"position": 1, "dont_know": true})
}

// TestGrammarUnit needs imported units, and CI imports content/ before the tests.
func TestGrammarUnit(t *testing.T) {
	e := setup(t)
	c := otherAccount(t, e)

	list := c.must(200, "GET", "/api/grammar/units", nil)
	units := list["units"].([]any)
	if len(units) == 0 {
		t.Skip("no grammar units imported")
	}
	key := units[0].(map[string]any)["key"].(string)
	if list["next"] != key {
		t.Fatalf("next = %v, want %v", list["next"], key)
	}
	unit := c.must(200, "GET", "/api/grammar/units/"+key, nil)
	if unit["rule_ja"] == "" || len(unit["examples"].([]any)) != 3 {
		t.Fatalf("unit = %v", unit)
	}
	c.must(404, "GET", "/api/grammar/units/g2-gr-nothing", nil)

	s := c.must(201, "POST", "/api/grammar/units/"+key+"/sessions", map[string]any{})
	if s["type_code"] != "g2-gr" || len(items(t, s)) != 6 {
		t.Fatalf("session = %v", s["type_code"])
	}
	path := fmt.Sprintf("/api/practice/sessions/%d", int(s["id"].(float64)))
	for pos := 1; pos <= 6; pos++ {
		c.must(200, "POST", path+"/answers", map[string]any{"position": pos, "choice": 0, "time_ms": 1000})
	}
	first := c.must(200, "GET", "/api/grammar/units", nil)["units"].([]any)[0].(map[string]any)
	if first["best"] == nil {
		t.Fatalf("a finished sitting was not recorded: %v", first)
	}
	for _, ty := range c.must(200, "GET", "/api/practice/types", nil)["types"].([]any) {
		if ty.(map[string]any)["code"] == "g2-gr" {
			t.Fatal("grammar must not be listed as a practice format")
		}
	}
	// Missed questions return in the routine, and no unseen grammar joins it.
	daily := c.must(201, "POST", "/api/practice/daily", map[string]any{})
	if len(items(t, daily)) != 12 {
		t.Fatalf("routine items = %d", len(items(t, daily)))
	}
}

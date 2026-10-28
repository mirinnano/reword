package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// argon2id parameters: OWASP minimum (19 MiB, t=2, p=1). Memory per hash is
// what bounds concurrency on the 1 GB staging VM, hence the semaphore.
const (
	argonMemory  = 19 * 1024
	argonTime    = 2
	argonThreads = 1
	argonKeyLen  = 32
	argonSaltLen = 16

	MinPasswordLen = 10
	MaxPasswordLen = 128
)

var ErrPasswordPolicy = fmt.Errorf("password must be %d to %d characters", MinPasswordLen, MaxPasswordLen)

func ValidatePassword(pw string) error {
	n := utf8.RuneCountInString(pw)
	if n < MinPasswordLen || n > MaxPasswordLen || strings.TrimSpace(pw) == "" {
		return ErrPasswordPolicy
	}
	return nil
}

type Hasher struct {
	sem   chan struct{}
	dummy string
}

func NewHasher(concurrency int) *Hasher {
	h := &Hasher{sem: make(chan struct{}, concurrency)}
	// Verifying against a dummy hash when the user does not exist keeps
	// login timing the same for known and unknown emails.
	h.dummy = encode(make([]byte, argonSaltLen), argon2.IDKey([]byte("dummy"), make([]byte, argonSaltLen), argonTime, argonMemory, argonThreads, argonKeyLen))
	return h
}

func (h *Hasher) acquire(ctx context.Context) error {
	select {
	case h.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *Hasher) release() { <-h.sem }

func (h *Hasher) Hash(ctx context.Context, pw string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	if err := h.acquire(ctx); err != nil {
		return "", err
	}
	defer h.release()
	return encode(salt, argon2.IDKey([]byte(pw), salt, argonTime, argonMemory, argonThreads, argonKeyLen)), nil
}

// Verify reports whether pw matches encoded. An empty encoded value checks
// against the dummy hash and always returns false.
func (h *Hasher) Verify(ctx context.Context, pw, encoded string) (bool, error) {
	if encoded == "" {
		encoded = h.dummy
		pw = "\x00" + pw
	}
	p, salt, want, err := decode(encoded)
	if err != nil {
		return false, err
	}
	if err := h.acquire(ctx); err != nil {
		return false, err
	}
	defer h.release()
	got := argon2.IDKey([]byte(pw), salt, p.time, p.memory, p.threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

type params struct {
	memory  uint32
	time    uint32
	threads uint8
}

var b64 = base64.RawStdEncoding

// encode produces a PHC string: $argon2id$v=19$m=...,t=...,p=...$salt$hash
func encode(salt, key []byte) string {
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, b64.EncodeToString(salt), b64.EncodeToString(key))
}

var errBadHash = errors.New("malformed password hash")

func decode(s string) (params, []byte, []byte, error) {
	parts := strings.Split(s, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return params{}, nil, nil, errBadHash
	}
	var v int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &v); err != nil || v != argon2.Version {
		return params{}, nil, nil, errBadHash
	}
	var p params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads); err != nil {
		return params{}, nil, nil, errBadHash
	}
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return params{}, nil, nil, errBadHash
	}
	key, err := b64.DecodeString(parts[5])
	if err != nil || len(key) == 0 {
		return params{}, nil, nil, errBadHash
	}
	return p, salt, key, nil
}

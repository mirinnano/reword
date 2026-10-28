package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// NewToken returns a 256-bit random token for the client and its SHA-256
// for storage. Only the hash ever reaches the database.
func NewToken() (plain string, hash []byte, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	return plain, HashToken(plain), nil
}

func HashToken(plain string) []byte {
	sum := sha256.Sum256([]byte(plain))
	return sum[:]
}

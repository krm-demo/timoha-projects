// Package auth hashes passwords (PBKDF2-SHA256) and issues/verifies
// HS256 JSON Web Tokens, using only the standard library.
package auth

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const iterations = 210_000 // OWASP 2023 recommendation for PBKDF2-HMAC-SHA256

var b64 = base64.RawURLEncoding

// HashPassword returns "pbkdf2-sha256$<iter>$<salt>$<key>".
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, iterations, 32)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", iterations, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// CheckPassword verifies a password against a stored hash in constant time.
func CheckPassword(password, stored string) bool {
	parts := strings.Split(stored, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	iter, err1 := strconv.Atoi(parts[1])
	salt, err2 := b64.DecodeString(parts[2])
	want, err3 := b64.DecodeString(parts[3])
	if err1 != nil || err2 != nil || err3 != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iter, len(want))
	return err == nil && subtle.ConstantTimeCompare(got, want) == 1
}

// Claims are the token payload.
type Claims struct {
	UserID int64  `json:"sub"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Exp    int64  `json:"exp"`
}

// Tokens signs and verifies JWTs with one HMAC secret.
type Tokens struct {
	Secret []byte
	TTL    time.Duration
	Now    func() time.Time
}

var ErrInvalidToken = errors.New("invalid or expired token")

var header = b64.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

func (t Tokens) now() time.Time {
	if t.Now != nil {
		return t.Now()
	}
	return time.Now()
}

func (t Tokens) sign(data string) string {
	m := hmac.New(sha256.New, t.Secret)
	m.Write([]byte(data))
	return b64.EncodeToString(m.Sum(nil))
}

// Issue creates a token for the user that expires after TTL.
func (t Tokens) Issue(userID int64, email, name string) (string, error) {
	ttl := t.TTL
	if ttl == 0 {
		ttl = 7 * 24 * time.Hour
	}
	payload, err := json.Marshal(Claims{userID, email, name, t.now().Add(ttl).Unix()})
	if err != nil {
		return "", err
	}
	data := header + "." + b64.EncodeToString(payload)
	return data + "." + t.sign(data), nil
}

// Verify checks the signature and expiry and returns the claims.
func (t Tokens) Verify(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	// We only ever issue one header ({"alg":"HS256"}), so we require exactly
	// that. This blocks the classic JWT attacks where a forged token says
	// "alg":"none" or switches algorithms.
	if len(parts) != 3 || parts[0] != header {
		return Claims{}, ErrInvalidToken
	}
	// hmac.Equal compares in constant time: a normal == would return
	// earlier for a wrong first byte, and that timing difference can be used
	// to guess a valid signature byte by byte.
	if !hmac.Equal([]byte(parts[2]), []byte(t.sign(parts[0]+"."+parts[1]))) {
		return Claims{}, ErrInvalidToken
	}
	raw, err := b64.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var c Claims
	if json.Unmarshal(raw, &c) != nil || c.UserID == 0 || t.now().Unix() >= c.Exp {
		return Claims{}, ErrInvalidToken
	}
	return c, nil
}

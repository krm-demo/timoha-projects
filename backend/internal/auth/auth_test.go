package auth

import (
	"testing"
	"time"
)

func TestPasswordHashing(t *testing.T) {
	h, err := HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword("secret123", h) || CheckPassword("secret124", h) || CheckPassword("secret123", "garbage") {
		t.Fatal("password check misbehaves")
	}
	if h2, _ := HashPassword("secret123"); h2 == h {
		t.Fatal("hashes must be salted")
	}
}

func TestTokens(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	tk := Tokens{Secret: []byte("k"), TTL: time.Hour, Now: func() time.Time { return now }}
	tok, _ := tk.Issue(7, "a@b.c", "Tim")
	c, err := tk.Verify(tok)
	if err != nil || c.UserID != 7 || c.Name != "Tim" {
		t.Fatalf("verify: %v %+v", err, c)
	}
	if _, err := (Tokens{Secret: []byte("other")}).Verify(tok); err == nil {
		t.Fatal("token signed with another secret must fail")
	}
	now = now.Add(2 * time.Hour)
	if _, err := tk.Verify(tok); err == nil {
		t.Fatal("expired token must fail")
	}
}

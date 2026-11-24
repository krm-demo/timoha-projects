package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestFileStoreContract(t *testing.T) {
	s, err := OpenFile("")
	if err != nil {
		t.Fatal(err)
	}
	runContract(t, s)
}

func TestFileStorePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.json")
	s, err := OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateUser(context.Background(), "a@b.c", "A", "h"); err != nil {
		t.Fatal(err)
	}
	s2, err := OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s2.UserByEmail(context.Background(), "a@b.c"); err != nil {
		t.Fatalf("user not persisted: %v", err)
	}
}

func TestKeysAndFee(t *testing.T) {
	k := NewKey()
	if len(k) != 17 || k[5] != '-' || k[11] != '-' {
		t.Fatalf("bad key %q", k)
	}
	if ServiceFee(1999) != 100 || ServiceFee(1000) != 50 {
		t.Fatalf("fee: %v %v", ServiceFee(1999), ServiceFee(1000))
	}
}

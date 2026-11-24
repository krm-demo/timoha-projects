//go:build postgres

package store

import (
	"context"
	"os"
	"testing"
)

// Runs only with -tags postgres and TEST_DATABASE_URL set (CI starts a
// postgres service container for this).
func TestPGStoreContract(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	s, err := openPG(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, q := range []string{`TRUNCATE orders, users RESTART IDENTITY CASCADE`} {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	runContract(t, s)
}

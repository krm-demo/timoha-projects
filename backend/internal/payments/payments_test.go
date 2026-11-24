package payments

import (
	"errors"
	"testing"
	"time"
)

func TestCharge(t *testing.T) {
	now := time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC)
	ok := Card{"4242 4242 4242 4242", "12/29", "123"}
	if id, err := Charge(ok, 1000, now); err != nil || id == "" {
		t.Fatalf("valid card: %v", err)
	}
	cases := []struct {
		card Card
		want error
	}{
		{Card{"4000 0000 0000 0002", "12/29", "123"}, ErrDeclined},
		{Card{"4000 0000 0000 9995", "12/29", "123"}, ErrInsufficientFunds},
	}
	for _, c := range cases {
		if _, err := Charge(c.card, 1000, now); !errors.Is(err, c.want) {
			t.Errorf("%v: got %v want %v", c.card.Number, err, c.want)
		}
	}
	for _, bad := range []Card{
		{"4242 4242 4242 4241", "12/29", "123"}, // luhn
		{"4242 4242 4242 4242", "10/26", "123"}, // expired
		{"4242 4242 4242 4242", "11/26", "12"},  // cvc
	} {
		var ve ValidationError
		if _, err := Charge(bad, 1000, now); !errors.As(err, &ve) {
			t.Errorf("%+v: want validation error, got %v", bad, err)
		}
	}
	if _, err := Charge(Card{"4242 4242 4242 4242", "11/26", "123"}, 1000, now); err != nil {
		t.Errorf("card expiring this month is still valid: %v", err)
	}
}

// Package payments simulates a card processor in "test mode", modelled on
// Stripe's test cards. No real money is ever involved.
package payments

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/krm-demo/timoha-projects/backend/internal/catalog"
)

// Card is what the checkout form sends.
type Card struct {
	Number string `json:"number"`
	Expiry string `json:"expiry"` // MM/YY
	CVC    string `json:"cvc"`
}

// ValidationError is a problem with the card details (HTTP 400).
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// ErrDeclined is a bank decline (HTTP 402).
var ErrDeclined = errors.New("payment declined by the bank (test card)")

// ErrInsufficientFunds is a decline with a specific reason (HTTP 402).
var ErrInsufficientFunds = errors.New("insufficient funds (test card)")

// Charge validates the card and "charges" amount. It returns a payment id.
func Charge(c Card, amount catalog.Money, now time.Time) (string, error) {
	num := strings.ReplaceAll(strings.ReplaceAll(c.Number, " ", ""), "-", "")
	if len(num) != 16 || !luhn(num) {
		return "", ValidationError{"card number is invalid"}
	}
	if !expiryOK(c.Expiry, now) {
		return "", ValidationError{"card expiry must be a future date in MM/YY format"}
	}
	if l := len(c.CVC); l < 3 || l > 4 || strings.Trim(c.CVC, "0123456789") != "" {
		return "", ValidationError{"CVC must be 3 or 4 digits"}
	}
	if amount <= 0 {
		return "", ValidationError{"nothing to pay"}
	}
	switch num {
	case "4000000000000002":
		return "", ErrDeclined
	case "4000000000009995":
		return "", ErrInsufficientFunds
	}
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "pay_test_" + hex.EncodeToString(b), nil
}

func luhn(s string) bool {
	sum := 0
	for i := len(s) - 1; i >= 0; i-- {
		d := int(s[i] - '0')
		if d < 0 || d > 9 {
			return false
		}
		if (len(s)-i)%2 == 0 {
			if d *= 2; d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return sum%10 == 0
}

func expiryOK(exp string, now time.Time) bool {
	var mm, yy int
	if len(exp) != 5 || exp[2] != '/' {
		return false
	}
	mm, err1 := strconv.Atoi(exp[:2])
	yy, err2 := strconv.Atoi(exp[3:])
	if err1 != nil || err2 != nil || mm < 1 || mm > 12 {
		return false
	}
	// valid through the last day of the expiry month
	end := time.Date(2000+yy, time.Month(mm)+1, 1, 0, 0, 0, 0, time.UTC)
	return now.Before(end)
}

// Describe is used in logs.
func Describe(c Card) string {
	n := strings.ReplaceAll(c.Number, " ", "")
	if len(n) < 4 {
		return "card"
	}
	return fmt.Sprintf("card ****%s", n[len(n)-4:])
}

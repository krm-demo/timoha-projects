package api

import (
	"net/http/httptest"
	"testing"
)

// Regression: 19.99*100 is 1998.9999… in float64; truncating it would make
// "max=19.99" exclude a product that costs exactly €19.99.
func TestParseQueryRoundsPrices(t *testing.T) {
	q, err := parseQuery(httptest.NewRequest("GET", "/api/products?min=0.29&max=19.99", nil))
	if err != nil {
		t.Fatal(err)
	}
	if *q.Min != 29 || *q.Max != 1999 {
		t.Fatalf("min=%d max=%d, want 29 and 1999 cents", *q.Min, *q.Max)
	}
}

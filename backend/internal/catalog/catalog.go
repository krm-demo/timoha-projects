// Package catalog holds the marketplace domain model and the pure functions
// that work on it: filtering, faceting, sorting and fuzzy search.
package catalog

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
)

// Money is an amount in euro cents. It is serialised as a decimal number
// (e.g. 19.99) so the frontend never deals with cents.
type Money int64

func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatFloat(float64(m)/100, 'f', 2, 64)), nil
}

func (m *Money) UnmarshalJSON(b []byte) error {
	var f float64
	if err := json.Unmarshal(b, &f); err != nil {
		return err
	}
	*m = Money(math.Round(f * 100))
	return nil
}

// Offer is one seller's listing of a product.
type Offer struct {
	Seller   string  `json:"seller"`
	Rating   float64 `json:"rating"`
	Reviews  int     `json:"reviews"`
	Price    Money   `json:"price"`
	Stock    int     `json:"stock"`
	Delivery string  `json:"delivery"`
}

// Product is a sellable item (game key, gift card, subscription, DLC).
type Product struct {
	ID          int      `json:"id"`
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Platform    string   `json:"platform"`
	Region      string   `json:"region"`
	Type        string   `json:"type"`
	Genre       string   `json:"genre"`
	Year        int      `json:"year"`
	MSRP        Money    `json:"msrp"`
	Colors      []string `json:"colors"`
	Rating      float64  `json:"rating"`
	Sold        int      `json:"sold"`
	Description string   `json:"description"`
	Offers      []Offer  `json:"offers"`
}

// SortOffers puts in-stock offers first, cheapest first.
func (p *Product) SortOffers() {
	sort.SliceStable(p.Offers, func(i, j int) bool {
		a, b := p.Offers[i], p.Offers[j]
		if (a.Stock > 0) != (b.Stock > 0) {
			return a.Stock > 0
		}
		return a.Price < b.Price
	})
}

// Best returns the cheapest in-stock offer, or false if the product is sold out.
func (p Product) Best() (Offer, bool) {
	var best Offer
	found := false
	for _, o := range p.Offers {
		if o.Stock > 0 && (!found || o.Price < best.Price) {
			best, found = o, true
		}
	}
	return best, found
}

// FromPrice is the price shown on cards: the best in-stock offer, else MSRP.
func (p Product) FromPrice() Money {
	if o, ok := p.Best(); ok {
		return o.Price
	}
	return p.MSRP
}

// Discount is the best offer's discount against MSRP, in whole percent.
func (p Product) Discount() int {
	if p.MSRP <= 0 {
		return 0
	}
	return int(math.Round((1 - float64(p.FromPrice())/float64(p.MSRP)) * 100))
}

// Offer finds a seller's offer on this product.
func (p Product) Offer(seller string) (Offer, int, bool) {
	for i, o := range p.Offers {
		if o.Seller == seller {
			return o, i, true
		}
	}
	return Offer{}, -1, false
}

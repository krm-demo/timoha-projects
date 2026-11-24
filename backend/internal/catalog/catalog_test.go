package catalog

import "testing"

func sample() []Product {
	mk := func(id int, name, platform string, sold int, msrp, price Money) Product {
		return Product{ID: id, Name: name, Platform: platform, Region: "Global", Type: "Game", Genre: "RPG",
			Sold: sold, MSRP: msrp, Offers: []Offer{{Seller: "A", Price: price, Stock: 5}}}
	}
	return []Product{
		mk(1, "Elden Ring", "Steam", 100, 5999, 1999),
		mk(2, "Elden Ring Nightreign", "Steam", 50, 3999, 1499),
		mk(3, "Baldur's Gate 3", "Steam", 300, 5999, 2599),
		mk(4, "Forza Horizon 5", "Xbox", 200, 5999, 2499),
	}
}

func TestFuzzySearch(t *testing.T) {
	cases := map[string]string{
		"elden ring": "Elden Ring", // exact
		"eldn rng":   "Elden Ring", // typos in both words
		"baldurs":    "Baldur's Gate 3",
		"forza":      "Forza Horizon 5", // partial name
		"horizn 5":   "Forza Horizon 5",
	}
	for q, want := range cases {
		got := Search(sample(), q)
		if len(got) == 0 || got[0].Name != want {
			t.Errorf("Search(%q): want %q first, got %v", q, want, names(got))
		}
	}
	if got := Search(sample(), "zelda"); len(got) != 0 {
		t.Errorf("Search(zelda) should be empty, got %v", names(got))
	}
}

func TestListFiltersSortsAndFacets(t *testing.T) {
	max := Money(2000)
	r := List(sample(), Query{Platforms: []string{"Steam"}, Max: &max, Sort: "price-asc"})
	if r.Total != 2 || r.Items[0].Name != "Elden Ring Nightreign" {
		t.Fatalf("unexpected result %v total=%d", names(r.Items), r.Total)
	}
	// platform facet ignores the platform filter itself but respects the price filter
	for _, f := range r.Facets["platform"] {
		if f.Value == "Xbox" && f.Count != 0 {
			t.Errorf("Xbox count = %d, want 0 (price filter)", f.Count)
		}
		if f.Value == "Steam" && f.Count != 2 {
			t.Errorf("Steam count = %d, want 2", f.Count)
		}
	}
}

func TestPaginationAndDefaults(t *testing.T) {
	r := List(sample(), Query{Limit: 3, Page: 2})
	if r.Sort != "popular" || r.Total != 4 || len(r.Items) != 1 || r.Items[0].Name != "Elden Ring Nightreign" {
		t.Fatalf("got sort=%s total=%d items=%v", r.Sort, r.Total, names(r.Items))
	}
	if r := List(sample(), Query{Q: "elden"}); r.Sort != "relevance" {
		t.Errorf("search should default to relevance, got %s", r.Sort)
	}
}

func TestBestSkipsOutOfStock(t *testing.T) {
	p := Product{MSRP: 1000, Offers: []Offer{{Seller: "cheap", Price: 100, Stock: 0}, {Seller: "ok", Price: 500, Stock: 1}}}
	if o, _ := p.Best(); o.Seller != "ok" || p.Discount() != 50 {
		t.Errorf("best=%v discount=%d", o, p.Discount())
	}
}

func names(ps []Product) []string {
	var out []string
	for _, p := range ps {
		out = append(out, p.Name)
	}
	return out
}

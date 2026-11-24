package catalog

import (
	"slices"
	"sort"
)

// Query describes a catalog listing request.
type Query struct {
	Q         string
	Platforms []string
	Regions   []string
	Types     []string
	Genres    []string
	Min, Max  *Money
	Sort      string // relevance, popular, price-asc, price-desc, discount, newest
	Page      int    // 1-based
	Limit     int
}

// FacetCount is one filter value and how many products it would yield.
type FacetCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// Result is one page of products plus facet counts for the filter sidebar.
type Result struct {
	Items  []Product               `json:"items"`
	Total  int                     `json:"total"`
	Page   int                     `json:"page"`
	Limit  int                     `json:"limit"`
	Sort   string                  `json:"sort"`
	Facets map[string][]FacetCount `json:"facets"`
}

var Sorts = []string{"relevance", "popular", "price-asc", "price-desc", "discount", "newest"}

type facet struct {
	name  string
	field func(Product) string
	sel   func(Query) []string
}

var facets = []facet{
	{"platform", func(p Product) string { return p.Platform }, func(q Query) []string { return q.Platforms }},
	{"region", func(p Product) string { return p.Region }, func(q Query) []string { return q.Regions }},
	{"type", func(p Product) string { return p.Type }, func(q Query) []string { return q.Types }},
	{"genre", func(p Product) string { return p.Genre }, func(q Query) []string { return q.Genres }},
}

func (q Query) priceOK(p Product) bool {
	price := p.FromPrice()
	return (q.Min == nil || price >= *q.Min) && (q.Max == nil || price <= *q.Max)
}

// matches applies every filter except the facet named skip.
func (q Query) matches(p Product, skip string) bool {
	if !q.priceOK(p) {
		return false
	}
	for _, f := range facets {
		if f.name == skip {
			continue
		}
		if sel := f.sel(q); len(sel) > 0 && !slices.Contains(sel, f.field(p)) {
			return false
		}
	}
	return true
}

// List runs a query over the catalog. Facet counts are "disjunctive": the
// count for a platform ignores the platform filter itself, so the sidebar
// shows what you would get by ticking another box.
func List(all []Product, q Query) Result {
	base := all
	if q.Q != "" {
		base = Search(all, q.Q) // relevance order
	}
	if q.Sort == "" || !slices.Contains(Sorts, q.Sort) || (q.Sort == "relevance" && q.Q == "") {
		if q.Q != "" {
			q.Sort = "relevance"
		} else {
			q.Sort = "popular"
		}
	}
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 24
	}
	if q.Page <= 0 {
		q.Page = 1
	}

	var items []Product
	for _, p := range base {
		if q.matches(p, "") {
			items = append(items, p)
		}
	}
	sortProducts(items, q.Sort)

	res := Result{Total: len(items), Page: q.Page, Limit: q.Limit, Sort: q.Sort, Facets: map[string][]FacetCount{}}
	start := min((q.Page-1)*q.Limit, len(items))
	end := min(start+q.Limit, len(items))
	res.Items = append([]Product{}, items[start:end]...)

	for _, f := range facets {
		counts := map[string]int{}
		for _, p := range all {
			counts[f.field(p)] += 0 // every value is listed, even with 0 hits
		}
		for _, p := range base {
			if q.matches(p, f.name) {
				counts[f.field(p)]++
			}
		}
		list := make([]FacetCount, 0, len(counts))
		for v, c := range counts {
			list = append(list, FacetCount{v, c})
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Value < list[j].Value })
		res.Facets[f.name] = list
	}
	return res
}

func sortProducts(ps []Product, by string) {
	less := map[string]func(a, b Product) bool{
		"popular":    func(a, b Product) bool { return a.Sold > b.Sold },
		"price-asc":  func(a, b Product) bool { return a.FromPrice() < b.FromPrice() },
		"price-desc": func(a, b Product) bool { return a.FromPrice() > b.FromPrice() },
		"discount":   func(a, b Product) bool { return a.Discount() > b.Discount() },
		"newest":     func(a, b Product) bool { return a.Year > b.Year },
	}[by]
	if less != nil { // "relevance" keeps search order
		sort.SliceStable(ps, func(i, j int) bool { return less(ps[i], ps[j]) })
	}
}

// Top returns the first n products by the given sort, skipping sold-out ones.
func Top(all []Product, by string, n int) []Product {
	var ps []Product
	for _, p := range all {
		if _, ok := p.Best(); ok {
			ps = append(ps, p)
		}
	}
	sortProducts(ps, by)
	return ps[:min(n, len(ps))]
}

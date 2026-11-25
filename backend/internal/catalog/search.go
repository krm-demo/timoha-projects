package catalog

import (
	"sort"
	"strings"
	"unicode"
)

// normalize lowercases, strips accents/punctuation and collapses spaces:
// "Baldur's Gate 3" -> "baldur s gate 3".
func normalize(s string) string {
	var b strings.Builder
	space := true
	for _, r := range strings.ToLower(s) {
		switch {
		case r > unicode.MaxASCII:
			// keep it simple: drop non-ASCII (accents, symbols)
			continue
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			space = false
		default:
			if !space {
				b.WriteByte(' ')
				space = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// levenshtein returns the edit distance, capped early for very different lengths.
func levenshtein(a, b string) int {
	// The distance is at least the length difference, so when that alone
	// exceeds any typo tolerance (max 2) we skip the O(len(a)*len(b)) work.
	if d := len(a) - len(b); d > 3 || d < -3 {
		return 9
	}
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

// tolerance is how many typos a query word of this length may contain.
func tolerance(n int) int {
	switch {
	case n <= 2:
		return 0
	case n <= 5:
		return 1
	default:
		return 2
	}
}

// Score rates how well a product name matches a query; 0 means no match.
// Exact > prefix > substring > every query word matches some name word
// (as a prefix or within the typo tolerance).
func Score(name, query string) float64 {
	n, q := normalize(name), normalize(query)
	switch {
	case q == "":
		return 0
	case n == q:
		return 100
	case strings.HasPrefix(n, q):
		return 90
	case strings.Contains(n, q):
		return 80
	}
	words := strings.Fields(n)
	total := 0.0
	qWords := strings.Fields(q)
	for _, w := range qWords {
		best := 0.0
		for _, nw := range words {
			if strings.HasPrefix(nw, w) {
				best = 1
				break
			}
			// Compare with the whole word and with a prefix of the same
			// length, so a typo in an unfinished word still matches:
			// "nightr" vs "nightreign" → prefix "nightr" (0 typos).
			d := levenshtein(w, nw)
			if len(nw) > len(w) {
				d = min(d, levenshtein(w, nw[:len(w)]))
			}
			if d <= tolerance(len(w)) {
				best = max(best, 1-float64(d)*0.25)
			}
		}
		if best == 0 {
			return 0
		}
		total += best
	}
	return 40 + 30*total/float64(len(qWords))
}

// Search returns products matching the query, best match first.
func Search(products []Product, query string) []Product {
	type scored struct {
		p Product
		s float64
	}
	var hits []scored
	for _, p := range products {
		if s := Score(p.Name, query); s > 0 {
			hits = append(hits, scored{p, s})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].s > hits[j].s })
	out := make([]Product, len(hits))
	for i, h := range hits {
		out[i] = h.p
	}
	return out
}

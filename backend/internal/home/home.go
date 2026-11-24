// Package home assembles the landing page. Each block is its own service
// with its own data source; the frontend asks for the layout, then fetches
// every block separately and only renders what it gets back.
package home

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"

	"github.com/krm-demo/timoha-projects/backend/internal/catalog"
)

//go:embed banner.json
var bannerJSON []byte // marketing content, edited by the "CMS" (this file)

// Catalog is the read access a block needs.
type Catalog interface {
	Products(ctx context.Context) ([]catalog.Product, error)
}

// Block produces the payload for one homepage section.
type Block interface {
	ID() string
	Load(ctx context.Context) (any, error)
}

// Registry holds the blocks in display order.
type Registry struct{ blocks []Block }

func NewRegistry(c Catalog) *Registry {
	return &Registry{blocks: []Block{
		Banner{c},
		Categories{c},
		ProductRail{c, "deals", "deals-service", "Hot deals", "discount", "#/store?sort=discount"},
		ProductRail{c, "bestsellers", "bestsellers-service", "Bestsellers", "popular", "#/store?sort=popular"},
	}}
}

// LayoutBlock tells the frontend which blocks exist and where to fetch them.
type LayoutBlock struct {
	ID       string `json:"id"`
	Endpoint string `json:"endpoint"`
}

func (r *Registry) Layout() []LayoutBlock {
	out := make([]LayoutBlock, len(r.blocks))
	for i, b := range r.blocks {
		out[i] = LayoutBlock{b.ID(), "/api/home/" + b.ID()}
	}
	return out
}

func (r *Registry) Get(id string) (Block, bool) {
	for _, b := range r.blocks {
		if b.ID() == id {
			return b, true
		}
	}
	return nil, false
}

// --- banner-service: CMS slides, one slide points at the live #1 bestseller

type Banner struct{ c Catalog }

type slide struct {
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle"`
	CTA      string   `json:"cta"`
	Link     string   `json:"link"`
	Colors   []string `json:"colors"`
}

func (Banner) ID() string { return "banner" }

func (b Banner) Load(ctx context.Context) (any, error) {
	var cms struct{ Slides []slide }
	if err := json.Unmarshal(bannerJSON, &cms); err != nil {
		return nil, err
	}
	ps, err := b.c.Products(ctx)
	if err != nil {
		return nil, err
	}
	slides := cms.Slides
	if top := catalog.Top(ps, "popular", 1); len(top) == 1 && len(slides) > 1 {
		p := top[0]
		slides[1] = slide{p.Name, fmt.Sprintf("Bestseller of the week — from €%.2f", float64(p.FromPrice())/100),
			"Buy now", "#/product/" + p.Slug, p.Colors}
	}
	return map[string]any{"service": "banner-service", "slides": slides}, nil
}

// --- categories-service: platforms that currently have products in stock

type Categories struct{ c Catalog }

func (Categories) ID() string { return "categories" }

func (c Categories) Load(ctx context.Context) (any, error) {
	ps, err := c.c.Products(ctx)
	if err != nil {
		return nil, err
	}
	count := map[string]int{}
	for _, p := range ps {
		if _, ok := p.Best(); ok {
			count[p.Platform]++
		}
	}
	type item struct {
		Label string `json:"label"`
		Count int    `json:"count"`
		Link  string `json:"link"`
	}
	items := []item{}
	for pl, n := range count {
		items = append(items, item{pl, n, "#/store?platform=" + url.QueryEscape(pl)})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Label < items[j].Label
	})
	return map[string]any{"service": "categories-service", "title": "Shop by platform", "items": items}, nil
}

// --- deals-service / bestsellers-service: ranked product rails

type ProductRail struct {
	c                            Catalog
	id, service, title, by, link string
}

func (r ProductRail) ID() string { return r.id }

func (r ProductRail) Load(ctx context.Context) (any, error) {
	ps, err := r.c.Products(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"service": r.service, "title": r.title, "link": r.link, "products": catalog.Top(ps, r.by, 10)}, nil
}

# Proposal: "Smart search" with pills

> Status: **proposal, not implemented yet.** This document describes what to build, why, how it would work and what it costs, so the idea can be judged and pitched before any code is written.
> Russian version: [SMART_SEARCH-ru.md](SMART_SEARCH-ru.md)

## 1. Why this, and not something else

Every candidate will deliver the same checklist: a catalog, filters, fuzzy search, a product page, checkout and login. Doing those well is necessary but not enough to win. A differentiator should meet four conditions:

1. **It solves a real problem of this business**, not a generic "nice to have".
2. **It can be seen in 10 seconds** of a demo, without explanation.
3. **It strengthens required features** (search and store) rather than adding an unrelated extra.
4. **It can be explained line by line** in the interview: no black box.

Ideas that were considered and rejected:

| Idea | Why not (as the main differentiator) |
|---|---|
| Multi-language (EN/DE) | Useful, but nobody asked for it, and every serious shop already has it. It reads as "translation", not innovation |
| AI chat assistant / LLM search | Impressive at first, but costly, slow, non-deterministic and hard to test. It is also hard to defend in an interview: "why did it answer that?" |
| Price history charts, wishlist | Good features, but they need data over time that a prototype doesn't have |
| Recommendations ("you may also like") | Already present in a simple form. A real recommender needs user behaviour data |

## 2. The problem

A digital-key marketplace sells **the same game many times**:

- on different **platforms**: Steam, PlayStation, Xbox, Nintendo, EA app, …;
- for different **regions**: Global, EU, US, …, and a key for the wrong region often cannot be activated;
- in different **editions**: Standard, Deluxe, Gold, …;
- from different **sellers**, at different prices.

So a search for one game returns a long list of near-identical cards, and the buyer has to work out alone:

- *which of these are for my platform?*
- *which ones will actually activate in my country?*
- *which is the cheapest one that works for me?*

A wrong choice leads to a key that doesn't activate, then a support ticket, a refund request and an unhappy customer. That costs the marketplace money.

> **Before pitching, verify on eneba.com.** Search for a popular title (for example "Elden Ring") and count how many cards come back and how many of them are the same game. Take a screenshot. One real screenshot of the problem is stronger than any argument. If Eneba already solves part of it, adjust the pitch to "here is how I would push it further".

## 3. The idea in one picture

The user types: `elden ps5 under 30`

```
┌──────────────────────────────────────────────────────────────────────┐
│ 🔍 elden                                                              │
│    [PlayStation ✕]  [≤ €30 ✕]            your country: [Spain ▾]     │
├──────────────────────────────────────────────────────────────────────┤
│  Refine:  (Standard 2)  (Deluxe 1)  (EU 2)  (Global 1)               │
├──────────────────────────────────────────────────────────────────────┤
│ ▣ ELDEN RING                                          from €24.50    │
│   ( PSN · EU · Standard · €24.50  ✓ works in Spain )   ← cheapest    │
│   ( PSN · Global · Standard · €27.90  ✓ works in Spain )             │
│   ( PSN · US · Standard · €21.00  ⚠ US account only )                │
│                                                                      │
│ ▣ ELDEN RING NIGHTREIGN                               from €29.99    │
│   ( PSN · EU · Standard · €29.99  ✓ works in Spain )                 │
└──────────────────────────────────────────────────────────────────────┘
```

Three parts, one visual language: the **pill**.

1. **The query becomes pills.** The words the system understood (`ps5`, `under 30`) turn into removable pills. Only the real name (`elden`) stays as text for the fuzzy search. The user *sees* how the system read the query and can undo any part with one click.
2. **Results are grouped by game.** One card per game, with its variants as pills. The cheapest variant that works for the user is highlighted.
3. **The "works for you" mark.** Every variant says whether it will activate in the user's country.

Note on the example: the US key is the cheapest (€21.00), but it is **not** highlighted, because it would not work in Spain. This one detail carries the whole idea: *we help the buyer pick the cheapest key that actually works for them*, not just the cheapest one.

## 4. How it works

### 4.1. Query → pills (intent parsing)

The query is normalized (lowercase, accents and punctuation removed) and read left to right. Known phrases are matched longest first, so that `gift card` is matched before `card`.

| Kind | Examples the user types | Becomes the pill | Filter |
|---|---|---|---|
| Platform | `ps5`, `ps4`, `playstation`, `psn` | PlayStation | `platform=PSN` |
| | `xbox`, `xsx`, `series x` | Xbox | `platform=Xbox` |
| | `switch`, `nintendo` | Nintendo | `platform=Nintendo` |
| | `steam` | Steam | `platform=Steam` |
| | `pc` | PC | `platform=Steam,GOG,EA app,Ubisoft,Rockstar` (a group of platforms) |
| Price | `under 30`, `below 30`, `<30`, `до 30`, `unter 30`, `bis 30` | ≤ €30 | `max=30` |
| | `10-30`, `from 10 to 30` | €10–30 | `min=10&max=30` |
| Type | `gift card`, `giftcard`, `voucher` | Gift card | `type=Gift card` |
| | `dlc`, `expansion` | DLC | `type=DLC` |
| | `subscription`, `game pass`*, `ps plus`* | Subscription | `type=Subscription` |
| Region | `eu`, `europe`, `global`, `us` | EU / Global / US | `region=…` |
| Genre | `rpg`, `shooter`, `racing`, … | RPG | `genre=RPG` |

Everything that is not recognised stays as text and goes to the existing fuzzy search (`catalog/search.go`).

**Important rules.** These are where most of the thinking is, and they make good interview material.

- **Names win over aliases.** Before turning a word into a pill, check whether it is part of a product name the user is typing. `xbox game pass` must find the product *Xbox Game Pass Ultimate*; it must not become "platform Xbox + text `game pass`". Rule: if a phrase of the query matches the start of a product name, the words inside that phrase are not turned into pills. (*) marks such phrases in the table.
- **Pills are the same filters as the sidebar.** A pill is just a visible URL parameter: `#/store?q=elden&platform=PSN&max=30`. Links can be shared, the back button works, and there is no second filter system.
- **Everything is reversible.** ✕ on a pill removes that filter. Typing the word in quotes (`"switch"`) searches it as text.
- **Several languages in the parser, cheaply.** `до 30` (Russian), `unter 30` / `bis 30` (German) and `under 30` all work. This shows i18n awareness without a full translation project.
- **Deterministic.** The same input always gives the same pills. Every rule is a row in a table that can be unit-tested, and the dictionary is easy for the business to extend ("add `ps6`").

### 4.2. Zero results → relax, don't dead-end

If the pills together give nothing (for example `elden switch under 5`), the system does not show an empty page. It removes the most restrictive pill and says what it did:

> No results for **Nintendo · ≤ €5**. Showing **Nintendo** without the price limit (2 games). [Undo]

The order for dropping pills: price first, then region, then type, then platform. The platform goes last because it usually matters most.

### 4.3. Grouping by game

Each listing gets two new fields:

- `title`: the canonical game, for example "Elden Ring";
- `edition`: "Standard", "Deluxe", "Gold", …

The search returns **groups** instead of a flat list:

- one group per `title`;
- the group's variants are the listings (platform · region · edition · best offer price);
- variants are sorted: *works for you* first, then by price;
- the card shows up to 4 variant pills, plus "+N more" that expands the rest;
- facet counts ("Deluxe 1", "EU 2") count **games**, not listings, so the numbers match what the user sees.

Clicking a variant pill opens the existing product page of that exact listing. Nothing else in the shop has to change.

### 4.4. "Works for you"

The user's country comes from the browser (`navigator.language`, e.g. `es-ES` → Spain). There is a country picker to change it, and the choice is remembered. No IP geolocation: it needs no extra service and raises no privacy concerns.

A simple, transparent rule table:

| Key region | Works in |
|---|---|
| Global | every country |
| EU | EU / EEA countries |
| US | United States only |
| other (e.g. EMEA, LATAM) | the listed countries |

The result is one of three marks:

- ✓ *works in Spain*;
- ⚠ *US account only*;
- ? *check the seller's description* (when the region is unknown).

For the prototype the table is hardcoded. In production it would come from publisher or seller data, because real region locks also depend on the platform.

## 5. API contract (proposal)

`GET /api/search?q=elden%20ps5%20under%2030&country=ES`

```json
{
  "query": "elden ps5 under 30",
  "text": "elden",
  "pills": [
    { "kind": "platform", "value": "PSN", "label": "PlayStation", "from": "ps5" },
    { "kind": "maxPrice", "value": 30,    "label": "≤ €30",       "from": "under 30" }
  ],
  "relaxed": null,
  "facets": { "edition": [{ "value": "Standard", "count": 2 }], "region": [{ "value": "EU", "count": 2 }] },
  "groups": [
    {
      "title": "Elden Ring",
      "slug": "elden-ring",
      "colors": ["#3a2a12", "#c9a24a"],
      "fromPrice": 24.50,
      "variants": [
        { "slug": "elden-ring-psn-eu", "platform": "PSN", "region": "EU", "edition": "Standard",
          "price": 24.50, "works": "yes", "best": true },
        { "slug": "elden-ring-psn-us", "platform": "PSN", "region": "US", "edition": "Standard",
          "price": 21.00, "works": "no" }
      ]
    }
  ]
}
```

`relaxed` explains a fallback, for example `{"dropped": {"kind": "maxPrice", "label": "≤ €5"}}`.

`GET /api/products` accepts `country` and `group=title` in the same way. The store page and the search dropdown therefore share one model.

## 6. What changes in our code

| Part | Where | Approx. lines | Change type |
|---|---|---|---|
| Intent parser (dictionary, longest-match, name-wins rule) | `backend/internal/catalog/intent.go` | ~150 | new |
| Parser tests (one table row per rule and edge case) | `intent_test.go` | ~100 | new |
| Grouping by title + facet counts per group | `catalog/group.go` + test | ~70 + ~60 | new |
| `Title`, `Edition` fields | `catalog/catalog.go`, `store/schema.sql`, `store/pgstore.go` | ~20 | edit |
| Region rule | `catalog/region.go` + test | ~40 + ~30 | new |
| Zero-result relaxation | `catalog/intent.go` | ~30 | new |
| API: `pills`, `groups`, `relaxed`, `country` | `api/api.go` | ~50 | edit |
| Demo data: each game on 2–4 platforms, regions and editions | `scripts/gen_data.py` | ~40 | edit |
| UI: pills in the search box, grouped cards, country picker | `web/app.js`, `web/styles.css` | ~200 | edit |
| Static mirror for GitHub Pages | `web/data.js` | ~80 | edit |
| **Total** | | **≈ 870 lines** | mostly new files |

Existing code changes only at the edges: the API response, the data model fields and the UI. Checkout, auth, orders and the homepage stay untouched.

**Effort:** about an hour of AI-agent work including tests, or 2–3 days for a person writing it by hand. Budget extra time for Tymofii to read it and be able to explain the parser and the grouping.

### Milestones (each one is demo-able on its own)

1. **Pills from the query.** The most visible part, and it can be shown in 10 seconds.
2. **Grouping by game.** It needs the new demo data.
3. **Works for you,** together with the zero-result relaxation.

If time runs out after milestone 1, the feature is still a clear differentiator.

## 7. How to demo it (about 60 seconds)

1. Open eneba.com (or your screenshot) and search "elden ring": *"A lot of near-identical cards. Which one should I buy?"*
2. In our app, type `elden ps5 under 30`. The pills appear: *"The system shows how it understood me."*
3. Point at the grouped card: *"One game, its variants as pills, the cheapest one that works for me highlighted. The US key is cheaper, but it would not activate in Spain, so it is not recommended."*
4. Switch the country to United States: the highlight moves to the US key.
5. Type `elden switch under 5`: *"No dead end: it tells me what it relaxed."*
6. Click ✕ on a pill, then copy the URL into a new tab: *"Pills are plain filters, so links are shareable."*

## 8. How we would know it works (in production)

- **Fewer wrong-region purchases and refunds.** This is the money metric.
- **Search → product click-through rate** goes up, and the **zero-result rate** goes down.
- **Time from search to add-to-cart** goes down.
- **Pill removal rate.** If users often remove a pill, that alias is wrong. This is a free quality signal for the dictionary.

## 9. Alternatives considered

| Alternative | Pros | Why not now |
|---|---|---|
| LLM to understand the query | Understands anything | Cost and latency per keystroke, non-deterministic, hard to test and explain. It could later be a *fallback* for queries the dictionary doesn't understand |
| Elasticsearch / OpenSearch / Algolia / Meilisearch | Built-in facets, typo tolerance, synonyms, grouping ("distinct") | An extra service to run and pay for. Our design maps 1:1 onto them: the alias dictionary becomes *synonyms*, and grouping becomes the *distinct attribute*. So it is a migration path, not a rewrite |
| Postgres full-text (`tsvector`) + `pg_trgm` | No extra service | The right next step when the catalog grows. It replaces only the text-matching part; pills and grouping stay |

## 10. Risks and how they are handled

| Risk | Mitigation |
|---|---|
| The parser misreads a word (e.g. a game called "Switch…") | Names win over aliases. Pills are visible and removable. Quotes force text search |
| The alias dictionary goes stale | It is data, not code: one table, unit-tested. The pill removal rate shows bad aliases |
| Region rules are more complex in reality | The prototype says "?" when unsure. Production data comes from sellers and publishers |
| Grouping hides a variant the user wanted | "+N more" expands all variants. Facet pills filter by edition and region |

## 11. Interview talking points

- *"I looked for the real cost in this business. Wrong-region keys mean refunds and support tickets, so I made search help the buyer choose a key that works, not just one that matches the name."*
- *"I chose a deterministic parser over an LLM: it's free, fast, testable, and I can explain every rule. An LLM could be a fallback later."*
- *"Pills are just URL filters, so there is one filter system, shareable links and a working back button."*
- *"The design maps onto Algolia or Elasticsearch concepts (synonyms, distinct), so it scales without a rewrite."*

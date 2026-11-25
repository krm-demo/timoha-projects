# Prompt playbook for the next steps

Eneba asks for the **real** AI chat history. The cleanest way to have a log you are proud of is to run the remaining work yourself, in your own sessions, with well-structured prompts. This file gives you those prompts, ready to copy.

Rules that make a log look professional (and make the agent work better):

1. **One session per task:** React port, smart search, deployment. Short, focused logs are easier to read than one endless chat.
2. **Plan first, then build.** Start every session by asking for a plan and approving it. Reviewers love seeing that you steered the work.
3. **Give context, goal, constraints and acceptance criteria.** That is the structure of every prompt below.
4. **Push back and ask "why".** Questions like *"why FOR UPDATE here?"* in your log show understanding, which is exactly what the interview checks.
5. **Finish with a review step.** Ask the agent to list what it is unsure about, then check it yourself.
6. **Export the whole conversation** (a share link, or copy it into a `.md` file) and add it to the submission next to this repo's `docs/AI_SESSION_LOG.md`.

---

## Session 1: React + TypeScript frontend (required)

### Prompt 1 (plan)

```
Context: I'm Tymofii, applying for the Eneba Software Engineer Intern role.
Repo: <link to my repo>. It has a Go REST API in backend/ (see README "API")
and a vanilla-JS prototype in web/. The assignment requires a React frontend,
TypeScript preferred.

Goal: port web/ to React + TypeScript in a new frontend/ folder, using the
existing API unchanged.

Constraints:
- Vite, react-router-dom, @tanstack/react-query; no UI kit, reuse web/styles.css
- Same pages and behaviour as web/: home blocks from /api/home, store with
  filters in the URL, product page with offers, cart, checkout, orders, auth
- Types in src/api/types.ts must mirror backend/internal/catalog and store

Before writing code: give me a short plan (folders, components, routes,
data flow, how auth token and cart are stored) and wait for my OK.
```

### Prompt 2 (build, after you approve the plan)

```
Plan approved, with these changes: <your changes>.
Build it step by step. After each step run the build and tests and tell me
what you verified. Acceptance criteria:
- npm run build passes, Vitest tests for SearchBox, Store filters and checkout errors
- Signed-out user is redirected from /checkout to /login?next=/checkout
- Works against the Go server locally (go run ./cmd/server) with VITE_API_BASE
```

### Prompt 3 (review)

```
Review your own work: list anything you're unsure about, any shortcuts you
took, and 3 questions an interviewer could ask me about this code with the
answers pointing to files and lines.
```

---

## Session 2: smart search (the differentiator)

### Prompt 1 (plan)

```
Context: same repo. docs/SMART_SEARCH.md describes a proposed feature:
query → pills, results grouped by game, and a "works in your country" mark.

Goal: implement milestone 1 only (query → pills) in the Go backend and the
React frontend.

Constraints:
- Deterministic parser in backend/internal/catalog/intent.go, no LLM
- Table-driven tests in intent_test.go: one row per rule from section 4.1,
  including "names win over aliases" (xbox game pass) and quoted text
- Pills map 1:1 to existing URL filters (platform, min, max, type, region, genre)

First, explain how you will implement the longest-match parsing and the
"names win" rule, with 3 example queries and the expected pills. Wait for my OK.
```

### Prompt 2

```
Go ahead. Then show me the test table and a short demo script I can follow.
After that, we do milestone 2 (grouping) the same way: plan first.
```

---

## Session 3: deployment

```
Context: same repo, docs/DEPLOY_AZURE.md and render.yaml.
Goal: deploy to <Render | Azure> so reviewers get a live URL.
Walk me through it step by step; I run the commands myself and paste the
output back to you. Stop after each step and tell me what to check.
Finally, give me a checklist to verify the live site (Live API badge,
search, checkout with test card, orders).
```

---

## Prompt template for anything else

```
Context:   what exists already (files, links), who I am, why I'm doing this
Goal:      one sentence, what "done" means
Constraints: libraries, style, what must not change
Acceptance criteria: how we verify it (tests, commands, behaviour)
Process:   plan first and wait for my OK / then build / then self-review
```

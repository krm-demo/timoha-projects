# AI session log (summary)

> **This is a summary, not the full log.** Eneba requires the *full* conversation (all prompts and responses). Export it from the Claude conversation where this repo was created (share link, or copy it into a `.md` file), and add your own later sessions too.

Tool: Claude (Anthropic) agent with shell, GitHub and Gmail access. Date: 24 Nov 2026. Prompts were written by Aleksey together with Tymofii.

| # | Prompt (paraphrased) | What the agent did |
|---|---|---|
| 1 | Find Tymofii's latest email and render the site it should become; publish on GitHub Pages in a new repo | Searched Gmail. The newest email from his first address was flight tickets, so it asked. The answer pointed to today's email with a PDF: the Eneba Intern Assignment. It read the PDF and built a static clickable prototype in the Eneba style (store, filters, fuzzy search, product offers, cart, test checkout with keys, local accounts), then tested it in a headless browser. It could not create the repo itself, because of session permissions |
| 2 | "Repository `timoha-projects` created, push everything" | Pushed the code and added a Pages workflow. The first deploy failed because Pages wasn't enabled, so the user switched it on in Settings and the agent re-ran the deploy |
| 3 | Add a Go backend according to the requirements | Wrote `backend/`, with standard library only plus optional pgx: catalog filtering and facets, typo-tolerant search, homepage block services, PBKDF2 + JWT auth, test-mode payments, transactional checkout, and two databases (JSON file and Postgres) behind one interface with shared tests. It moved the frontend into `web/` with an API-or-static data layer, and added CI (Go tests, Postgres integration, Docker smoke test), a Dockerfile and render.yaml. CI failed twice (missing transitive go.sum entries for pgx, then a wrong grep in the smoke test); both were fixed |
| 4 | How hard is EN/DE? Explain frontend ↔ backend; compare Go with Java/Node/Python | Added README sections: a sequence diagram, measured Go numbers (≈6 MB binary, <0.3 s start, ≈12 MB RAM), a comparison table including PHP (the assignment allows only PHP or Go), and i18n estimates |
| 5 | Keep it as is; prepare for Tymofii to take over | Wrote `docs/HANDOFF.md`, `docs/ROADMAP.md`, `docs/INTERVIEW_NOTES.md` and this file |

| 6 | Azure deployment guide; propose an innovation; write it up as docs (EN + RU) | `DEPLOY_AZURE.md`, image publishing to GHCR, `SMART_SEARCH.md`/`-ru.md`, `ROADMAP-ru.md` |
| 7 | Add comments to tricky places in the Go code | Commented the non-obvious parts: disjunctive facets, constant-time JWT check, atomic file writes, row locks, rounding. The review found and fixed two real bugs, each with a test: price filters truncated cents (`max=19.99` acted as 19.98), and multi-item checkouts could deadlock in Postgres (lines are now locked in a fixed order) |

For the remaining work, see `PROMPT_PLAYBOOK.md`: structured prompts to run in your own sessions, so that part of the log is genuinely yours.

## Decisions worth mentioning in the submission

- **Static fallback.** The site works on GitHub Pages with no server, so reviewers always have something to click. With the API available it switches to live data automatically.
- **Prices are never trusted from the client.** Checkout reads prices and stock inside one DB transaction.
- **One Store interface, two databases, one contract test.** This keeps the demo setup-free without giving up a real database.
- **Known gap:** the frontend is not React yet (see ROADMAP).

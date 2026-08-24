# Frontend brainstorm

## The question

mem is intentionally local and terminal-native. A frontend needs to add reach
without betraying that. Two realistic directions: **TUI** and **web app**.

---

## TUI

**Pros**
- Stays in the terminal — zero new dependencies if built into the binary
- Fast, keyboard-driven, feels native to the tool
- Libraries exist in Go: bubbletea, tview

**Cons**
- Not accessible on smartphone
- Cannot share a link to a note ("see note X")
- Rendering rich Markdown is painful in a terminal

**Verdict:** great for power users on desktop. Not a replacement for mobile access.
Could be `mem ui` as an optional local mode.

---

## Web app (served by `mem serve`)

`mem serve` already speaks HTTP. Adding a UI route (`GET /`) makes the server serve
a single-page app — no new process, no separate binary. The browser becomes the
frontend; the phone gets it for free over LAN or via the existing nginx/TLS setup.

**Verdict:** recommended primary frontend. Composable with TUI later.

---

## Architecture sketch

```
browser / mobile
      │
      ▼
  mem serve
  ├── GET /              → serves bundled SPA (embed into binary)
  ├── GET /api/notes     → existing — list
  ├── GET /api/notes/:id → existing — detail
  ├── POST /api/notes    → existing — create
  ├── PATCH /api/notes/:id → existing — update
  └── GET /api/search?q= → NEW — full-text search
```

The SPA is a single HTML+JS file compiled into the binary with `go:embed`.
No build step required by the user; no CDN; works offline over LAN.

---

## Feature ideas

### Core browsing

- **Timeline / day view** — same as `mem day`, vertical scroll of days with
  notes grouped under date headers; infinite scroll upward into the past
- **Note detail** — rendered Markdown, frontmatter shown as pill tags
- **Tag browser** — sidebar or filter bar listing all tags with note counts;
  clicking narrows the list (like `mem ls -t`)
- **Slug search** — instant filter by slug prefix (what fzf currently does in
  the terminal)

### Search

The most compelling feature gap over the CLI.

- **Full-text search** — server-side: on `GET /api/search?q=hello` the server
  runs `strings.Contains` or regexp over all note bodies; returns ranked hits
  with matched excerpts (like `rg --json`). No index needed for small corpora.
- **Regex mode** — toggle: treat query as regexp. Mirrors `rg -e`.
- **Tag filter + text search combined** — `q=hello&tag=log` narrows to tagged
  notes first then full-text within that set. Cheap intersection.
- **Large corpora** — if notes grow past ~10k files, add a trigram index on
  the server (pure Go, no sqlite FTS needed). Defer until needed.
- **Search-as-you-type** — debounce 200ms, stream results as server sends them
  (chunked JSON or SSE). Feels like ripgrep in the browser.

### Capture (write from browser / phone)

- **Quick snap** — a single text input at the top; Enter creates a note via
  `POST /api/notes`. Mirrors `mem snap` from a phone.
- **Full editor** — `textarea` with live Markdown preview split-pane.
  On mobile the preview collapses; full-screen edit.
- **Tag autocomplete** — suggest existing tags as the user types `#`.

### Navigation

- **Backlink view** — notes that mention the current note's slug in their body.
  Computed at read time, no index needed for small sets.
- **Related notes** — reuse the IDF scorer from `mem related` (ROADMAP item);
  server endpoint `GET /api/notes/:id/related`.
- **Chronological neighbours** — prev/next note by timestamp in the detail view.

### Quality of life

- **Dark/light theme** — respects `prefers-color-scheme`; toggle override stored
  in `localStorage`.
- **Keyboard shortcuts** — `/` to focus search, `j`/`k` to navigate list,
  `Enter` to open, `Esc` to go back — same muscle memory as the TUI.
- **Share link** — `GET /<id>` (or `/#id=<id>`) opens that note directly.
  Useful for sharing within a home network.
- **Offline-ish** — notes already fetched are cached in a service worker so the
  recent timeline is readable without the server up. Stretch goal.

---

## What the SPA does NOT need

- User accounts / login — auth is already the bearer token on the server
- A framework — vanilla JS + a Markdown renderer (marked.js inlined) is enough
- A database — all state lives in the server's note files
- A build pipeline — one HTML file embedded via `go:embed`

---

## Open questions

1. **Embed vs separate binary** — embedding keeps `mem serve` self-contained but
   inflates the binary. A separate `mem-web` command could serve only the UI.
   Probably embed is the right default; binary size is not a real constraint.

2. **Auth in the browser** — bearer token must be passed with every request.
   Options: prompt on first load and store in `sessionStorage`; or serve the SPA
   only if the request already carries the token (forward the token in the URL
   hash, stripped immediately by JS — never sent to server). The URL-hash approach
   avoids prompting on mobile if the user opens a bookmark.

3. **TUI as companion** — `mem ui` could open a bubbletea TUI that proxies the
   same `/api/*` endpoints. The server-side API is then the source of truth for
   both frontends. Worth keeping as a future path.

4. **Sync status indicator** — if the user has sync configured, show last-sync
   time and a push/pull button. Useful on mobile where you can't run the CLI.

---

## Rough priority order

1. Read-only timeline + day view + tag filter  ← most useful, least risk
2. Full-text search endpoint + search UI
3. Quick snap from browser
4. Full editor
5. Backlinks / related
6. Service worker / offline cache

# mem-cli

Shared external memory for human and AI. Atomic notes for facts, findings,
commands, and ideas — captured quickly, looked up fast.

---

## Storage

```
~/.mem/
  notes/          # markdown note files
  attachments/    # copied attachment files
```

Config: `$XDG_CONFIG_HOME/mem/config.json` → `~/.config/mem/config.json`

---

## File format

**Filename:** `20260511T143022-kafka-rebalance.md`

The slug in the filename is the title. It does not appear again inside the
file — no `# Heading`, no `title:` field. Timestamp prefix is the sole source
of creation time.

**Frontmatter** (tool-managed, never shown to the human):

```markdown
---
id: a3f2b1c4
tags: [b1c2d3e4, c2d3e4f5]
from: [f5a6b7c8]
links: [d3e4f5a6]
attachments: [/Users/you/.mem/attachments/20260511T143022-diagram.png]
---

Consumer group rebalance blocks all partitions for ~2min with the default eager
protocol. Fix: switch to cooperative-sticky assignor. Confirmed by @kate.
```

- `id` — 8-char hex UUID, assigned at creation, never changes
- `tags` — omitted if none; list of tag UUIDs (registry maps UUID → slug)
- `from` — omitted if none; list of entity UUIDs (registry maps UUID → slug)
- `links` — omitted if none; list of note UUIDs for `[[wiki-links]]` in prose
- `attachments` — omitted if none; absolute paths
- Nothing else — no `created`, no `updated`, no `title`

The prose body always uses human-readable slugs — `@kate`, `[[kafka-session-timeout]]`,
`#kafka`. Frontmatter stores UUIDs. `FinalizeNote` resolves slugs to UUIDs via
the registry on every save. Renaming a tag, person, or note slug updates the
registry; frontmatter is immediately consistent without touching any note file.

`from` is entities only — slugs that map to `@` inline mentions. URLs belong
inline in the body as markdown links, not in `from`.

**The human only writes the body.** The editor opens a temp file containing
the body only — the note file is never modified until `FinalizeNote` writes
the final result atomically. If the editor crashes, the note file is untouched.

---

## Inline notation

Tags and people can be expressed inline in the note body. `FinalizeNote`
extracts them and merges with any flag-supplied metadata.

- `#tag` — inline tag, e.g. `#kafka`, `#ios`
- `@slug` — inline attribution, e.g. `@kate`, `@thomas-mueller`, `@acme-team`
- `[[slug]]` — explicit cross-reference, e.g. `[[kafka-rebalance]]`

All are optional. A note with no tags, no attribution, and no links is valid.

**Extraction safety — stripped before regex runs:**

1. Fenced code blocks (` ```...``` `)
2. Indented code blocks (4-space prefix lines)
3. Inline code (`` `...` `` spans)
4. URLs (`https?://\S+`) — prevents `#fragment` and `@user` in URLs being
   extracted as tags or from values

Stripping order matters: code blocks first, then inline code, then URLs. What
remains is prose — the only text where inline notation is meaningful.

**UUID resolution** — after extraction, `FinalizeNote` resolves each slug to a
UUID via the registry:
- Found → use the existing UUID
- Not found → mint a new 8-char hex UUID; check it against all existing UUIDs
  in the registry across all namespaces before writing — if it collides, mint
  again (retry until unique). Add to the registry under the appropriate
  namespace (`tags`, `from`, or `notes`).

Frontmatter is then written with UUIDs. If a slug that appeared in the previous
frontmatter no longer appears in prose or flags, its UUID is dropped from this
note's frontmatter. The UUID remains in the registry (other notes may still
reference it).

---

## Tags

**Format:** lowercase, hyphens — `kafka`, `consumer-group`, `ios`

**Source of truth:** the notes themselves. mem scans all notes and builds a
local index cache at `~/.mem/notes/.mem-index.json` after every write. The
index stores all unique tags and `from` values for O(1) shell completion — no
external backend.

**Consistency:** enforced by completion, not by the tool. The inline regex
accepts only `[a-z][a-z0-9-]*` — uppercase and special characters are silently
dropped at finalization. Shell and editor completions steer the human to
existing spellings; the format constraint prevents divergence.

**Shell completion:** `-l/--label` flag reads from the index via `mem tags`.

**Editor completion (nvim):** nvim reads `~/.mem/notes/.mem-index.json`
directly. The index has `tags[]`, `from[]`, and `links[]`. A Lua snippet in
nvim config (~20 lines) provides:

- `#` trigger → completes from `tags[]`
- `@` trigger → completes from `from[]`
- `[[` trigger → completes from note slugs (derived from filenames)

Reads the index lazily — only when triggered, not on every keystroke. Uses
`omnifunc` or a `nvim-cmp` custom source. No new mem commands needed — the
index is the interface (P2, P8). The Lua snippet lives in nvim config, not
in mem-cli.

---

## From

**Format:** slug — lowercase, hyphens — `kate`, `thomas-mueller`, `acme-team`

**Rationale:** `@slug` inline syntax requires a single token. Spaces are
incompatible with `@word` matching. The slug is the natural form for inline
mention. Values can be individuals, groups, teams, or any named origin.

**Multiple values:** a note can have many. Stored as a list in frontmatter
(`from: [kate, thomas-mueller]`) and deduplicated on merge. Meetings with
multiple participants are the canonical case.

**URLs:** not stored in `from`. URLs are references, not attribution — put them
inline in the body as markdown links.

**Shell completion:** `-f/--from <slug>` flag reads from the index. Editor
completion same as tags via index.json.

---

## Commands

### `mem new [<title>] [-l <tag>] [-f <person>] [--body <text>]`

Create a new note.

- Title is optional — sets the filename slug at creation time
- No title → timestamp-only file (`20260519T143022.md`), rename later with `mem rename`
- `-l/--label <tag>` — repeatable, tab-completes from index
- `-f/--from <slug>` — repeatable, tab-completes from index
- `--body <text>` — supply body inline; skips the editor entirely (non-interactive mode)
- Stdin alternative: if stdin is not a TTY and `--body` is not set, body is read from stdin

**Interactive mode** (default — no `--body`, stdin is a TTY):
- Writes body only to a temp file (`/tmp/mem-new-<uuid>.md`), opens `$EDITOR` on it
- Note file is not created until the editor closes successfully
- After editor closes (exit 0): runs `FinalizeNote` on temp content + flags,
  writes note file atomically (`os.WriteFile` to temp path, then `os.Rename`)
- Temp file deleted after write
- If editor closes with empty body AND no flags supplied → no file created, exit 0
- If editor closes with empty body BUT flags were supplied → note saved with
  frontmatter only, no body. Valid per P7 (gradual enrichment).
- If editor exits non-zero → no file created, exit 0

**Non-interactive mode** (`--body` supplied, or stdin piped):
- Runs `FinalizeNote` immediately on the supplied body + flags
- Writes note file atomically, rebuilds index
- Empty body with no flags → no file created, exit 0
- Empty body with flags → note saved with frontmatter only (same as interactive)
- `$EDITOR` is never consulted — safe to run from scripts and AI agents

### `mem edit [<identifier>]`

Open a note in `$EDITOR`.

- No identifier → shared fzf picker (same as `mem get`, action on selection differs)
- Identifier is a slug (`kafka-rebalance`) or bare timestamp (`20260519T143022`)
- Bare timestamp addresses unnamed notes directly
- Body extracted from note into a temp file (`/tmp/mem-edit-<uuid>.md`);
  note file untouched until editor closes
- After editor closes (exit 0): runs `FinalizeNote` on temp content +
  preserved metadata, writes note file atomically (`os.Rename`)
- If editor exits non-zero → note file unchanged, temp file discarded
- Tab-completes slugs and bare timestamps of unnamed notes — timestamps shown
  with first non-empty body line as context: `20260519T143022  thomas said kafka…`

### `mem get [<identifier>]`

View a note.

- No identifier → shared fzf picker (same as `mem edit`, opens in bat on selection)
- Identifier is a slug or bare timestamp
- Displays body only — frontmatter stripped, consistent with what the human writes
- Opens in bat (pager mode); falls back to less, then cat
- Tab-completes slugs and bare timestamps of unnamed notes — timestamps shown
  with first non-empty body line as context: `20260519T143022  thomas said kafka…`

`mem get` and `mem edit` share one picker implementation. The action on
selection is the only difference — view vs edit. Same list, same preview pane,
same key bindings. fzf preview shows body only — tags and from are visible
in the fzf line columns and need not be repeated in the preview.

### `mem ls [--tag <tag>] [--from <person>] [--unnamed]`

Browse notes interactively via fzf.

- `--tag/-t` — pre-filters by tag
- `--from` — pre-filters by person
- `--unnamed` — shows only timestamp-only notes (the inbox queue), sorted oldest first
- Each fzf line: `<slug>\t<created>` — unnamed notes show `(unnamed)\t<created>`
- bat preview pane
- Key bindings:
  - `Enter` — open selected note in `mem get` (view)
  - `Ctrl-E` — open selected note in `mem edit`
  - `Ctrl-R` — rename selected note (prompts for new title)
  - No delete binding — destructive actions stay explicit CLI commands

### `mem search <query>`

Full-text search across `~/.mem/notes/`.

Searches prose and frontmatter. Because frontmatter stores UUIDs, `mem search`
resolves the query against the registry first: if the query matches a known
slug, the corresponding UUID is also searched in frontmatter. This ensures
`mem search thomas-mueller` finds notes where he appears only in a `from:` UUID
and not in prose.

Output: slug + matching line with context, one result per match:
```
kafka-rebalance
  Consumer group rebalance blocks all partitions for ~2min

kafka-session-timeout-default
  @thomas-mueller confirmed session timeout defaults to 3s
```

Slug derived from filename. Multiple matches in one note appear as separate
lines under the same slug.

### `mem tags`

List all tags with note counts. Output: `<tag>\t<count>`

### `mem index [--embeddings]`

Manually rebuild the index cache.

- Plain `mem index` — rebuilds `~/.mem/notes/.mem-index.json` from scratch
- `--embeddings` — additionally regenerates vectors for notes whose mtime is
  newer than their last-embedded timestamp in `~/.mem/notes/.mem-vectors.db`
- Needed after external edits (AI writes, direct file edits outside mem-cli)
- Automatic rebuild happens after every mem write command; this is the manual
  escape hatch

### `mem from`

List all `from` values across notes. Output: `<slug>\t<count>`

Symmetric with `mem tags`. Used by nvim integration and shell completion.

### `mem rename [--tag | --from] <old> <new-title>`

Rename a note, tag, or from-entity. Always updates both the registry and all
prose references — no partial rename is offered.

Shows affected files before writing, then prompts for confirmation:

```
mem rename --tag kafka kafka-streams

Will rewrite #kafka in 12 notes:
  20260512T093011-kafka-session-timeout.md
  20260514T110432-partition-strategy.md
  ...

Proceed? [y/N]
```

If no prose references exist, no prompt — renames immediately.

**Note rename** (no flag):
- `<old>` is a slug or bare timestamp
- New title slugified automatically: `Kafka Session Timeout` → `kafka-session-timeout`
- Updates registry, renames file, rewrites `@old-slug` and `[[old-slug]]` in prose

**Tag rename** (`--tag`):
- Updates registry, rewrites `#old-slug` in prose across all notes

**From rename** (`--from`):
- Updates registry, rewrites `@old-slug` in prose across all notes

`--yes/-y` skips the confirmation prompt. Non-TTY without `--yes` exits `2`.
Rebuilds index after completion.

### `mem attach <identifier> <file>`

Attach a file to an existing note.

- Identifier is a slug or bare timestamp
- Copies file to `~/.mem/attachments/<nanosecond-timestamp>-<filename>`
- Nanosecond timestamp prefix guarantees uniqueness without hashing
- Appends absolute path to note's frontmatter `attachments` list

### `mem serve [--port <n>] [--lan] [--token <secret>]`

Start a local HTTP capture server.

- Default port: `4444`
- Binds to `127.0.0.1` by default — `--lan` to expose on the local network
- `--token <secret>` — when set, all requests must supply `?token=<secret>`.
  Intended for `--lan`: bookmark `http://192.168.1.x:4444/?token=abc123` on
  your phone. No sessions, no OAuth — shared secret is sufficient for a
  trusted local network. Requests without a valid token return 403.
- Single-page form: body textarea and optional title field only — no separate
  tag/from fields. Write `#tag` and `@slug` inline in the body, same as the
  editor. Consistent with the writing model, simpler UI.
- Empty title field → timestamp-only note (mirrors `mem new` with no title)
- Filled title field → slugified filename
- On submit: creates note, runs FinalizeNote, rebuilds index
- Post-submit: inline confirmation showing created slug, form cleared for next
  capture — no page reload
- Foreground process — Ctrl-C to stop. No daemon, no background mode.
- Capture only — no editing, no browsing. Those stay CLI.

### `mem similar <query>`

Semantic search using local embeddings.

- Free-text query embedded on the fly, matched against stored note vectors
- Output: ranked list of slugs with similarity score
- Requires an embeddings endpoint (default: ollama at `http://localhost:11434`)
- Endpoint configurable in `~/.config/mem/config.json` via `embeddings_url`
- Errors with a setup message if the endpoint is unreachable — no silent fallback
- Vector store: `~/.mem/notes/.mem-vectors.db` (SQLite, one row per note)
- Embeddings are generated lazily — on first `mem similar` call or via
  `mem index --embeddings`
- Staleness: note mtime compared against last-embedded timestamp; stale notes
  re-embedded automatically before search

---

## Index cache

`~/.mem/notes/.mem-index.json`:

```json
{
  "dir_mtime": 1234567890,
  "total_notes": 847,
  "tags": ["kafka", "kubernetes", "music"],
  "tag_counts": {"kafka": 34, "kubernetes": 12, "music": 3},
  "from": ["kate", "thomas-mueller"],
  "from_counts": {"kate": 41, "thomas-mueller": 18},
  "links": ["kafka-rebalance", "partition-strategy"]
}
```

Stores slugs (resolved from registry) for shell and editor completion.
`total_notes`, `tag_counts`, and `from_counts` are used by `mem related` to
compute IDF weights. Rebuilt after every write (`mem new`, `mem edit`,
`mem attach`). On read, dir mtime is checked — if stale, index is rebuilt
before returning.

---

## Registry

`~/.mem/notes/.mem-registry.json`:

```json
{
  "notes": {
    "a3f2b1c4": "kafka-rebalance",
    "d3e4f5a6": "kafka-session-timeout"
  },
  "tags": {
    "b1c2d3e4": "kafka",
    "c2d3e4f5": "consumer-group"
  },
  "from": {
    "f5a6b7c8": "kate",
    "e4f5a6b7": "thomas-mueller"
  }
}
```

Maps UUID → current slug for notes, tags, and from-entities. Maintained live by
`FinalizeNote` (on every save) and `mem rename` (on every rename). The index
stores the resolved slugs; the registry stores the stable identity underneath.

`mem index` verifies the registry and prunes orphaned UUIDs — UUIDs no longer
referenced by any note's frontmatter. Note UUIDs are fully reconstructible from
filenames. Tag and from UUIDs are reconstructible by scanning `#tag` and `@slug`
prose across all notes; flag-supplied metadata without a prose counterpart is
re-minted with new UUIDs if the registry is lost (frontmatter links become stale
until notes are re-saved).

---

## Exit codes

Consistent across all commands for scripting (P8):

| Code | Meaning |
|------|---------|
| `0`  | Success — including deliberate no-op (empty editor close, fzf cancelled) |
| `1`  | Error — note not found, invalid slug, file write failed |
| `2`  | Usage error — wrong flags, missing required argument |
| `130`| Interrupted — Ctrl-C, user cancelled fzf picker |

Empty editor close is always `0` — a deliberate choice, not a failure.

## Empty notes directory

When the notes directory is empty or does not exist, commands that list or
search notes print a single message to stderr and exit `0`:

```
No notes yet — run: mem new
```

fzf is not invoked with empty input — a blank screen with no explanation is
worse than a clear message.

## Config

`~/.config/mem/config.json` — all fields optional, defaults shown:

```json
{
  "notes_dir":          "~/.mem/notes",
  "attachments_dir":    "~/.mem/attachments",
  "attachment_backend": "local",
  "embeddings_url":     "http://localhost:11434"
}
```

- `notes_dir` — where note files live
- `attachments_dir` — where attached files are copied (local backend)
- `attachment_backend` — `"local"` or `"paperless-ngx-cli"`
- `embeddings_url` — OpenAI-compatible embeddings endpoint for `mem similar`

If the config file does not exist, all defaults apply. No error, no prompt.
Unknown fields are ignored — forward compatibility.

**`local` attachment backend (default)** — copies file to
`<attachments_dir>/<timestamp>-<filename>`, stores absolute path in frontmatter.

**`paperless-ngx-cli` attachment backend (future)** — uploads via
paperless-ngx-cli, stores document ID or URL in frontmatter.

mem-cli never reads attachment content — it only stores and displays the
reference.

---

## Design decisions

Decisions made, alternatives considered, and why.

---

### Frontmatter vs no frontmatter

**Decision:** YAML frontmatter, tool-managed.

Considered: sidecar files, trailing metadata block, inline-only with no
structured metadata. All trade one problem for another — sidecar files break
portability, trailing blocks aren't standard, inline-only loses per-file
self-containment (P3). Frontmatter is the established convention for structured
metadata on plain text files. Every serious plain-text tool speaks it (Obsidian,
Hugo, Pandoc). Grep-friendly, durable, parseable in isolation without the index.

The sandwich pattern (strip before edit, restore after) is the deliberate
consequence: the tool manages structured data so the human never has to.

---

### Inline notation: merge-up vs derive-down

**Decision:** merge-up. Frontmatter is the authority; inline `#tags` and `@from`
in the body are a writing shortcut that flows up into frontmatter at finalization.

Considered: derive-down — frontmatter as a pure reflection of body, body as the
sole truth. Cleaner conceptually (one source, no duplication) but forces
injecting flag-supplied metadata (`-l kafka`, `-f thomas-mueller`) into prose
as `#kafka` or `@thomas-mueller`. That contaminates natural writing with
machine-placed tokens.

Merge-up accepts visible redundancy (a tag written inline also appears in
frontmatter) in exchange for clean prose. Flags feed frontmatter directly
without touching the body. Frontmatter is the union of all sources: inline
extraction, CLI flags, and metadata preserved across prior edits.

---

### `from` field: list vs singular

**Decision:** list — `from: [kate, thomas-mueller]`.

Considered: singular `source: kate` enforced by P6 (atomic). But atomicity
applies to the idea, not the attribution. A single piece of information can
genuinely emerge from a meeting with multiple participants. Singular would
require splitting notes that shouldn't be split.

---

### `from` is entities only — URLs go inline

**Decision:** `from` holds slugs (people, teams, groups). URLs are not stored
in `from`; they go inline in the body as markdown links.

Rationale: a URL is a reference, not attribution. Mixing slugs and URLs in one
field makes querying harder (`mem ls --from thomas-mueller` would need to filter
out URL strings) and blurs the semantic distinction between "who told you" and
"where you can read more."

---

### Field name: `from` not `sources`

**Decision:** `from`.

"I heard this from X" is the natural sentence. `from` is shorter, unambiguous,
and works equally for individuals and groups. `sources` carries a journalism
connotation (confidential informants, citations) that doesn't fit the use case.
Uniform with the CLI: `mem from` mirrors the field name, same as `mem tags`.

---

### Title is the filename slug — not repeated in body

**Decision:** `20260511T143022-kafka-rebalance.md` — the slug is the title.
No `# Heading` in the body, no `title:` frontmatter field.

A heading in the body duplicates the slug and requires keeping them in sync.
The filename is already the identity of the note. Body is content only.

---

### Cross-referencing: implicit + `[[wiki-links]]`

**Decision:** both. Implicit association via shared `tags` and `from` (always
free, zero syntax). Explicit `[[slug]]` links in body for intentional
connections (extracted to `links:` frontmatter at finalization).

Implicit alone misses connections the author explicitly has in mind. Explicit
alone requires discipline and adds friction. Combined: implicit handles the
common case, `[[slug]]` is available when the author wants to state a connection
directly.

---

### `mem serve` binds localhost by default

**Decision:** `mem serve` binds to `127.0.0.1` unless `--lan` is passed.

LAN exposure means anyone on the network can write your notes without
authentication. Opt-in only. `--token` adds a shared secret for `--lan` use —
sufficient for a trusted local network without adding sessions or OAuth.
Default `127.0.0.1` protects users who run `mem serve` without thinking about
network access.

---

### Semantic search requires explicit setup

**Decision:** `mem similar` errors clearly when the embeddings endpoint is
unreachable. No silent fallback to ripgrep.

Falling back would change the command's semantics — the user asked for semantic
results, not keyword results. A clear error with setup instructions is more
honest. The embedding model (default: ollama `nomic-embed-text`) is ~274MB and
runs on CPU — reasonable ask for a knowledge base tool.

---

### Inline extraction strips code and URLs first

**Decision:** before extracting `#tags`, `@from`, and `[[links]]`, strip fenced
code blocks, indented code blocks, inline code spans, and URLs from the body.

Prevents false positives: `#readme` in a GitHub URL, `@user` in an email
address, `[[bracket]]` in code examples. The extraction order is deterministic:
code blocks → inline code → URLs → extract. What remains is prose only.

---

### `links:` re-extracted fresh on every save

**Decision:** `[[slug]]` mentions are re-extracted from the body on every
finalization, same as `#tags` and `@from`. The body is the truth for links.

A link removed from the body disappears from `links:` on next save. A link
added inline appears in `links:` immediately. No manual frontmatter editing
needed or expected. Unresolved links (slugs that don't exist yet) are stored
silently — P7, the linked note may not exist yet.

---

### Frontmatter fields omitted when empty

**Decision:** all frontmatter fields (`tags`, `from`, `links`, `attachments`)
are omitted when empty. A note with no tags has no `tags:` line.

Considered: always writing `tags: []` as an explicit signal the note has been
processed. Rejected — the sandwich pattern guarantees finalization runs on every
save, so an empty list carries no useful information. Omitting empty fields
keeps frontmatter quiet and consistent across all fields.

---

### Index staleness uses mtime, not content hash

**Decision:** the index cache is considered stale when the notes directory mtime
is newer than the index file mtime. No content hashing.

A content hash would require reading every note file on every command — defeats
the purpose of the cache. The edge case where mtime is preserved on an in-place
edit (e.g. `cp --preserve`, deliberate mtime manipulation) is rare and
tool-specific. `mem index` is the documented escape hatch for when the cache
is wrong. Trust mtime; don't over-engineer staleness detection.

---

### `mem serve` form uses inline notation only

**Decision:** the capture form has a body textarea and an optional title field.
No separate tag or from input fields. Users write `#tag` and `@slug` inline in
the body, consistent with the editor writing model.

Separate fields would duplicate the inline notation and require maintaining two
input paths through FinalizeNote. Inline-only keeps the form minimal and the
mental model consistent.

---

### Note identifiers: slug or bare timestamp

**Decision:** everywhere a slug is accepted as an argument, a bare timestamp
(`20260519T143022`) is also accepted. Unnamed notes have no slug — the
timestamp is their only identifier.

`FindBySlug` becomes `FindByIdentifier`: checks if the argument matches a slug
first, then falls back to timestamp prefix match. Tab completion includes both
slugs and bare timestamps of unnamed notes.

---

### Attachment filenames use nanosecond timestamps

**Decision:** attachment files are named `<nanosecond-timestamp>-<filename>`,
e.g. `20260519T143022143000000-diagram.pdf`.

Nanosecond precision makes collisions practically impossible without hashing or
counters. The filename stays human-readable. No external dependencies.

---

### Slug collision on `mem new` errors and aborts

**Decision:** error with a helpful message, do not open the existing note or
append a disambiguator.

```
mem new kafka-rebalance
Error: note "kafka-rebalance" already exists — use: mem edit kafka-rebalance
```

Considered: silently opening the existing note (surprising — user said `new`,
not `edit`), appending `-2` (violates P4 — one slug per concept). If two things
are different enough to need separate notes, they need distinct names. The error
forces the human to be intentional. This is already the current behavior.

---

### Inbox and drafts are the same concept

**Decision:** no separate inbox directory. `mem new` with no title is the
inbox. Unnamed notes (timestamp-only filenames) are the inbox.

Considered: a separate `~/.mem/inbox/` with `mem inbox` subcommands. Rejected
— two storage locations means `mem search` and ripgrep must scan both, notes
can be in either place, and the human has to remember where to look. The
distinction adds complexity without adding value.

`mem new` with no title opens the editor immediately — zero-friction capture.
The human dumps the thought and closes. The note sits in `notes/` as a
timestamp-only file until promoted via `mem rename`.

`mem ls --unnamed` provides the focused review view: all timestamp-only notes
that still need a title, sorted oldest first as a processing queue.

The inbox as a mental model stays. The inbox as a directory does not exist.

---

### Non-interactive creation via `--body` and stdin

**Decision:** `mem new` accepts a `--body <text>` flag and reads from stdin when
not a TTY. Both skip the editor and run `FinalizeNote` directly. All other flags
(`-l`, `-f`) compose normally.

Rationale: the editor-based flow is designed for human capture. AI agents and
scripts need a programmatic path that still runs the full `FinalizeNote` pipeline
— UUID resolution, inline extraction, atomic write, index rebuild. Writing raw
files directly bypasses that and produces notes without UUIDs in frontmatter.

`--body` takes precedence over stdin. Stdin is the pipe-friendly form:

```sh
echo "Hooks run after tool calls — use PostToolUse for side effects. #claude-code" \
  | mem new claude-code-hook-timing
```

`--body` is the explicit form suited to AI tool calls:

```sh
mem new claude-code-hook-timing \
  --body "Hooks run after tool calls — use PostToolUse for side effects. #claude-code"
```

Both produce identical output. No new command needed — the flag is an input mode
switch on `mem new`, not a separate subcommand.

---

### `mem new` has no `--file` flag

**Decision:** file attachment is only possible via `mem attach` on an existing
note. `mem new` has no `--file` flag.

A `--file` flag on `mem new` creates an ordering problem: if the file is copied
before the editor opens and the user cancels, the attachment is orphaned in
`~/.mem/attachments/` with no note referencing it. If copied after, the
semantics are unclear (what if FinalizeNote fails?). Removing the flag
eliminates the problem entirely. Note creation and file attachment are separate
concerns — capture the thought first, attach the file after (P1, P6).

---

### Draft notes use timestamp-only filenames

**Decision:** `mem new` with no title creates `20260519T143022.md` — no slug
suffix. Promotion to a named note is done via `mem rename`.

Considered: `draft` as a slug suffix (`20260519T143022-draft.md`). Rejected —
`draft` is a fake name that encodes state rather than identity. Timestamp-only
is more honest: the note simply has no name yet. Collision-free by construction
(two drafts in the same second errors).

`mem ls` shows untitled entries by timestamp. `mem rename 20260519T143022
kafka-session-timeout` gives it a permanent slug. No special prompting in
`mem edit` — rename is explicit and deliberate.

Implementation note: `parseFilename` must handle the no-slug case (no hyphen
after the timestamp).

---

### UUID-based identity

**Decision:** notes, tags, and from-entities each carry a stable UUID. Prose
always uses human-readable slugs. Frontmatter stores UUIDs. The registry maps
UUID → current slug.

Renaming `thomas-mueller` to `tom-mueller` updates one registry entry. All
notes with `from: [uuid-of-thomas-mueller]` silently become correct — no files
touched. Prose references (`@thomas-mueller`) are cosmetically stale until the
user confirms a prose rewrite via `mem rename --from`.

This separates two concerns that were previously conflated: identity (UUID,
stable) and label (slug, changeable). F1 (stale wiki-links after note rename)
and F2 (from slug drift) are resolved by this model.

Considered: keeping slugs as identity (simpler, but F1/F2 persist), UUIDs
everywhere including prose (unreadable). Middle path: UUIDs in machine-managed
frontmatter only; prose stays human-readable and is the durable layer if
mem-cli disappears.

---

### Editor uses a temp file — note file never in intermediate state

**Decision:** `mem new` and `mem edit` write body-only to a temp file in `/tmp`
and open the editor on that. The note file is not touched until `FinalizeNote`
is ready to write the complete result. The final write is atomic: write to a
second temp path, then `os.Rename` into place (atomic on the same filesystem).

Considered: strip frontmatter from the note file in-place before opening the
editor (prior design). This created a crash window where frontmatter was
permanently lost if the editor exited abnormally. The temp-file approach
eliminates that window entirely — a crash at any point leaves the note file in
its last valid state.

This resolves F4 (no crash recovery) and F5 (non-atomic write) together.

---

### `mem ls` shows slug and date only

**Decision:** the fzf line is `<slug>\t<created>`. No tag or from columns.

Considered: displaying per-note tags and from in columns. Rejected for two
reasons. First, it requires O(N) file reads on every invocation — one per note
to read frontmatter. Second, the slug is already the title and carries
sufficient identity; tag and from columns add noise more than signal in a
picker. The cases where attribution or topic matter are exactly the cases where
`--tag` or `--from` pre-filtering is used — the columns are redundant with the
filters that already exist.

---

### Renaming: one command, always complete

**Decision:** `mem rename` always updates both the registry and all prose
references. No partial rename is offered. `mem mv` is removed.

Considered: offering a choice — update registry only (instant) vs update
registry + prose (rewrites files). Rejected because a declined prose rewrite
creates silent divergence: the next save of any note containing the old slug
in prose mints a new UUID for that slug, splitting the collection. There is no
safe "registry only" path. The confirmation prompt shows scope (which files
will be rewritten) but the outcome is always complete.

---

### Profile notes use `#profile` tag

**Decision:** `#profile` marks a note that documents what a `@slug` refers to.

Considered: `#contact` (implies a person), `#about` (reads oddly as a tag),
`#entity` (too technical). `#profile` is neutral — works for individuals,
groups, and teams without implying a relationship type. `mem ls -t profile`
lists all profile notes.

The tag is a convention, not enforced. Any note tagged `#profile` is treated
as a profile by tooling and by the user's own mental model.

---

### Plain files, not a database

**Decision:** notes are plain `.md` files. Metadata lives in frontmatter.
The index and registry are derived JSON files. No SQLite, no embedded DB for
the primary store.

At realistic personal scale (5k–20k notes, 3–5 captures/day over many years),
flat files with ripgrep are fast enough for every operation. The more important
reason is P2: an AI agent can read notes with nothing but filesystem access —
no mem-cli process, no DB connection, no schema. Moving metadata into a DB
would make mem-cli a required intermediary and break that guarantee.

The scalability problems that arise at large scale are solvable within the
flat-file model: IDF weighting for `mem related` (this file), pre-filtered
`mem ls`, year-based subdirectory sharding as opt-in config. A DB adds
operational complexity and a dependency that the tool's core value proposition
explicitly avoids.

The vector store (`.mem-vectors.db`) is the deliberate exception: embeddings
are binary blobs that have no plain-text representation, and SQLite is the
right tool for nearest-neighbor queries. It is opt-in and affects only
`mem similar`.

---

### `mem related` scoring

**Decision:** weighted scoring with IDF for tags, threshold ≥ 2 pts.

```
[[explicit link]]       10 pts      intentional — author stated the connection
shared from value        3 pts      strong signal — same origin
shared tag               IDF(tag)   scales with tag rarity
```

IDF per tag: `log(total_notes / notes_containing_tag)`

A tag on 3 of 1000 notes scores `log(333)` ≈ 5.8 — stronger than a shared
`from` value. A tag on 200 of 1000 notes scores `log(5)` ≈ 1.6. A tag on 900
of 1000 notes scores `log(1.1)` ≈ 0.1 — noise, effectively filtered by the
threshold. `total_notes` and `tag_counts` are read from the index cache; no
per-query file scanning needed.

Notes sorted descending by total score. Notes scoring < 2 suppressed.

Explicit links always surface regardless of metadata overlap. Rare tags
surface genuinely related notes. Common tags contribute proportionally less
as the collection grows — results stay meaningful at scale.

Backlinks are free: a note containing `[[kafka-rebalance]]` in its `links:`
field appears in `mem related kafka-rebalance` results as an explicit link,
without any annotation needed on `kafka-rebalance` itself.

---

## Binary

- Binary: `mem-cli`
- Alias: `mem` in `~/.shared/aliases`
- Module: `mem-cli`
- Repo: `~/git/hub/mem-cli`

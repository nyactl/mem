# Roadmap

Ideas ranked by impact on the core use case. Not a commitment.

---

## mem import

Migrate existing plain-markdown journals into mem's filename schema.

Old format (common): `2025-10-12_slug.md` — date only, no time, underscore separator.
mem format: `20251012T120000-slug.md` — local noon used as synthetic time when no
time is present in the file.

```
mem import ~/journal/          # dry-run by default: prints what would happen
mem import ~/journal/ --apply  # writes files to notes dir, skips duplicates
mem import ~/journal/ --apply --tag imported --tag journal
```

**What it does per file:**
1. Parse `YYYY-MM-DD_slug.md` (and `YYYY-MM-DD-slug.md`) from the filename
2. Read existing frontmatter — preserve tags, sources, attachments
3. Merge any `--tag` flags into the tags list
4. Write to `notesDir` as `YYYYMMDDTHHmmSS-slug.md` with the synthetic timestamp
5. Skip if a note with that slug and date already exists

**Multiple source dirs:** run the command once per dir. Tags distinguish origin:

```
mem import ~/journal/  --apply --tag journal
mem import ~/atlas/    --apply --tag atlas
mem import ~/notes-rewe/ --apply --tag rewe
```

**Does not delete the originals** — that is the user's decision after verifying.

---

## mem related

Show notes connected to a given note by shared tags, from values, and explicit
links.

```
mem related kafka-rebalance
```

Output:
```
kafka-consumer-offsets     explicit link (+10), tags: kafka (+1)   → 11 pts
partition-strategy         explicit link (+10), tags: kafka (+1)   → 11 pts
thomas-mueller             from: thomas-mueller (+3)               →  3 pts
kafka-consumer-offsets     tags: kafka (+1)                        →  1 pt
```

Scoring: `[[explicit link]]` = 10 pts, shared `from` value = 3 pts, shared tag
= 1 pt. Threshold ≥ 2 pts. Backlinks are free — any note with `[[kafka-rebalance]]`
in its `links:` field surfaces here without annotation on the target.

Complements `mem search` (exact terms) with association-based retrieval.

---

## mem append

Append text to an existing note's body without opening an editor.

```
mem append kafka-rebalance "Follow-up: confirmed fix works in staging. #resolved"
```

Appends as a new paragraph, runs the same finalization logic as `mem edit` —
inline tags and from values extracted and merged into frontmatter.

**Why this matters for P2 (shared external memory):** AI can already read notes
via files and the index. But writing requires replicating finalization logic.
`mem append` gives AI a clean, safe write interface: one command, no editor,
same frontmatter guarantees as any human-driven edit.

---

## Profile notes

`from` values are slugs — `thomas-mueller` appears in frontmatter and inline
`@thomas-mueller` mentions, but there's nowhere to record who or what that slug
refers to.

A profile is a regular mem note tagged `#profile`.

```
mem new thomas-mueller -l profile
```

Editor opens. You write:
```
Staff engineer at Acme. Primary contact for Kafka architecture questions.
Reliable on distributed systems, cautious on estimates.

Works with: @kate, @jin
```

The slug `thomas-mueller` in the filename is the identity. Discovery:

```
mem search @thomas-mueller    # all notes mentioning him, including his profile
mem get thomas-mueller        # view the profile
mem related thomas-mueller    # notes connected to him by tag or from values
mem ls -t profile             # all profile notes
```

No separate contacts store — P3 and P8. One place, all commands work for free.

---

## mem similar

Semantic search via local embeddings.

```
mem similar "kafka consumer timeout"
```

Embeds the query on the fly, matches against stored note vectors, returns a
ranked list. Requires ollama running locally with `nomic-embed-text` pulled
(~274MB, CPU-only). Endpoint configurable via `embeddings_url` in config.

Vectors stored in `~/.mem/notes/.mem-vectors.db` (SQLite). Generated lazily
on first call or explicitly via `mem index --embeddings`. Stale notes
(mtime newer than last-embedded) re-embedded automatically before search.

Complements `mem search` (exact) and `mem related` (structural). The three
together cover most retrieval cases: you know the words, you know the
connections, or you just remember the idea.

---

## paperless-ngx-cli attachment backend

Implement the paperless-ngx-cli attachment backend so `mem attach` uploads
documents to paperless instead of storing them locally.

```json
{ "attachment_backend": "paperless-ngx-cli" }
```

`mem attach kafka-rebalance diagram.pdf` → uploads via paperless-ngx-cli,
stores `paperless://1234` in `attachments:` frontmatter instead of a local
path. mem stays note-centric; paperless owns the document files.

The reverse direction — annotating a document already in paperless — needs no
integration. Create a mem note, put the paperless ID inline:

```
Q3 contract from @acme-team. paperless://1234 #invoice #legal

Watch clause 4.2 — liability cap lower than usual.
```

`mem search paperless://1234` finds it.

---

## Archiving records from external systems

External systems degrade retrieval once a record is closed. A closed ticket, a
completed task, an archived thread: the data usually still exists, but the paths
back to it — search, listings, filters — stop covering it. The record becomes
reachable only by an ID nobody remembers. Functionally that is the same as
losing it.

This is the case mem is already shaped for: the external system owns the
operational record, mem owns the story that has to outlive it.

**This is a convention, not an integration.** mem gains no knowledge of any
external tool — no import command, no adapter, no provider config. Any command
that prints text composes today:

```sh
mem new "account closure at the fund provider" \
  --body "$(some-cli show 1234)" \
  -t archive -t gdpr
```

`--body` already skips the editor, so nothing blocks and nothing new is needed
for the basic flow (P1, P8).

### Origin references go inline, not in frontmatter

The origin reference — a URL, or a `scheme://id` — belongs **in the note body,
verbatim**:

```
Closed 2026-08-12. Account may or may not still exist; login now returns a
generic error, which proves nothing either way.

Origin: https://example.invalid/app/task/1234
```

Not in `from`. `from` values are slugs by design (P4), and slugification
destroys a URL — `https://example.invalid/app/task/1234` becomes
`httpsexampleinvalidapptask1234`, which resolves to nothing and is not even
recognisable. This is the same choice already made for paperless references,
which live inline as `paperless://1234` and are found with `mem search`.

Two consequences worth stating: the body stays self-contained for a human
reading it with any text tool (P3), and no new frontmatter field is introduced
for a job an existing mechanism already does.

### One record, one note

Archive a record when there is a story worth keeping — a decision, a sequence
of attempts, a dated request, an outcome that surprised you. Not on every
closure. Bulk-exporting a backlog produces a write-only pile and violates P6;
it also recreates the exact problem the archive was meant to solve.

A useful test: would you be annoyed to re-derive this in eighteen months? If
not, close the record and write nothing.

### Reopening the thread

The value shows up when the thing resurfaces — a letter arrives, a charge
appears, someone asks. The user searches a term they remember, finds the note,
and appends what just happened. That is `mem append` (above), and this use case
is the strongest argument for it: an archived note is not a tombstone, it is a
thread that goes quiet and later resumes.

### Possible small addition

`--body -` to read the body from stdin. `--body "$(cmd)"` already works and is
adequate; stdin only matters for bodies large enough to strain argument limits,
or when the producing command streams. Low priority, and it stays agnostic —
mem reads text, it does not know who wrote it.

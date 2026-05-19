# Roadmap

Ideas ranked by impact on the core use case. Not a commitment.

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

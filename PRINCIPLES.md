P0 - Best in Class note taking
I want to achieve the best in class cli compatible note taking tool here.
Consider principles below as suggestions if a design choice clearly drives towards this P0.

P1 — Capture-first
Nothing should block the editor from opening. Title, tags, and source can all
emerge during or after writing — inline while thinking, or added later. The tool
adapts to the human, not the other way around.

P2 — Open by design
Notes are plain files. Any tool that can read a directory can read mem —
ripgrep, bat, editors, scripts. No API, no plugin, no session required.

P3 — Self-contained
The prose body of a note is self-contained for humans — it is what you write
and read, and remains meaningful with any text tool. Frontmatter is
machine-managed and not expected to be human-readable in isolation. The local
index and registry are derived caches; no external services are required.
Notes are the source of truth for prose; the registry is the source of truth
for stable identity.

P4 — Unified notation
One concept, one spelling. Tags, sources, and slugs all use the same format
(lowercase, hyphens) so the same person, topic, or idea is always found under
the same key across every note.

P5 — Durable storage
Plain text markdown with YAML frontmatter. The format predates and will outlive
mem. Notes remain readable and searchable with any text tool if mem
disappears.

P6 — Atomic
One note, one idea. Not a document, not a log. A note that covers multiple
unrelated things is harder to tag, harder to find, and harder to enrich.

P7 — Gradual enrichment
A note is valid the moment it is captured, even if sparse. Tags, sources, and
attachments can be added later. Depth accumulates over
time without requiring completeness upfront.

P8 — Composable
mem owns the store, not the workflow. It speaks plain text and integrates with
ripgrep, fzf, bat, and other CLIs without requiring any of them.

P9 — Local-first
Notes live on the user's machine. No cloud sync, no telemetry, no accounts.
The user decides if and how notes leave the device.

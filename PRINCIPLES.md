P1 — Capture-first
Nothing should block the editor from opening. Title, tags, and source can all
emerge during or after writing — inline while thinking, or added later. The tool
adapts to the human, not the other way around.

P2 — Shared external memory
mem is the common ground between user and AI. Both read and write notes the
same way — via the CLI or directly as files. No API, no plugin, no session
required. A note written by a human is immediately available to AI, and
vice versa.

P3 — Self-contained
Notes are the single source of truth. Tags, sources, and slugs are derived from
the notes themselves — no external backends, registries, or services required.
The local index is a performance cache derived from notes, never the authority.

P4 — Unified notation
One concept, one spelling. Tags, sources, and slugs all use the same format
(lowercase, hyphens) so the same person, topic, or idea is always found under
the same key across every note.

P5 — Durable storage
Plain text markdown with YAML frontmatter. The format predates and will outlive
mem-cli. Notes remain readable and searchable with any text tool if mem-cli
disappears.

P6 — Atomic
One note, one idea. Not a document, not a log. A note that covers multiple
unrelated things is harder to tag, harder to find, and harder to enrich.

P7 — Gradual enrichment
A note is valid the moment it is captured, even if sparse. Tags, sources, and
attachments can be added later — by the user or by AI. Depth accumulates over
time without requiring completeness upfront.

P8 — Composable
mem owns the store, not the workflow. It speaks plain text and integrates with
ripgrep, fzf, bat, and other CLIs without requiring any of them.

P9 — Local-first
Notes live on the user's machine. No cloud sync, no telemetry, no accounts.
The user decides if and how notes leave the device.

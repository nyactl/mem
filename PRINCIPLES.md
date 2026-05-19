P1 — Capture-first
Nothing should block the editor from opening. Title, tags, and source can all
emerge during or after writing — inline while thinking, or added later. The tool
adapts to the human, not the other way around.

P2 — Shared external memory
mem is readable and writable by both the user and AI. Plain text + structured
frontmatter means no special access layer is needed. Notes written by a human
are immediately useful to AI, and vice versa. Offline access is always possible.

P3 — Self-contained
Notes are the single source of truth. Tags, sources, and slugs are derived from
the notes themselves — no external backends, registries, or services required.
The local index is a performance cache derived from notes, never the authority.

P4 — Unified notation
Tags, sources, and slugs share the same slug format (lowercase, hyphens).
Completion enforces consistency so the same person or concept is always spelled
the same way across all notes.

P5 — Durable storage
Plain text markdown with YAML frontmatter. The format predates and will outlive
mem-cli. Notes remain readable and searchable with any text tool if mem-cli
disappears.

P6 — Atomic
One note, one idea. Not a document, not a log. A note that covers multiple
unrelated things is harder to tag, harder to find, and harder to enrich. When
in doubt, split.

P7 — Gradual enrichment
A note is valid the moment it is captured, even if sparse. Tags, sources, and
attachments can be added later — by the user or by AI. Depth accumulates over
time without requiring completeness upfront.

P8 — Composable
mem is one tool in a chain, not a monolith. It speaks plain text and works with
ripgrep, fzf, bat, and other CLIs. It does not own the workflow — it owns the
store.

P9 — Local-first
Notes live on the user's machine. No cloud sync, no telemetry, no accounts.
The user decides if and how notes leave the device.

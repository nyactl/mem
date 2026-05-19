P1 — Capture-first
mem should demand as little upfront organisation as possible. Title is optional;
#tags and @sources can be written inline while thinking; the slug is derived
after writing. The tool adapts to the human, not the other way around.

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

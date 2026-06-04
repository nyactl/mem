# mem-cli

One idea per note. Plain Markdown. Yours forever.

Capture a fact in one command, find it in one more. No app, no account, no sync
service — just files in `~/.mem` that open in any editor and survive any tool
switch.

---

## Install

```sh
make install
```

Requires Go. Optionally: `fzf`, `bat`, `rg` (ripgrep) for the full experience.

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

**Filename:** `20260511T143022-<slug>.md`

Timestamp is the sole source of creation time — not repeated in frontmatter.

**Frontmatter:**

```markdown
---
tags: [kafka, rebalance]
source: kate
attachments: [/Users/you/.mem/attachments/20260511T143022-diagram.png]
---

Consumer group rebalance blocks all partitions for ~2min with the default eager
protocol. Fix: switch to cooperative-sticky assignor.
```

- `tags` — always present, may be empty `[]`
- `source` — omitted if none; person name or URL
- `attachments` — omitted if none
- Nothing else — no `created`, no `updated`, no `title`

---

## Commands

### `mem new <title> [-l <tag>] [-f <file>]`

Create a new note. Opens `$EDITOR` after creation.

- `-l/--label <tag>` — repeatable, tab-completes from tag backend
- `-s/--source <value>` — person name or URL
- `-f/--file <path>` — repeatable; copies file into `~/.mem/attachments/`

### `mem get [<slug>]`

View a note. No slug opens fzf picker with bat preview.

### `mem edit [<slug>]`

Open a note in `$EDITOR`. No slug opens fzf picker.

### `mem ls [--tag <tag>]`

Browse all notes via fzf with bat preview. `--tag/-t` pre-filters by tag.

### `mem search <query>`

Full-text search via ripgrep across `~/.mem/notes/`.

### `mem tags`

List all tags with note counts, merged from notes and tag backend.

### `mem attach <slug> <file> [<file>...]`

Attach one or more files to an existing note.

---

## Tag backend

Tags are presented as mem's own. Source is configurable in
`~/.config/mem/config.json`:

```json
{ "tag_backend": "todoist-cli labels" }
```

Any command producing `id\tname` or `name` per line works. Falls back to tags
from existing notes if the backend is unavailable.

---

## Planned

### `mem append <slug> <line>`

Append a timestamped line to an existing note without opening the editor. Intended for maintenance log notes — one persistent note per entity (a car, a device, a health record) that accumulates entries over time:

```
mem append car-maintenance "winter tires fitted, 87,432 km, next ~Nov"
```

Appends:
```
2026-06-02 — winter tires fitted, 87,432 km, next ~Nov
```

This requires a log note convention — a note whose body is a list of dated entries rather than a single atomic fact. The frontmatter format stays the same.

### `@log` label integration (via glue script)

A convention where Todoist tasks tagged `@log` trigger a `mem append` on completion. The task description holds the target mem slug. A personal glue script — separate from both todoist-cli and mem-cli — reads the completed task, extracts the slug, and calls `mem append`.

mem-cli stays fully standalone. The integration is owned by the glue layer, not by either tool.

### Open design questions

- How should mem surface notes related to a set of Todoist tasks without manual tag lookup?
- Should domain tags be defined in Todoist (as labels) and pulled by mem, or maintained independently in mem?
- When paperless-ngx attachment backend lands, how do documents link back to Todoist tasks (e.g. a receipt to a Finance task)?
- What is the stable format for embedding a mem slug in a Todoist task description, so a glue script can parse it reliably?

---

## Attachment backend

```json
{ "attachment_backend": "local" }
```

**`local` (default)** — copies to `~/.mem/attachments/`, stores absolute path.

**`paperless-ngx-cli` (future)** — uploads via paperless-ngx-cli, stores
document reference in frontmatter.

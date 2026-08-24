# mem

Shared external memory for human and AI. Atomic, tagged notes for facts,
findings, commands, and ideas — captured quickly, looked up fast, readable
offline or with AI assistance.

Fully independent. No dependency on any other tool.

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

## Attachment backend

```json
{ "attachment_backend": "local" }
```

**`local` (default)** — copies to `~/.mem/attachments/`, stores absolute path.

**`paperless-ngx-cli` (future)** — uploads via paperless-ngx-cli, stores
document reference in frontmatter.

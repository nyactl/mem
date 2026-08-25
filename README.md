# mem

Atomic, tagged notes for facts, findings, commands, and ideas —
captured quickly, looked up fast, readable anywhere.

---

## Philosophy

Most note-taking tools treat notes as living documents: editable titles,
movable folders, endless reorganisation. mem takes the opposite position.

**A note is a snapshot.** The moment you capture something, that record
is fixed. The slug in the filename — derived from the title you give at
creation — is the permanent identity of that note. It never changes, even
if you rewrite the body. The only operation that retires a note is delete.

This is not a limitation. It is the point.

- **No rewriting history.** You can refine the content, fix a typo, add a
  tag — but the *name* of what you captured stays anchored to the moment
  you captured it.
- **Sync is always safe.** Because IDs never change, push and pull never
  produce ambiguous renames. A note is either there or it isn't.
- **Retrieval over organisation.** Instead of spending energy reorganising,
  you search. `mem search` and `mem ls --tag` surface what you need without
  a hierarchy to maintain.
- **Plain files.** Every note is a Markdown file. No database, no lock-in.
  Read, grep, or back up with any tool that understands files.

If an idea evolves, write a new note and reference the old one. The old
record remains true to the moment it was written.

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
date: 2026-05-10
tags: [kafka, rebalance]
sources: [alice]
attachments: [/Users/you/.mem/attachments/20260511T143022-diagram.png]
---

Consumer group rebalance blocks all partitions for ~2min with the default eager
protocol. Fix: switch to cooperative-sticky assignor.
```

- `tags` — always present, may be empty `[]`
- `date` — optional; overrides the filename timestamp for display and `mem day` filtering. Accepts a date (`2026-05-10`) or datetime (`2026-05-10T09:30`). Use when capturing something that happened in the past — a dream, an experience from last week, a backdated log entry. The filename timestamp still records when you actually captured it.
- `sources` — omitted if none; person name or URL
- `attachments` — omitted if none
- Nothing else — no `created`, no `updated`, no `title`

---

## Commands

### `mem new <content>  |  mem new <title> <content>`

Capture a note instantly from the command line — no editor opens.

```sh
mem new "kafka rebalance blocks all partitions for ~2min #kafka"
mem new "kafka rebalance" "consumer group blocks all partitions for ~2min"
mem new "dream about the mountains" "we were climbing..." -d 2026-08-19
```

- One argument: slug derived from first few words of content
- Two arguments: first is the title (sets the slug), second is the body
- `-t/--tag <tag>` — repeatable
- `-s/--source <value>` — person name or URL
- `-f/--file <path>` — attach file, repeatable
- `-d/--date <date>` — override display date, e.g. `2026-08-19` or `2026-08-19T09:30`

### `mem edit [<slug>]`

Open a note's body in `$EDITOR`. No slug opens fzf picker.

### `mem ls [<slug>] [--tag <tag>]`

Browse notes via fzf with bat preview. Pass a slug to view directly. `--tag/-t` pre-filters by tag.

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

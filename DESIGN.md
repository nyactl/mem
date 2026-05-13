# ref-cli

A personal reference tool. Atomic, tagged notes for facts, findings, commands,
and ideas — captured quickly, looked up fast.

Fully independent. No dependency on ryo or any other tool.

---

## Storage

```
~/.ref/
  notes/          # markdown note files
  attachments/    # copied attachment files
```

Config: `$XDG_CONFIG_HOME/ref/config.json` → `~/.config/ref/config.json`

---

## File format

**Filename:** `20260511T143022-<slug>.md`

Timestamp prefix is the sole source of creation time — it does not appear again
in frontmatter or file content.

**Frontmatter:**

```markdown
---
tags: [kafka, rebalance]
attachments:
  - /Users/you/.ref/attachments/20260511T143022-rebalance-diagram.png
---
Consumer group rebalance blocks all partitions for ~2min with the default eager
protocol. Fix: switch to cooperative-sticky assignor.
```

- `tags` — always present, may be empty `[]`
- `attachments` — omitted if none; absolute paths; multiple allowed
- Nothing else — no `created`, no `updated`, no `title`

---

## Commands

### `ref new <title> [-l <tag>] [-f <file>]`

Create a new note.

- Title is required — no args = error
- Slug derived from title: `"Kafka Rebalance Blocks Partitions"` → `kafka-rebalance-blocks-partitions`
- `-l/--label <tag>` — repeatable, tab-completes from tag backend
- `-f/--file <path>` — repeatable; copies file to `~/.ref/attachments/<timestamp>-<filename>`,
  adds absolute path to frontmatter `attachments` list
- Opens `$EDITOR` after file is created

### `ref get [<slug>]`

View a note.

- No slug → fzf picker over all notes, bat preview pane
- With slug → opens in bat (pager mode); falls back to less, then cat
- Tab-completes slugs

### `ref edit [<slug>]`

Open a note in `$EDITOR`.

- No slug → fzf picker
- Tab-completes slugs

### `ref ls [--tag <tag>]`

Browse notes interactively via fzf.

- Invokes fzf with bat preview pane
- `--tag/-t` — pre-filters by tag before passing to fzf
- Each line passed to fzf: `<slug>\t<tags>\t<created>`
- Tab-completes tags

### `ref search <query>`

Full-text search via ripgrep across `~/.ref/notes/`.

### `ref tags`

List all tags with note counts.

Output: `<tag>\t<count>` — includes tags from notes and from the configured backend.

### `ref attach <slug> <file>`

Attach a file to an existing note.

- Copies file to `~/.ref/attachments/<note-timestamp>-<filename>`
- Appends absolute path to note's frontmatter `attachments` list
- Note timestamp is parsed from the note's filename

---

## Attachment backend

Attachment storage is pluggable. The backend receives a file and returns a
reference stored in the note's frontmatter `attachments` list.

`~/.config/ref/config.json`:
```json
{
  "attachment_backend": "local"
}
```

**`local` (default)** — copies file to `~/.ref/attachments/<timestamp>-<filename>`,
stores absolute path in frontmatter.

**`paperless-ngx-cli` (future)** — uploads file via paperless-ngx-cli, stores
document ID or URL in frontmatter. Enables paperless-ngx as the document store
for all ref attachments.

The frontmatter reference format depends on the backend:
```yaml
# local
attachments:
  - /Users/you/.ref/attachments/20260511T143022-diagram.png

# paperless-ngx (future)
attachments:
  - paperless://1234
```

ref-cli never reads the attachment content — it only stores and displays the
reference. Opening an attachment is delegated to the appropriate tool.

---

## Tag backend

Tags are presented as ref-cli's own. The source is configurable.

`~/.config/ref/config.json`:
```json
{
  "tag_backend": "todoist-cli labels"
}
```

ref-cli shells out to the configured command and parses `id\tname` output.
Default: `todoist-cli labels`. Any command producing that format works.
If the backend is unavailable, ref-cli falls back to tags found in existing notes.

Tab completion for `-l/--label` calls the backend at completion time.

---

## Binary

- Binary: `ref-cli`
- Alias: `ref` in `~/.shared/aliases`
- Module: `ref-cli`
- Repo: `~/git/hub/ref-cli`

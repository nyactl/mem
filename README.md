# mem-cli

One idea per note. Plain Markdown. Yours forever.

Capture a fact in one command, find it in one more. No app, no account, no sync
service — just files in `~/.mem` that open in any editor and survive any tool
switch.

---

## Why

Your notes shouldn't live in someone else's infrastructure. Notion can go down,
Roam can pivot, subscriptions can lapse. mem-cli writes plain `.md` files to a
directory you own. Back them up with rsync, search them with ripgrep, read them
in vim — no mem-cli process required, ever.

Capture is a single command from anywhere in the terminal. Notes stay atomic:
one idea, one file. When you need something, `mem search` or bare `rg` gets you
there without opening a browser tab.

---

## Install

```sh
make install
```

Requires Go. Optionally: `fzf`, `bat`, `rg` (ripgrep) for the full experience.

---

## Commands

### `mem new [<title>] [-t <tag>] [-f <person>] [--body <text>]`

Create a note. Opens `$EDITOR`. Inline tags (`#kafka`) and attribution (`@kate`)
in the body are extracted automatically on save.

`--body` skips the editor — useful for scripts or piped input:

```sh
mem new kafka-rebalance --body "Eager protocol blocks all partitions ~2min. Fix: cooperative-sticky. #kafka"
echo "..." | mem new kafka-rebalance
```

No title → timestamp-only file, promote later with `mem rename`.

### `mem get [<identifier>]`

View a note. No identifier opens an fzf picker with bat preview.

### `mem edit [<identifier>]`

Edit a note in `$EDITOR`. No identifier opens an fzf picker.

### `mem ls [--tag <tag>] [--from <person>] [--unnamed]`

Browse notes via fzf. `--unnamed` shows only untitled notes, oldest first — the
processing queue for fast captures that need a title.

### `mem search <query>`

Full-text search via ripgrep across `~/.mem/notes/`.

### `mem rename <old> <new>`

Rename a note. Rewrites all `[[old-slug]]` references across every note.

### `mem attach <identifier> <file>`

Copy a file into `~/.mem/attachments/` and link it to a note.

### `mem tags` / `mem from`

List all tags or attribution values with note counts.

### `mem index`

Rebuild the index cache manually — needed after edits outside mem-cli.

---

## File format

**Filename:** `20260511T143022-kafka-rebalance.md`

The slug is the title. No heading in the body, no `title:` field.

**Frontmatter** (tool-managed, never shown in the editor):

```markdown
---
tags: [kafka, consumer-group]
from: [kate]
---

Consumer group rebalance blocks all partitions for ~2min with the default eager
protocol. Fix: switch to cooperative-sticky assignor. Confirmed by @kate.
```

Fields omitted when empty. You write the body; the tool manages the metadata.

---

## Configuration

`~/.config/mem/config.json` — all fields optional, defaults shown:

```json
{
  "notes_dir":       "~/.mem/notes",
  "attachments_dir": "~/.mem/attachments",
  "embeddings_url":  "http://localhost:11434"
}
```

---

## Log notes

Some notes aren't atomic facts — they're running histories. A maintenance record, a health log, a habit journal. One note per entity, body is a list of dated entries:

```
2026-06-04 — gym, upper body, 45 min
2026-06-02 — gym, legs, 40 min
2026-05-30 — gym, upper body, 50 min
```

Use `mem edit <slug>` to append manually, or pipe via `--body` in a script:

```sh
mem new gym-log --body "2026-06-04 — upper body, 45 min #health"
```

No streak visualization — that belongs in a dedicated habit tracker. What mem gives you is a permanent, offline, searchable record that survives any app switch.

See [ROADMAP.md](ROADMAP.md) for planned features.

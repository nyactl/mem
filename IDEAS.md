I1: is the tool native to how a human stores information and thinks? [done]
I2: support multiple sources for one note. [done — from: [] list]
I3: what format should be sources, especially names (last and first name(s)) [done — slug, e.g. thomas-mueller]
I4: which concepts can be imported from other note taking apps (e.g. apple notes) [done — daily notes conflict P6, nested tags deferred, mem serve + mem similar added to ROADMAP.md]
I5: how to handle file captures? [done — mem attach + pluggable backend, paperless-ngx-cli backend in ROADMAP.md]

---

Open questions:

Q1: what tag marks a profile note? [done — #profile, see Design Decisions in DESIGN.md]

Q2: mem rename / mem mv — how do they work? [done — see Design Decisions in DESIGN.md]

Q3: how is cross-referencing done? [done — implicit via tags/from + explicit [[slug]], see Design Decisions in DESIGN.md]

Q4: draft lifecycle [done — timestamp-only filename, mem rename for promotion, see Design Decisions in DESIGN.md]

Q5: inbox vs draft — are these the same concept? [done — merged, see Design Decisions in DESIGN.md]

Q6: slug collision on `mem new <title>` [done — error and abort, see Design Decisions in DESIGN.md]

Q7: `mem related` scoring [done — see Design Decisions in DESIGN.md]

Q8: `tags: []` always written vs omit when empty [done — omit when empty, see Design Decisions in DESIGN.md]

Q9: mem index command [done — manual index rebuild + --embeddings flag, see DESIGN.md]
Q10: mem search output format [done — slug + matching line with context, see DESIGN.md]
Q11: mem serve form behavior [done — empty title → unnamed note, inline confirmation, see DESIGN.md]
Q12: nvim integration [done — documented in Tags section of DESIGN.md]

Q13: mem serve LAN protection [done — --token shared secret, 403 on mismatch, see DESIGN.md]
Q14: mem ls fzf key bindings [done — Enter=view, Ctrl-E=edit, Ctrl-R=rename, see DESIGN.md]
Q15: mem get vs mem edit fzf picker [done — shared picker, action differs on selection, see DESIGN.md]
Q16: config schema [done — full schema with defaults documented, see DESIGN.md]

Q17: flag-only notes [done — valid if flags supplied, deleted only if truly empty, see DESIGN.md]
Q18: mem search scope [done — searches full file including frontmatter, see DESIGN.md]
Q19: exit codes [done — 0/1/2/130 table documented, see DESIGN.md]
Q20: mem ls empty directory [done — helpful message to stderr, exit 0, see DESIGN.md]

Q21: mem edit on unnamed note [done — bare timestamp accepted as identifier, see DESIGN.md]
Q22: mem rename on unnamed note [done — FindByIdentifier accepts slug or timestamp, see DESIGN.md]
Q23: attachment filename collision [done — nanosecond timestamp prefix, see DESIGN.md]
Q24: fzf preview [done — body only, tags/from visible in fzf columns, see DESIGN.md]

Q25: mem mv --yes flag [done — --yes/-y skips prompt, non-TTY without --yes exits 2, see DESIGN.md]
Q26: mem serve form format [done — inline notation only, no separate tag/from fields, see DESIGN.md]
Q27: timestamp completions [done — shown with first body line as context, see DESIGN.md]
Q28: index staleness [done — mtime sufficient, mem index is escape hatch, see DESIGN.md]

Q29: does `mem new <title>` slugify the title automatically, or does the user pass a slug directly?
[proposed] Auto-slugify. `mem new Kafka Session Timeout` → `kafka-session-timeout.md`. Consistent with `mem rename` behavior. Passing a pre-slugified title still works (idempotent). This matches P1 — capture-first, no friction.

Q30: does `mem mv` update `from:` frontmatter values in other notes, not just inline `@slug` and `[[slug]]` in bodies?
[proposed] Yes. `mem mv` should rewrite all three: `@slug` in prose, `[[slug]]` in prose, and `from: [slug]` in frontmatter. A person-slug rename that misses frontmatter leaves `mem ls --from old-slug` returning stale results. The confirmation prompt lists affected files regardless of which field matched.

Q31: `mem search <query>` with no results — what output and exit code?
[proposed] Print nothing to stdout, exit 0. Consistent with grep/rg conventions: no match is not an error. Callers checking for empty output can use `if mem search foo | grep -q .`. Exit 1 is reserved for errors (file unreadable, ripgrep not found).

Q32: `mem rename <old> <new>` where the new slug already exists — error or what?
[proposed] Error and abort, same as `mem new` on collision: `Error: note "kafka-rebalance" already exists`. Exit 1. Renaming onto an existing slug would silently overwrite it — never correct.

Q33: `mem edit` or `mem get` when `$EDITOR` is not set — what happens?
[proposed] Error with a clear message: `Error: $EDITOR is not set — set it in your shell profile (e.g. export EDITOR=vim)`. Exit 1. No silent fallback to a hardcoded editor; that would violate P8 (composable — mem speaks the environment's conventions).

Q34: `FinalizeNote` when the editor exits with a non-zero code (e.g. vim `:cq`) — save or discard?
[proposed] Discard. A non-zero exit from the editor is the user's explicit signal that the edit was abandoned. Save anyway would lose the user's intent. For `mem new`: delete the newly created file (as if the editor was never opened). For `mem edit`: restore the original content from the pre-edit backup taken during the sandwich strip.


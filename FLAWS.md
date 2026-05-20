# Flaws

Known limitations and correctness issues. F1–F3 are structural, F4–F5 are bugs,
F6–F10 found in second audit pass.

---

F1: Slug identity is unstable [resolved]
  Resolved by UUID-based identity. Notes, tags, and from-entities each carry
  a stable UUID in frontmatter. The registry maps UUID → current slug. Renaming
  updates the registry and always rewrites prose — no partial rename is offered.
  Residual limitation: flag-supplied tags/from without a prose counterpart
  cannot be fully reconstructed from notes alone if the registry is lost;
  mem index remints UUIDs in that case, making frontmatter links stale until
  notes are re-saved.

F2: from slug drift for real names [resolved]
  Resolved by UUID-based identity and mem rename --from. thomas-mueller and
  t-mueller being different entries is now correctable: mem rename --from
  t-mueller thomas-mueller updates the registry (all notes silently correct)
  and optionally rewrites prose with confirmation. New captures are still
  steered by completion; fast captures that bypass completion produce new
  UUIDs, which mem rename can merge.

F3: mem related scoring degrades at scale [resolved]
  Resolved by IDF weighting. Tag score is now log(total_notes/notes_with_tag)
  instead of a flat 1pt. Common tags contribute near-zero; rare tags contribute
  more than a shared from-value. tag_counts and total_notes are stored in the
  index cache — no per-query file scanning. Results stay meaningful as the
  collection grows.

F4: sandwich pattern has no crash recovery [resolved]
  Resolved by temp-file editing. The editor opens a body-only temp file in
  /tmp; the note file is never modified until FinalizeNote is ready to write
  the complete result. A crash at any point leaves the note file untouched.

F5: FinalizeNote write is not atomic [resolved]
  Resolved together with F4. FinalizeNote writes to a second temp path then
  calls os.Rename into place — atomic on the same filesystem. No partial
  writes are possible.

F6: mem ls requires O(N) file reads for per-note tag/from display [resolved]
  Resolved by dropping the tag/from columns. mem ls fzf line is now
  <slug>\t<created> only — a pure directory listing, no file reads. The slug
  is the title and carries sufficient identity for browsing. Tag and from
  context is available via --tag/--from pre-filtering and the fzf preview pane.
  Caching per-note data in the index was rejected: the sync is free but the
  columns add noise more than signal in a picker where the slug already
  identifies the note.

F7: prose/registry diverge silently after a declined rename [resolved]
  The option to decline prose rewrite after a registry rename was removed.
  mem rename always rewrites both registry and prose. A confirmation prompt
  shows scope before writing; --yes skips it. No partial path exists, so
  divergence cannot occur.

F8: --file attachment copied before note exists [bug]
  mem new --file copies the file to attachments dir, but the spec does not
  say when relative to the editor session. If copying happens before the
  editor opens and the editor is cancelled, the attachment sits in
  ~/.mem/attachments/ with no note referencing it.

F9: no mem delete
  No command removes a note. Junk captures from mem serve, accidental
  mem new invocations, and processed inbox notes cannot be cleaned up.

F10: @slug extracted from bare email addresses [bug]
  The extraction safety strips https?:// URLs but not bare email addresses.
  contact@kate.io in prose extracts @kate as a from-entity. noreply@github.com
  extracts @github. False positives pollute the from index.

# Flaws

Known limitations and correctness issues. F1–F3 are structural, F4–F5 are bugs.

---

F1: Slug identity is unstable [resolved]
  Resolved by UUID-based identity. Notes, tags, and from-entities each carry
  a stable UUID in frontmatter. The registry maps UUID → current slug. Renaming
  updates the registry only; frontmatter across all notes stays correct without
  touching any file. Prose rewrite is optional and confirmed via mem rename.
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

F3: mem related scoring degrades at scale
  Common tags (#backend, #kafka) appear on 100+ notes.
  1pt-per-shared-tag has no frequency weighting.
  A note sharing three common tags scores 3pts alongside genuinely related notes.
  No TF-IDF equivalent. Results become noisy beyond ~500 notes.

F4: sandwich pattern has no crash recovery [resolved]
  Resolved by temp-file editing. The editor opens a body-only temp file in
  /tmp; the note file is never modified until FinalizeNote is ready to write
  the complete result. A crash at any point leaves the note file untouched.

F5: FinalizeNote write is not atomic [resolved]
  Resolved together with F4. FinalizeNote writes to a second temp path then
  calls os.Rename into place — atomic on the same filesystem. No partial
  writes are possible.

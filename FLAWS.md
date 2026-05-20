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

F4: sandwich pattern has no crash recovery [bug]
  If $EDITOR crashes, frontmatter has been stripped but never restored.
  Note left with body only — tags, from, links silently lost.
  No recovery path specified.

F5: FinalizeNote write is not atomic [bug]
  os.WriteFile writes directly to the file.
  Process killed mid-write leaves file truncated or corrupt.
  Fix: write to temp file, then os.Rename (atomic on same filesystem).

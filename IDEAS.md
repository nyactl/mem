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


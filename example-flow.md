# Example Flows

---

## Quick capture → named note

You're in a meeting. Something comes up. No time to think about title or tags.

```
mem new
```

Editor opens. You dump the thought raw:

```
thomas said kafka consumer groups have a 3-second session timeout by default
when using the new heartbeat protocol — worth checking
```

Close. Done. File sits in `~/.mem/notes/20260519T091204.md` — no frontmatter,
no slug, nothing imposed.

---

Later at your desk, review unnamed notes:

```
mem ls --unnamed
```

fzf opens with all unnamed notes, oldest first, bat preview. You pick the one
above and rename it:

```
mem rename 20260519T091204 kafka-session-timeout-default
```

Then edit to enrich it:

```
mem edit kafka-session-timeout-default
```

```
@thomas-mueller mentioned kafka consumer groups default to 3s session timeout
with new heartbeat protocol. #kafka #backend

Worth verifying — could explain the rebalance spikes we saw.
```

Close. Result on disk:

**`~/.mem/notes/20260519T091204-kafka-session-timeout-default.md`**
```
---
tags: [kafka, backend]
from: [thomas-mueller]
---
@thomas-mueller mentioned kafka consumer groups default to 3s session timeout
with new heartbeat protocol. #kafka #backend

Worth verifying — could explain the rebalance spikes we saw.
```

---

## Cross-referencing with wiki-links

You're writing a note about Kafka session timeouts and want to explicitly
connect it to a related note you already have.

```
mem edit kafka-session-timeout-default
```

Body in editor:

```
@thomas-mueller confirmed session timeout defaults to 3s with the new heartbeat
protocol. This directly affects [[kafka-rebalance]] behaviour — a timeout spike
triggers a full rebalance. See also [[partition-strategy]]. #kafka #backend
```

Close. `FinalizeNote` extracts `[[kafka-rebalance]]` and `[[partition-strategy]]`
and writes them to frontmatter.

**Result on disk:**
```
---
tags: [kafka, backend]
from: [thomas-mueller]
links: [kafka-rebalance, partition-strategy]
---
@thomas-mueller confirmed session timeout defaults to 3s with the new heartbeat
protocol. This directly affects [[kafka-rebalance]] behaviour — a timeout spike
triggers a full rebalance. See also [[partition-strategy]]. #kafka #backend
```

Now `mem related kafka-session-timeout-default` scores connections:

```
kafka-rebalance       explicit link (+10), shared tag kafka (+1)  → 11 pts
partition-strategy    explicit link (+10), shared tag kafka (+1)  → 11 pts
thomas-mueller        shared from (+3)                            →  3 pts
kafka-consumer-offsets shared tag kafka (+1)                      →  1 pt
```

And from the other direction — `mem related kafka-rebalance` surfaces
`kafka-session-timeout-default` as a backlink, because its `links:` field
contains `kafka-rebalance`. No annotation needed on `kafka-rebalance` itself.

---

## Profile note

You reference `@thomas-mueller` across many notes but there's nowhere to record
who he is. Create a profile note:

```
mem new thomas-mueller -l profile
```

Editor opens. You write what you know:

```
Staff engineer at Acme. Primary contact for Kafka architecture questions.
Reliable on distributed systems, cautious on estimates.

Works with: @kate, @jin
```

Close. Result on disk:

**`~/.mem/notes/20260519T094512-thomas-mueller.md`**
```
---
tags: [profile]
from: [kate, jin]
---
Staff engineer at Acme. Primary contact for Kafka architecture questions.
Reliable on distributed systems, cautious on estimates.

Works with: @kate, @jin
```

The slug `thomas-mueller` in the filename is the identity — no heading inside
the file. From here:

```
mem search @thomas-mueller     # every note that mentions him, including this one
mem get thomas-mueller         # view the profile
mem related thomas-mueller     # notes sharing his tags or from values
mem ls -t profile              # all profile notes
```

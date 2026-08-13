# mem — user stories and constraints

Job-story format for stories. `DONE` lines are testable acceptance conditions.
Status: collecting. Nothing here is a solution decision.

---

## Stories

### S1 — Dream capture on waking

WHEN   I wake up and remember a dream
I WANT to capture it before it fades
SO I   can review patterns months later

DONE
- capture completes in under ~30 seconds, one-handed, on a phone
- works with no network
- works without unlocking or opening a laptop
- reaches the same store as desktop notes, with no re-filing later
- capture time recorded automatically

IMPLIED
- no decisions at capture time — no folder choice, no mandatory tagging
- text-first; voice a bonus, not the primary path
- "patterns months later" ⇒ dreams must be isolatable and searchable in aggregate

### S2 — Offline capture, later sync

WHEN   I am offline on any device
I WANT to still create and edit notes
SO I   never lose a thought waiting for connectivity

DONE
- notes are created and edited with no network at all
- on reconnect, changes reach the central instance without manual action
- no data loss and no silent overwrite when two devices edited while apart

### S3 — Terminal-native work on macOS  *(hard requirement)*

WHEN   I am at my Mac
I WANT to edit and search notes with nvim and ripgrep directly
SO I   keep my existing terminal workflow

DONE
- notes are plain files in a directory on the local filesystem
- `rg <term>` over that directory returns hits in note bodies
- `nvim <file>` edits a note in place, with no export/import step
- no proprietary container, database, or blob store in the path

### S4 — A finished task becomes a durable note

WHEN   I close a task that carried a story worth keeping (attempts, dates,
       correspondence, an outcome that surprised me)
I WANT its description and timeline captured into mem automatically
SO I   can close it in the task manager without losing the history, and find it
       again when the thing resurfaces months later

DONE
- closing a task marked for archiving produces one note, without further action
- the note carries the task description and its timeline: creation, comments
  with their dates, completion date
- the origin reference (task URL/id) is in the note body, verbatim
- tasks not marked for archiving produce nothing — this is opt-in, not a dump

IMPLIED / OPEN
- trigger mechanism undecided: webhook from the task manager, or a poll by an
  external integration. A webhook needs a publicly reachable endpoint or a
  tunnel, which is real infrastructure for a personal tool.
- **mem must not learn what Todoist is.** P8/C6: mem exposes a generic
  note-ingestion API; the task-manager-specific part lives outside mem (in the
  task CLI, or a tiny integration). Otherwise mem grows a vendor dependency.
- DEPENDENCY: retrieving a *completed* task's content is currently broken in
  todoist-cli (issue #16 — `ls --done` and `stats` report zero for every
  period). This story cannot be built against that CLI until it is fixed.
- attachments on the source task (image-only comments) are not retrievable
  through the task CLI today — see C1.

---

## Constraints

### C1 — Media as first-class content
Captured media (photos, audio notes, video) is stored alongside text, not
separately managed by hand.

### C2 — Encryption at rest  *(see TENSION-1)*
Text and media are sensitive and should be protected at rest where possible.

### C3 — Central source of truth
Many devices, many capture sources. One authoritative instance that all devices
converge on. (Note: this reverses an earlier recommendation of remote-as-
meeting-point; the user has decided for a central instance.)

### C4 — Mature VCS as storage backend
Version control for the text files, chosen for stability and proven conflict
resolution. Git unless there is a strong reason otherwise.

### C5 — Uniform frontmatter, tool-independent
Every entry carries the same metadata header so entries outlive the tool.
Already mem's P3/P5. Plain markdown + YAML frontmatter.

### C6 — Standalone and shareable as open source
- No dependency on the user's personal setup — explicitly NOT chezmoi, which is
  not installed everywhere.
- Simple releases. Single artefact.
- No wrapper scripts, no "extra tool for this and that". Functionality belongs
  in the tool, not in glue around it.

---

## Resolutions (decided 2026-08-13)

### R1 — Storage is plaintext; encryption is volume-level
The repository is **unencrypted**. Files stay plain markdown so ripgrep (S3),
git three-way merge (C4) and tool-independence (C5) all work — per-file
encryption would have broken all three.

"At rest" is satisfied by the underlying volume: **FileVault verified On** on
this Mac; the server and phones must likewise use full-volume/device
encryption. Transport is encrypted separately (TLS or VPN).

Residual exposure, accepted: anyone with access to a *running, unlocked* device
or server reads everything. The mitigation is convention, not cryptography —
credentials, full account numbers and tokens are never stored in notes, only
referenced (e.g. "siehe Enpass").

### R2 — Sync belongs inside the tool; no wrappers
No shell wrappers, no launchd agents, no chezmoi provisioning. If sync is a
requirement it ships in the binary.

Removed on 2026-08-13: the `mem-sync` script, its launchd agent and plist, and
both chezmoi source files. The note store at `~/.mem` remains, git-backed, but
**nothing commits automatically any more** — that is now `mem`'s job to provide.

### R3 — Git is the server's storage backend, not the sync protocol
Devices sync to the central instance over the instance's own protocol. The
instance commits to git. This gives history and three-way conflict resolution
without requiring a git client on a phone, and keeps offline capture (S2)
independent of git's model.

### R4 — Separate server and client internally; ship one binary
Not two products. A *mode*, not a split artefact.

- `mem new` / `edit` / `search` — operate directly on the local store. No server
  required, offline by construction.
- `mem serve` — hosts the PWA, exposes the sync and ingestion APIs, commits to
  git. The central instance (C3) is simply an instance running in this mode.

**Why not a thin client:** S3 forces it. nvim and ripgrep operate on the
filesystem, not through an API, so every machine used for real work holds actual
files. The "client" is therefore a full local replica, not a view onto a remote
store. What exists is peers, one of which is designated central.

**Why one binary:** C6. One artefact, one release, nothing extra to install.
Two binaries would mean two installs, two release processes and a client/server
compatibility matrix — pure cost for a single-user tool.

**Internally, do separate them:** distinct packages for store, sync, HTTP and
CLI, with the store package knowing nothing about HTTP. Ordinary hygiene, and it
keeps a genuine split available later without paying for it now.

Follows from this:
- **Embed the PWA in the binary** (Go `embed`). Static assets shipped separately
  to deploy would be exactly the "extra tool for this and that" C6 forbids.
- **S4 tests the boundary**: ingestion is a serve-mode API and the CLI needs
  none of it. A feature that only makes sense in one mode means the line is in
  the right place.

---

## Original tensions (for the record)

### TENSION-1 — C2 (encryption at rest) vs S3 (ripgrep, hard requirement)
`rg` cannot search encrypted files. Per-file encryption (age, git-crypt) makes
S3 impossible. These are only compatible if "at rest" means **volume-level**
encryption — FileVault on macOS, an encrypted dataset/LUKS volume on the server
— so files are plaintext while mounted and unreadable when the disk is at rest.

Since S3 is a hard requirement, the reading must be: full-disk/volume
encryption plus TLS in transit, NOT per-file encryption. Needs confirmation.

Residual exposure under that reading: anyone with access to a *running* server
or an unlocked device reads everything. Mitigation is convention, not crypto —
secrets (credentials, full account numbers, tokens) never enter notes; they are
referenced (e.g. "siehe Enpass").

### TENSION-2 — C6 (no wrappers) retires earlier work
The chezmoi provisioning script and the launchd + `mem-sync` shell script built
on 2026-08-12 are exactly the "wrapper and extra tools" C6 rules out. If sync
is a requirement, it belongs inside the tool. That earlier setup is superseded.

### TENSION-3 — C4 (git backend) vs S2 (mobile offline sync)
Git is a poor sync protocol for phones. Resolvable: git is the *server's*
storage backend; devices sync to the central instance over its own protocol,
and the instance commits. Git then provides history and conflict resolution
without a git client on the phone.

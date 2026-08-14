# Architecture decisions

This file captures every non-obvious decision made during implementation so
future sessions can pick up context without reading the full source.

---

## 1. One binary, two modes

**Decision:** `mem` is a single binary. Local note commands (`new`, `edit`, `ls`,
`search`, …) work with no server. `mem serve` activates the server mode.

**Why:** C6 — no wrappers, no extra tools, single release artefact. Two binaries
would mean two install steps, two release pipelines, and a client/server version
matrix. A mode flag costs nothing.

**Trade-off:** The binary embeds the PWA, so it is larger than a pure CLI. Acceptable
for a personal tool; not a concern for size-sensitive environments.

---

## 2. PWA embedded via `go:embed`, not a native app

**Decision:** The mobile client is a PWA served by `mem serve`. Static assets live
in `internal/server/pwa/` and are embedded at build time (`//go:embed pwa/*`).

**Why (per R4):** Embedding means one binary = one install = one release. A native
app is a separate deliverable with its own store submission, signing, and update
cycle — exactly the "extra tool" C6 forbids for the initial release.

**Known limits on iOS:**
- PWA install requires Safari "Add to Home Screen" (not App Store)
- Push notifications require iOS 16.4+ and the user to manually enable them
- Background sync (`sync` event) is not supported on iOS Safari — the app falls
  back to a 30-second polling interval while the page is open

**Accepted for MVP.** A native app can be built later against the same API without
changing the server.

---

## 3. No WebDAV — custom REST API over HTTP

**Decision:** The sync protocol is a simple REST API (`/api/notes`, `/api/delta`,
`/api/ingest`). WebDAV was considered and rejected.

**WebDAV problems:**
- No push; clients must poll (same limitation as the custom API, but no upside)
- Whole-file transfers — no way to send only changed fields
- No built-in conflict detection
- Complex server setup (Nextcloud, Apache modules, etc.)
- iOS/Android WebDAV client libraries are poorly maintained
- Server always sees cleartext filenames and metadata even with volume encryption

**Custom API advantages:**
- Server can be ~200 LOC Go with no external deps
- We control conflict detection (ETags + 409)
- Clean separation: server stores encrypted volumes; API is the only surface
- `POST /api/ingest` for external integrations (S4) maps naturally

---

## 4. ETag-based optimistic concurrency (not CRDTs)

**Decision:** Each note has an ETag (FNV-64 of file content, hex-encoded). `PATCH`
requires an `If-Match` header; a stale ETag returns 409 with the current ETag in
the response header. The client (PWA or `mem sync`) marks the note as conflicted
for manual resolution.

**Why not CRDTs:**
CRDTs solve automatic merge for concurrent text edits. For a personal tool used by
one person across 2-3 devices, true concurrent edits on the same note are rare. The
complexity of a CRDT library (encoding, tombstones, vector clocks) is not justified.
Last-write-wins would be simpler but lossy. ETag + manual resolution is the right
middle ground.

**Conflict UX:**
- PWA: note is marked with ⚠ conflict badge; user opens the note and edits to resolve
- CLI: `mem sync pull/push` prints the conflicting IDs; user edits locally

---

## 5. Git is the server's storage backend, not the sync protocol

**Decision:** The server commits every write to git in the notes directory. Clients
sync via the REST API. Devices do not need a git client.

**Why:** TENSION-3 from REQUIREMENTS.md: git is a poor sync protocol for phones —
clones, rebases, and merge conflicts are not phone-friendly workflows. But git
provides exactly what we want on the server: history, three-way merge capability,
and a proven conflict resolution model for future use.

**Commit message format:** `note: add {id} {slug}` / `note: update {id} {slug}` /
`note: delete {id} {slug}` / `ingest: {id} {slug}`.

---

## 6. Volume-level encryption only — no per-file encryption

**Decision:** Files are plain markdown on disk. Encryption is the responsibility of
the underlying volume (FileVault on macOS, LUKS or encrypted dataset on the server).

**Why (TENSION-1 from REQUIREMENTS.md):** `rg` (ripgrep) cannot search encrypted
files. S3 — terminal-native search — is a hard requirement. Per-file encryption (age,
git-crypt) is incompatible with ripgrep. The only compatible reading is that "at rest"
means volume-level, not file-level.

**Residual exposure accepted:** anyone with access to a running, unlocked device reads
everything. Mitigation is convention: credentials and tokens are never stored in notes
(`see Enpass`), only referenced.

---

## 7. Ingest API is generic — mem knows nothing about Todoist

**Decision:** `POST /api/ingest` accepts `{title, body, tags, from}`. The caller
provides the content; mem stores it like any other note. The Todoist-specific logic
(fetching task description + comments, detecting `#archive` label, authenticating to
Todoist API) lives in `todoist-cli` or a tiny adapter script — outside mem.

**Why (C6/P8):** Adding a Todoist dependency to mem would:
- couple mem to a third-party API that can change
- force mem to carry auth credentials for an external service
- prevent mem from being used with other task managers

The boundary: mem exposes a write surface (`/api/ingest`); integrations own the
read logic for their source.

---

## 8. Note identity: `{timestamp}-{slug}` (full filename minus extension)

**Decision:** The canonical note identifier is the full filename without `.md`:
`{15-char-ts}-{slug}`, e.g. `20260814T120000-alpha`. Both parts travel together.

**Why (not just timestamp):** Two notes created within the same second would share
a pure-timestamp ID — `FindByID` prefix scan would return the wrong note and trigger
false conflicts. Embedding the slug makes every ID unique at creation time.

**In filenames:** `{ts}-{slug}.md`. The ID *is* the filename stem.

**API path:** `/api/notes/{id}` where `id = note.ID()` = `{ts}-{slug}`.

**`FindByID`:** exact match — `filepath.Join(dir, id+".md")`. No prefix scan needed.

**Rename:** renaming a note changes its slug → changes its ID → new file, old file
deleted. The server handles this by storing the new ID; the client must delete the
old local file on the next pull if it no longer appears in the remote listing.

---

## 9. Auth: bearer token or none

**Decision:** If `auth_token` is set in config, every `/api/*` request must carry
`Authorization: Bearer {token}`. If unset, no auth is applied (suitable for
localhost / VPN access).

**Why not OAuth:** This is a personal single-user tool. OAuth is for delegating
access to third parties. A pre-shared token is simpler, auditable, and sufficient.

**Token storage:** config file (`~/.config/mem/config.json` or `$MEM_CONFIG`). The
config file should be `chmod 600`. Never stored in the binary or environment in
production — the environment variable `MEM_CONFIG` points to the file.

---

## 10. Filename timestamps are local time; wire format is UTC

**Decision:** Note filenames use local time (`time.Now()` or `t.Local()`). The API
sends and receives UTC (`time.RFC3339` with `Z` suffix). The server converts incoming
UTC to local before constructing the filename.

**Why:** Filename timestamps are how a human looks up notes (`ls ~/.mem/notes/`).
Local time is more legible. UTC on the wire is unambiguous across timezones.

**Implication for offline capture:** The PWA uses `new Date().toISOString()` (UTC).
The server converts to local on receipt. If the phone is in a different timezone than
the server, the filename timestamp reflects the _server's_ local time, not the phone's.
This is acceptable for a personal single-server tool.

---

## 11. Sandbox

**Location:** `.sandbox/` in the repo root, gitignored.
**Config:** `.sandbox/config.json` — points notes dir at `.sandbox/notes/`.
**Usage:** `MEM_CONFIG=.sandbox/config.json ./mem-cli <command>` or `make sandbox-serve`.

The sandbox's `notes/` directory is itself a git repo (separate from the mem-cli
repo). This mirrors production: the server's notes dir is always git-backed.

To reset the sandbox: `rm .sandbox/notes/*.md`.

---

## 12. TLS — nginx reverse proxy, not native Go TLS

**Decision:** `mem serve` binds to `127.0.0.1:4747` (loopback only). nginx
terminates TLS on port 443 and proxies to it. Cert renewal is handled by Certbot.

**Why not native Go TLS:** Go's `ListenAndServeTLS` requires a cert path and a
restart on renewal — Certbot post-deploy hooks can do this, but it adds ceremony.
nginx with Certbot is a solved, self-renewing stack that every Linux host already
knows how to operate. Keeping TLS out of the Go binary also means the binary works
on LANs (no cert needed) without any flag changes.

**Why not a container:** Docker adds a runtime dependency and an extra process
boundary. The goal is a single binary on the server host, managed by systemd.

**nginx site config** (`/etc/nginx/sites-available/mem`):

```nginx
server {
    listen 80;
    server_name mem.example.internal;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name mem.example.internal;

    ssl_certificate     /etc/letsencrypt/live/mem.example.internal/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/mem.example.internal/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         HIGH:!aNULL:!MD5;

    location / {
        proxy_pass         http://127.0.0.1:4747;
        proxy_set_header   Host $host;
        proxy_set_header   X-Real-IP $remote_addr;
        proxy_read_timeout 60s;
    }
}
```

**systemd unit** (`/etc/systemd/system/mem.service`):

```ini
[Unit]
Description=mem note server
After=network.target

[Service]
Type=simple
Environment=MEM_CONFIG=/etc/mem/config.json
ExecStart=/usr/local/bin/mem-cli serve --addr 127.0.0.1:4747
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

**Setup checklist:**
1. `certbot --nginx -d mem.example.internal` (or DNS challenge for private domains)
2. `systemctl enable --now mem`
3. Set `server_url` in the client config to `https://mem.example.internal`
4. Generate a token: `openssl rand -hex 32` → set `auth_token` on server and client

**LAN-only alternative:** skip nginx and TLS entirely; run `mem serve` bound to
the LAN interface and access via Tailscale or WireGuard. The bearer token is still
required; the VPN provides transport security.

---

## Open decisions (not yet resolved)

| # | Question | Options | Blocking? |
|---|---|---|---|
| O1 | Mobile tech stack | PWA (current) vs SwiftUI vs Flutter | No — PWA ships in v1; native is an upgrade path |
| O2 | `mem sync` ETag baseline | **Resolved.** `.mem-sync-state.json` in the notes dir stores last-known server ETag per note. Pull compares local ETag, last-known, and server ETag to distinguish safe overwrite from true conflict. | — |
| O3 | Todoist integration trigger | Webhook (needs public endpoint) vs polling | No — deferred; depends on todoist-cli issue #16 fix |
| O4 | License | MIT vs AGPL | Before first public release |
| O5 | Push notifications on mobile | Requires VAPID, service + infra | After PWA baseline is stable |

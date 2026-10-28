# Changelog

## [0.6.0] - 2026-10-28

### Bug Fixes

- Mem new creates the notes directory, so the first note on a new machine is saved (#35)

### Features

- Notes keep their title as typed in a title field, shown, searched and synced in place of the slug (#34)
- Mem g looks up terms in a glossary of notes, exact matches first, and mem g add captures one (#34)


## [0.5.0] - 2026-10-27

### Bug Fixes

- Mem edit keeps a note's date, and its frontmatter when the editor fails (#30)

### Features

- Mem ls renders notes with their mermaid diagrams, and its picker edits, deletes and supersedes in place (#30)
- The mem ls picker shows a note and comes back, lists notes by the date they are sorted by, and fits narrow terminals (#30)
- The picker's preview names each diagram in one line, the full view draws it (#30)


## [0.4.0] - 2026-10-27

### Bug Fixes

- Update vite, dompurify and source-map-js to patched versions and rebuild the web UI
- Mem auth login reads the token without showing it
- A sync rejected for its token points to mem auth login
- The web UI's packages come from the public npm registry, also where another registry is configured

### Features

- Search by terms, tags and date with JSON output, superseding notes, and transliterated slugs (#25)


## [0.3.0] - 2026-10-03

### Features

- Store the auth token in the system keychain


## [0.2.2] - 2026-10-01

### Bug Fixes

- Override lodash-es to a patched version across the tree (#12)
- Name the cause when the server rejects a sync

### Documentation

- List the config mem sync consults in its help


## [0.2.1] - 2026-09-30

### Bug Fixes

- Keep dagre layout and classic look after the mermaid 12 upgrade
- Store note dates in utc so they survive sync unchanged

### CI

- Add a temporary tap token check
- Drop the temporary tap token check
- Keep the node base image on the lts line

### Documentation

- Verify images against the exact signing workflow and tag


## [0.2.0] - 2026-09-22

### Bug Fixes

- ID collision, ETag casing, sync conflict detection, body size limit
- Ls picker shows full timestamp, slug, then #tags
- Snap slug uses first 4 words, strips #tags and @sources
- Snap rejects empty or whitespace-only text
- Track local ETag in sync state to eliminate false conflicts on pull

### CI

- Target cmd/mem and build the frontend in the pipelines
- Keep dependencies and pinned base images current with dependabot
- Build the frontend with node 24
- Read the smoke-test notes repo as root

### Documentation

- Add nginx TLS deployment to ARCHITECTURE.md
- Add mem import to roadmap
- Remove AI framing; reposition P2 as open-by-design
- Update command help text
- Document running and verifying the server image

### Features

- Add mem serve, mem sync, embedded PWA, sandbox
- Add mem snap and mem day for moment capture and day review
- Scaffold Svelte 5 frontend, replace hand-written pwa
- Immutable slugs, combine new+snap, absorb get into ls, date-aware mem day
- Add mem delete command
- Mem sync with no args runs pull then push
- Add note editing in web UI
- Delete note in web UI
- Add mem related command
- Add mem --version
- Read config overrides from MEM_* environment variables
- Publish a signed multi-arch container image


## [0.1.0] - 2026-06-05

### CI

- Add release pipeline, goreleaser, cliff, license

### Documentation

- Rewrite README opener
- Rewrite README with sovereignty framing; drop tag backend
- Add log notes section and mem append preview
- Move planned content to ROADMAP, link from README
- Fix README to match implemented commands and flags
- Remove feature names from ROADMAP reference

### Features

- Implement core commands




# Changelog

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




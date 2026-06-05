# Releasing

## Semver guidelines

| Bump | When |
|------|------|
| **Major** `v2.0.0` | Breaking changes — renamed or removed commands, changed flag semantics, output format changes scripts may depend on |
| **Minor** `v0.x.0` | New commands, new flags, new features — fully backward-compatible |
| **Patch** `v0.1.x` | Bug fixes, performance improvements, dependency updates, documentation |

When in doubt between minor and patch: if a user has to change how they invoke the tool, it's minor. If not, it's patch.

## Pre-release checklist

- [ ] `go vet ./...` — clean
- [ ] `go build ./...` — clean
- [ ] `go test ./...` — passes
- [ ] GoReleaser dry run — `make snapshot`
- [ ] README reflects any new commands or flags

## Stable release

```sh
git tag v0.x.0
git push origin v0.x.0
```

Release tags (`v*`) are protected against deletion and force-pushes.

GitHub Actions takes it from there:
1. Build and vet run — release is blocked if they fail
2. GoReleaser builds binaries, `.deb`, `.rpm`, and tarballs
3. GitHub Release is published with all artifacts signed via cosign
4. Homebrew formula in `nyactl/homebrew-tap` is updated automatically
5. `CHANGELOG.md` is committed back to `main`

After the workflow completes, replace GoReleaser's auto-generated notes with human-written ones:

```sh
gh release edit vX.Y.Z --notes "..."
```

## Pre-release / RC

```sh
git tag v0.2.0-rc.1
git push origin v0.2.0-rc.1
```

GoReleaser marks it as a pre-release — not set as "Latest". Homebrew tap is not updated for RCs. RC entries are excluded from `CHANGELOG.md`. Promote to stable once validated:

```sh
git tag v0.2.0
git push origin v0.2.0
```

## Verifying release signatures

Releases are signed with [cosign](https://github.com/sigstore/cosign) via keyless signing on Sigstore. Each release publishes `checksums.txt` and `checksums.txt.sigstore.json`.

```sh
VERSION=v0.1.0

curl -LO https://github.com/nyactl/mem-cli/releases/download/${VERSION}/checksums.txt
curl -LO https://github.com/nyactl/mem-cli/releases/download/${VERSION}/checksums.txt.sigstore.json

cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity "https://github.com/nyactl/mem-cli/.github/workflows/release.yml@refs/tags/${VERSION}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  checksums.txt

curl -LO https://github.com/nyactl/mem-cli/releases/download/${VERSION}/mem_${VERSION#v}_darwin_arm64.tar.gz
sha256sum --check --ignore-missing checksums.txt
```

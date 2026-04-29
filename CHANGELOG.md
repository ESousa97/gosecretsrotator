# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Full project documentation: `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `LICENSE` (MIT), and `docs/ARCHITECTURE.md`.

### Changed

- CI: bumped `golangci/golangci-lint-action` from `v8` to `v9` and pinned the binary to `v2.11.4`.
- CI: added a minimal `.golangci.yml` (required by golangci-lint v2) using the standard linter set.

## [0.4.0] — 2026-04-29

### Added

- **Webhooks**: every successful rotation now POSTs a JSON payload to `GOSECRETS_WEBHOOK_URL` (Slack/Discord/custom).
- **Prometheus metrics** exposed by the daemon on `/metrics`:
  - `gosecrets_rotations_total{status}`
  - `gosecrets_secrets_expiring_soon`
  - `gosecrets_last_rotation_success{secret_name}`
- **`incident rollback`** subcommand that restores a secret to the previous successful value using the SQLite history.
- **SQLite history database** (`history.db`) recording every rotation and rollback attempt with status and error message.

### Changed

- Migrated Docker SDK from the deprecated `docker/docker` import path to `github.com/moby/moby/client`.
- Dropped `WithAPIVersionNegotiation` (now the default) and switched container creation to the new `client.ContainerCreateOptions` shape.

### Fixed

- Errors from `HistoryDB.Record` and `HistoryDB.Close` are now logged instead of silently dropped.
- `fmt.Sscanf` return value in `config.LoadConfig` is checked, falling back to the default port on parse failure.

## [0.3.0] — 2026-04-26

### Added

- **Rotation engine** in `internal/rotation`:
  - Cryptographically secure password generator (`crypto/rand`, 74-char alphabet, default length 32).
  - `RotateSecret` applies the new value to all attached `Target`s atomically per secret.
  - `DueSecrets` selects secrets whose `IntervalDays` has elapsed.
- **`daemon` command** with `--check-interval` flag; graceful shutdown on `SIGINT` / `SIGTERM`.
- **`target add docker | file`** subcommands to register propagation targets per secret.
- **`rotation set --days`** command to configure per-secret rotation policy.
- GitHub Actions CI workflow (`build`, `vet`, `gofmt`, `test -race`, `golangci-lint`) and Dependabot config.

### Changed

- `Secret` struct extended with `IntervalDays` and `Targets`; `Store.Load` keeps backwards compatibility with the legacy `map[string]string` vault shape.

## [0.2.0] — 2026-04-26

### Added

- **`inject docker`** subcommand that recreates a container with an updated environment variable (preserves Config, HostConfig, and network endpoints; refuses Compose-managed containers).
- **`inject file`** subcommand for `.env` (line-level rewrite preserving inline comments) and `.yaml`/`.yml` (AST-based update preserving structure).
- `internal/providers/docker` and `internal/providers/file` packages.

### Fixed

- Docker SDK build error resolved by upgrading dependencies and pinning `docker/go-connections`.

## [0.1.0] — 2026-04-26

### Added

- Initial CLI skeleton with Cobra: `add`, `get`.
- Encrypted vault: AES-256-GCM with PBKDF2-SHA256 key derivation from `GOSECRETS_MASTER_PWD`.
- Vault file written with `0600` permissions; nonce prepended to ciphertext.
- Backwards-compatible JSON vault shape.

[Unreleased]: https://github.com/esousa97/gosecretsrotator/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/esousa97/gosecretsrotator/releases/tag/v0.4.0
[0.3.0]: https://github.com/esousa97/gosecretsrotator/releases/tag/v0.3.0
[0.2.0]: https://github.com/esousa97/gosecretsrotator/releases/tag/v0.2.0
[0.1.0]: https://github.com/esousa97/gosecretsrotator/releases/tag/v0.1.0

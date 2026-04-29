# Contributing to GoSecretsRotator

Thanks for your interest! This is a study project, so contributions of any size — from a typo fix to a new provider — are welcome.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Development Setup](#development-setup)
- [Project Layout](#project-layout)
- [Running the Test Suite](#running-the-test-suite)
- [Linting & Formatting](#linting--formatting)
- [Commit Style](#commit-style)
- [Pull Request Checklist](#pull-request-checklist)
- [Adding a New Target Provider](#adding-a-new-target-provider)
- [Reporting Bugs](#reporting-bugs)
- [Reporting Security Issues](#reporting-security-issues)

## Code of Conduct

Be respectful, assume good intent, prefer concrete feedback over judgement. That's the whole policy.

## Development Setup

Requirements:

- Go **1.25.0** or newer (the CI pins `1.25`)
- A C toolchain (`gcc` / `clang`) — required by the SQLite history driver and the race detector
- Docker daemon — only needed if you touch the Docker provider

```bash
git clone https://github.com/esousa97/gosecretsrotator.git
cd gosecretsrotator

# Pull deps and build
go mod download
CGO_ENABLED=1 go build -o gosecretsrotator .

# Sanity check
./gosecretsrotator --help
```

A throwaway vault is handy while developing:

```bash
export GOSECRETS_MASTER_PWD='dev-only-do-not-reuse'
./gosecretsrotator add EXAMPLE_KEY 'example-value'
```

## Project Layout

```text
gosecretsrotator/
├── main.go                   # entry point
├── cmd/                      # Cobra command tree (one file per top-level command)
├── internal/
│   ├── config/               # env-var parsing
│   ├── crypto/               # AES-GCM + password generator
│   ├── storage/              # encrypted vault + SQLite history
│   ├── rotation/             # rotation engine, metrics, webhook
│   └── providers/
│       ├── docker/           # Docker container env update
│       └── file/             # .env / .yaml writers
└── tests/                    # end-to-end CLI tests
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the deeper walkthrough.

## Running the Test Suite

```bash
# Quick (no race, works without CGO)
go test ./...

# Full (race detector — what CI runs)
CGO_ENABLED=1 go test -race -count=1 ./...

# With coverage
go test -cover ./...

# Verbose
go test -v ./...
```

The end-to-end tests under `tests/` shell out to `go run ../main.go ...` — they assume the working directory is `tests/`.

## Linting & Formatting

CI runs:

1. `go mod tidy` (and fails on diff)
2. `gofmt -l .` (must be empty)
3. `go vet ./...`
4. `go build ./...`
5. `go test -race -count=1 ./...`
6. `golangci-lint run` (v2.11.4 with the [`.golangci.yml`](.golangci.yml) defaults)

Run them locally before pushing:

```bash
go mod tidy
gofmt -w .
go vet ./...
go build ./...
CGO_ENABLED=1 go test -race -count=1 ./...

# Install once
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4
golangci-lint run --verbose
```

If `golangci-lint` complains about something you can't (or shouldn't) fix, prefer narrowing the scope with `//nolint:linter // reason` over disabling the linter globally.

## Commit Style

This repo follows [Conventional Commits](https://www.conventionalcommits.org/):

- `feat(rotation): support multi-target atomic rollback`
- `fix(docker): preserve labels when recreating container`
- `chore(deps): bump moby/moby/client to 0.5.0`
- `docs(readme): clarify CGO requirement`
- `test(file): cover quoted '#' in .env values`
- `ci: pin golangci-lint to v2.11.4`

Keep the subject under 70 characters; let the body explain the *why*. **Do not add a `Co-Authored-By` trailer for AI assistants.**

## Pull Request Checklist

Before opening a PR:

- [ ] CI passes locally (`go vet`, `gofmt`, `go test -race`, `golangci-lint run`)
- [ ] New code is covered by a test where it makes sense (especially in `internal/crypto`, `internal/storage`, `internal/providers`)
- [ ] User-facing changes are reflected in `README.md` and/or `CHANGELOG.md`
- [ ] No real secrets, tokens, or `.env` contents committed
- [ ] No binaries committed (`gosecretsrotator`, `*.exe`, `secrets.json`, `history.db` are `.gitignore`d)
- [ ] Commit messages follow Conventional Commits

In the PR description, include:

1. **What** changed in one sentence.
2. **Why** — the motivation or the bug it fixes.
3. **How to verify** — the commands a reviewer should run.

## Adding a New Target Provider

To add, say, a Kubernetes Secret provider:

1. **Create the package**: `internal/providers/k8s/k8s.go` exposing a function with a clear contract, e.g. `UpdateSecret(namespace, name, key, value string) error`.
2. **Extend the `Target` struct** in `internal/storage/storage.go` with any new fields (`Namespace`, `Name`, …) — make them `omitempty`.
3. **Wire `ApplyTarget`** in `internal/rotation/rotation.go` to dispatch on the new `target.Type`.
4. **Add a CLI subcommand** in `cmd/target.go` (`target add k8s ...`) plus a one-shot in `cmd/inject.go` if useful.
5. **Tests** — at minimum a unit test for the writer; an integration test if practical.
6. **Docs** — update the [Providers](README.md#providers) and [Rotation Targets](README.md#rotation-targets) sections of the README.

## Reporting Bugs

Open a GitHub issue with:

- The command(s) you ran
- What you expected vs. what happened
- The output of `gosecretsrotator --help` and `go version`
- Whether `secrets.json` / `history.db` already existed (don't paste their contents)

A minimal reproduction is worth 1 000 logs.

## Reporting Security Issues

**Do not open a public issue for security problems.** Follow the process in [SECURITY.md](SECURITY.md).

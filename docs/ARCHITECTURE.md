# Architecture

A deeper walkthrough of how GoSecretsRotator is wired together. Read [the README](../README.md) first for the high-level overview; this document is for contributors who want to extend the rotation engine, add a new provider, or understand the security boundaries.

## Table of Contents

- [Goals & Non-Goals](#goals--non-goals)
- [Module Layout](#module-layout)
- [Data Model](#data-model)
- [Rotation Lifecycle](#rotation-lifecycle)
- [Rollback Lifecycle](#rollback-lifecycle)
- [Storage Layer](#storage-layer)
- [Crypto Layer](#crypto-layer)
- [Provider Contract](#provider-contract)
- [Daemon Loop](#daemon-loop)
- [Observability](#observability)
- [Error Handling Strategy](#error-handling-strategy)
- [Backwards Compatibility](#backwards-compatibility)

## Goals & Non-Goals

### Goals

- **Local-first.** No external services required to run the core rotation flow.
- **Single binary.** Static-ish Go binary (CGO is needed only for SQLite); no agents, no sidecars.
- **Pluggable propagation targets.** Adding a new sink (Kubernetes Secret, AWS SSM, Hashicorp Vault) is a matter of writing a small adapter and one `case` in `ApplyTarget`.
- **Auditability.** Every rotation and rollback ends up as a row in the SQLite history database, with status and (on failure) the error message.
- **Operational simplicity.** All configuration via environment variables and CLI flags — no config file required.

### Non-Goals

- HSM / KMS-backed master keys. The master key is derived locally from `GOSECRETS_MASTER_PWD`.
- Multi-tenant access control. There is one vault per process.
- Distributed coordination. Two daemons pointing at the same vault file will race.
- Production-grade key derivation parameters. PBKDF2 iterations and salt are intentionally simple — see [SECURITY.md](../SECURITY.md#known-limitations).

## Module Layout

```text
main.go                        # main() → cmd.Execute()
cmd/                           # Cobra commands (one file per top-level verb)
  ├── root.go
  ├── add.go         get.go
  ├── rotation.go              # rotate + rotation set
  ├── target.go                # target add (docker|file)
  ├── inject.go                # inject parent
  ├── inject_docker.go
  ├── inject_file.go
  ├── daemon.go                # background runner + /metrics server
  └── incident.go              # incident rollback
internal/
  ├── config/                  # env-var parsing
  ├── crypto/                  # AES-GCM seal/open + password generator
  ├── storage/                 # encrypted vault + SQLite history
  ├── rotation/                # engine, metrics, webhook
  └── providers/
      ├── docker/              # container env update
      └── file/                # .env / .yaml writers
tests/                         # end-to-end CLI integration tests
```

The dependency arrows always point **inward** from `cmd/` towards `internal/`. Providers and storage know nothing about Cobra, environment variables, or HTTP — they receive plain Go arguments.

## Data Model

```go
// Target is where a rotated value must land.
type Target struct {
    Type      string // "docker" | "file"
    Container string // docker only
    EnvKey    string // docker only
    Path      string // file only
    FileKey   string // file only
}

// Secret is the canonical record stored in the vault.
type Secret struct {
    Value        string
    LastRotated  time.Time
    IntervalDays int      // 0 disables auto-rotation
    Targets      []Target
}

// Store is the in-memory representation of secrets.json.
type Store struct {
    filePath string
    password string                // master password (in memory only)
    Secrets  map[string]*Secret
}

// HistoryEntry is one row in history.db.
type HistoryEntry struct {
    ID         int
    SecretName string
    Value      string
    RotatedAt  time.Time
    Status     string  // "success" | "failure"
    ErrorMsg   string
    Operation  string  // "rotation" | "rollback" | "baseline"
}
```

The vault on disk is the GCM-encrypted JSON marshalling of `Store.Secrets`. Nothing outside `internal/storage` ever touches the on-disk format.

## Rotation Lifecycle

```mermaid
sequenceDiagram
    participant U  as User / Daemon Tick
    participant CMD as cmd/rotate
    participant ST  as storage.Store
    participant H   as storage.HistoryDB
    participant R   as rotation.RotateSecret
    participant CR  as crypto.GeneratePassword
    participant T   as Target Provider
    participant W   as Webhook

    U->>CMD: rotate KEY
    CMD->>ST:  Load() (decrypt vault)
    CMD->>R:   RotateSecret(store, hdb, KEY, webhookURL)
    R->>H:     GetLastSuccessful(KEY) — if none, record baseline
    R->>CR:    GeneratePassword(32)
    loop for each Target
        R->>T: ApplyTarget(target, newVal)
        T-->>R: error?
    end
    alt all targets succeeded
        R->>ST: sec.Value = newVal; Save() (re-encrypt)
        R->>H:  Record(success, "rotation")
        R->>W:  POST {secret_name, ts, "Rotation successful"}
    else any target failed
        R->>H:  Record(failure, "rotation", errMsg)
        Note right of R: vault is NOT mutated;<br/>old value still active
    end
    R-->>CMD: error or nil
```

Key invariants:

- **The vault is updated only after every target succeeds.** A partial propagation never leaves the on-disk vault out of sync with reality.
- **History is append-only.** Failures get a row too, so there is always a paper trail.
- **The webhook is best-effort.** A non-2xx response is logged but the rotation is not undone.

## Rollback Lifecycle

```mermaid
sequenceDiagram
    participant U   as User
    participant CMD as cmd/incident
    participant ST  as storage.Store
    participant H   as storage.HistoryDB
    participant R   as rotation.RollbackSecret
    participant T   as Target Provider

    U->>CMD: incident rollback --secret-name KEY
    CMD->>ST: Load()
    CMD->>R:  RollbackSecret(store, hdb, KEY)
    R->>H:    GetLastSuccessful(KEY) → previous value
    loop for each Target
        R->>T: ApplyTarget(target, previousValue)
    end
    alt all targets succeeded
        R->>ST: sec.Value = previousValue; Save()
        R->>H:  Record(success, "rollback")
    else any target failed
        R->>H:  Record(failure, "rollback", errMsg)
    end
    R-->>CMD: error or nil
```

`GetLastSuccessful` uses `OFFSET 1` to skip the *current* successful entry and return the one before it — i.e. the value the system was using before the most recent rotation.

## Storage Layer

### Vault (`secrets.json`)

- Marshalled JSON (`Store.Secrets`) → encrypted via `crypto.Encrypt` → written to disk with `0600`.
- On load: file read → `crypto.Decrypt` → `json.Unmarshal` into `map[string]*Secret`. If unmarshal fails, retry as `map[string]string` (legacy v0.1 shape).
- No file-level locking. Concurrent writers will race.

### History (`history.db`)

```sql
CREATE TABLE secret_history (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  secret_name TEXT     NOT NULL,
  value       TEXT     NOT NULL,
  rotated_at  DATETIME NOT NULL,
  status      TEXT     NOT NULL,   -- "success" | "failure"
  error_msg   TEXT,
  operation   TEXT     NOT NULL    -- "rotation" | "rollback" | "baseline"
);
CREATE INDEX idx_secret_name ON secret_history(secret_name);
```

`Record` always inserts; the schema never updates rows in place. This makes the history immutable from the application's perspective.

> ⚠️ The `value` column is plaintext — it has to be, for rollback to work without re-deriving an old key. Treat `history.db` as sensitive (see [SECURITY.md](../SECURITY.md)).

## Crypto Layer

| Step | Detail |
|---|---|
| Key derivation | PBKDF2-SHA256, salt = `gosecretsrotator-salt-2026` (constant), 10 000 iterations, 32-byte output |
| Cipher | AES-256 in GCM mode |
| Nonce | 12 random bytes (`crypto/rand`) per encrypt; prepended to ciphertext |
| Tag | 16-byte GCM auth tag, appended by `Seal` |
| Layout | `nonce \|\| ciphertext \|\| tag` — single binary blob |
| Password gen | `crypto/rand.Int` over a 74-char alphabet |

`Encrypt` and `Decrypt` are pure functions over `(plaintext, password)` / `(ciphertext, password)`; they have no other state and no I/O.

## Provider Contract

A provider is anything that can take `(target, newValue) → error`. The dispatch lives in [`rotation.ApplyTarget`](../internal/rotation/rotation.go):

```go
func ApplyTarget(t storage.Target, value string) error {
    switch t.Type {
    case "docker":
        return docker.UpdateContainerEnv(t.Container, t.EnvKey, value)
    case "file":
        switch ext := strings.ToLower(filepath.Ext(t.Path)); ext {
        case ".env":          return file.InjectEnv(t.Path, t.FileKey, value)
        case ".yaml", ".yml": return file.InjectYAML(t.Path, t.FileKey, value)
        default:              return fmt.Errorf("unsupported file extension: %s", ext)
        }
    default:
        return fmt.Errorf("unknown target type: %q", t.Type)
    }
}
```

To add a new provider:

1. Implement the writer in `internal/providers/<name>` exposing one or more pure functions.
2. Add fields to `storage.Target` (with `omitempty`) for the new target's parameters.
3. Add a `case` to `ApplyTarget`.
4. Add a Cobra subcommand under `cmd/target.go` (and optionally `cmd/inject.go`).
5. Tests + docs.

### Docker Provider Specifics

`docker.UpdateContainerEnv` does:

1. `ContainerInspect` → fetches Config, HostConfig, Networks.
2. **Aborts** if `inspect.Config.Labels["com.docker.compose.project"]` is set — recreating a Compose-managed container would orphan it.
3. Substitutes the env entry (`KEY=newVal`) in `inspect.Config.Env`.
4. `ContainerStop` → `ContainerRename(<name>_old_rotate)` → `ContainerCreate` (with original name + preserved networks) → `ContainerStart` → `ContainerRemove(force=true)` of the old.
5. On `ContainerCreate` failure, renames the old container back to its original name (best-effort).

### File Provider Specifics

- **`.env`** parses each line, finds `KEY=...`, swaps the value, and reuses `splitEnvComment` to keep any trailing inline comment. The comment splitter is quote-aware so `KEY="val#ue"` is left untouched.
- **`.yaml` / `.yml`** unmarshals into a `yaml.Node` tree, recurses into mapping nodes, and updates the first key matching the target name. The marshalled output preserves comments and ordering thanks to `gopkg.in/yaml.v3`'s round-tripping.
- Both writers use `0600` permissions on output.

## Daemon Loop

`cmd/daemon.go` runs three things concurrently:

1. **Signal handler** — `signal.NotifyContext(SIGINT, SIGTERM)` cancels the context.
2. **Metrics HTTP server** — `http.Server` with a 5 s `ReadHeaderTimeout`, mounted at `/metrics`.
3. **Tick loop** — `time.Ticker` with the user-configured interval. On each tick, the loop calls `runOnce`, which:
    - Loads the vault (decrypt).
    - Calls `rotation.DueSecrets` to get the keys whose interval elapsed.
    - For each due key, calls `rotation.RotateSecret`, logging success/failure per secret.

The vault is reloaded on every tick so external edits are picked up without restarting the daemon.

## Observability

| Mechanism | Where | Notes |
|---|---|---|
| Prometheus metrics | `/metrics` on `GOSECRETS_METRICS_PORT` (default `2112`) | Counters and gauges defined in `internal/rotation/metrics.go` via `promauto` |
| Webhook | `GOSECRETS_WEBHOOK_URL` | JSON `{secret_name, timestamp, message}`; no secret values ever sent |
| Logs | stdlib `log` package → stdout/stderr | Structured enough for `journalctl`/Docker log drivers |

The metric names are intentionally lowercase / snake_case (`gosecrets_rotations_total`) to match Prometheus conventions.

## Error Handling Strategy

- **Failures during rotation are persisted to history before being returned.** This keeps the audit trail useful even when callers ignore errors.
- **`hdb.Record` errors are logged, not propagated.** A history-DB write failure should not mask the underlying rotation error.
- **Webhook failures are logged, not propagated.** Networks are unreliable; an alert miss is preferable to a rotation rollback.
- **Provider failures abort the rotation.** The vault is not updated, so the previous value remains live.

## Backwards Compatibility

- **Vault schema.** `Store.Load` first tries the modern `map[string]*Secret` shape and falls back to the v0.1 `map[string]string` shape, lifting each entry into a `Secret{Value: v, LastRotated: now}`. The first `Save` after a load rewrites the file in the new shape.
- **History schema.** `CREATE TABLE IF NOT EXISTS` ensures fresh deployments and existing databases coexist; column additions in the future will need an explicit migration.
- **CLI surface.** Flags are added, never renamed silently. Removals go through one minor-version deprecation cycle.

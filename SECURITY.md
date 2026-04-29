# Security Policy

GoSecretsRotator handles credentials, so the security posture matters even though this is a study project. This document covers the threat model, known limitations, and how to disclose vulnerabilities responsibly.

## Table of Contents

- [Disclosure Process](#disclosure-process)
- [Supported Versions](#supported-versions)
- [Threat Model](#threat-model)
- [What Is Protected](#what-is-protected)
- [Known Limitations](#known-limitations)
- [Hardening Recommendations](#hardening-recommendations)
- [Cryptographic Details](#cryptographic-details)

## Disclosure Process

**Please do not file public GitHub issues for security vulnerabilities.**

Instead:

1. Email the maintainer through the contact link on [the author's portfolio](https://enoquesousa.vercel.app), or open a private security advisory on GitHub: <https://github.com/esousa97/gosecretsrotator/security/advisories/new>.
2. Include:
   - A clear description of the issue and impact.
   - Steps to reproduce (a minimal PoC if possible).
   - Affected version / commit SHA.
   - Any suggested mitigation.
3. Expect an acknowledgement within **7 days**. As a study project there is no formal SLA, but credible reports will be triaged promptly.
4. Please give a reasonable time window (typically 30 days) before public disclosure so a fix can be released.

Researchers acting in good faith will be credited in the release notes if they wish.

## Supported Versions

Only the `master` branch receives fixes. There are no LTS releases.

| Version | Supported |
|---------|-----------|
| `master` (HEAD) | ✅ |
| Tagged releases | ❌ (study project — no backports) |

## Threat Model

### In scope

- Recovery of vault contents from a stolen `secrets.json` file.
- Forgery / tampering of vault contents.
- Privilege escalation through the Docker provider (e.g. container escape, label spoofing).
- Information leakage via webhook payloads or Prometheus metric labels.
- Path traversal / injection through the file provider.

### Out of scope

- Compromise of the host (root access, malicious kernel modules, hostile process scraping `/proc/<pid>/mem`).
- Compromise of `GOSECRETS_MASTER_PWD` itself (e.g. via shell history, env-dump, or key-logging).
- Theft of `history.db` — by design it contains the previous plaintext value to enable rollback. Treat it like the vault.
- Targeted side-channel attacks on the Go runtime or `crypto/aes`.

## What Is Protected

| Asset | Protection |
|---|---|
| `secrets.json` | AES-256-GCM authenticated encryption; key derived from `GOSECRETS_MASTER_PWD` via PBKDF2-SHA256 (10 000 iterations). Tampering is detected on `Decrypt`. |
| File permissions | `secrets.json` and any rewritten `.env` / `.yaml` are written with `0600`. |
| Master password | Read from `GOSECRETS_MASTER_PWD`; never logged, never written to disk by this tool. |
| Webhook payloads | Only `secret_name`, `timestamp`, and a fixed message string are sent — **never the secret value**. |
| Prometheus labels | Use `secret_name` only — never the value. |
| Docker recreation | Network endpoints, env, and `HostConfig` are preserved on recreation; old container is force-removed only after the new one is started. |
| Compose containers | Refused outright (label-based check) so a Compose project never gets orphaned. |

## Known Limitations

These are conscious trade-offs given the study scope. Treat them as TODOs, not as design endorsements.

1. **Fixed PBKDF2 salt.** The salt is a constant string baked into the binary (`internal/crypto/crypto.go`). This means two vaults using the same master password produce the same key — viable for offline rainbow-table attacks against the master password if the vault leaks. *Mitigation*: use a long, high-entropy master password until per-vault salts are added (see roadmap).
2. **PBKDF2 iteration count.** 10 000 iterations is below current OWASP guidance (≥ 600 000 for PBKDF2-SHA256). Sufficient for a study project; do not use in production.
3. **History stores plaintext rotated values.** `history.db` is a normal SQLite file with no encryption layer. Required for rollback, but treat the file with the same care as the vault.
4. **No process-level locking on the vault.** Concurrent writers (two CLI invocations at once) can interleave and lose updates. Run one process at a time.
5. **Master password lives in an env var.** Anything that can read `/proc/<pid>/environ` can read it.
6. **Docker provider needs root-equivalent access.** Talking to the Docker daemon socket effectively grants host-level privileges. Be deliberate about who runs the binary.
7. **No KMS / HSM integration.** The master key never leaves the local machine; there is no envelope encryption.
8. **Webhook delivery is best-effort.** A non-2xx response is logged but does not roll back the rotation — downstream systems must be idempotent.

## Hardening Recommendations

If you do experiment with this beyond a sandbox, at least:

- Pick a master password ≥ 32 random characters; store it in a real secret manager (1Password CLI, `pass`, or systemd `LoadCredential=`).
- Run `gosecretsrotator` under a dedicated, unprivileged user; restrict that user's access to the Docker socket via group membership.
- Place `secrets.json` and `history.db` on a filesystem that is encrypted at rest.
- Set `umask 077` in the service unit so any rewritten files inherit tight permissions even if the binary's `0600` is bypassed by a future bug.
- Front the Prometheus port with an auth proxy or bind it to `127.0.0.1` and scrape via SSH tunnel.
- Restrict the webhook URL to TLS endpoints; verify the receiver's certificate chain.

## Cryptographic Details

| Property | Value |
|---|---|
| Symmetric cipher | AES-256 in **GCM** mode (`crypto/cipher.NewGCM`) |
| Authentication | GCM tag (16 bytes), verified on decrypt — tampering raises `cipher: message authentication failed` |
| Nonce | 12 random bytes from `crypto/rand`, prepended to the ciphertext |
| Key derivation | PBKDF2-SHA256, 10 000 iterations, 32-byte output |
| Salt | Fixed string `gosecretsrotator-salt-2026` *(see [Known Limitations](#known-limitations))* |
| Password generation | `crypto/rand.Int` over a 74-char alphabet (alphanumeric + `!@#$%^&*-_=+`); default length 32 |
| Vault on-disk layout | `nonce \|\| ciphertext \|\| GCM-tag` — a single binary blob |

References:

- NIST SP 800-38D (GCM)
- NIST SP 800-132 (PBKDF2)
- OWASP Password Storage Cheat Sheet
- Go [`crypto/cipher`](https://pkg.go.dev/crypto/cipher) and [`golang.org/x/crypto/pbkdf2`](https://pkg.go.dev/golang.org/x/crypto/pbkdf2)

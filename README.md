# sshw

![GitHub](https://img.shields.io/github/license/vaska94/sshw) ![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/vaska94/sshw)

ssh client wrapper for automatic login.

![usage](./assets/sshw-demo.gif)

## install

use `go get`

```
go install github.com/vaska94/sshw/cmd/sshw@latest
```

or download binary from [releases](//github.com/vaska94/sshw/releases).

## config

config file load in following order:

- `~/.sshw`
- `~/.sshw.yml`
- `~/.sshw.yaml`
- `./.sshw`
- `./.sshw.yml`
- `./.sshw.yaml`

config example:

<!-- prettier-ignore -->
```yaml
- { name: dev server fully configured, user: appuser, host: 192.168.8.35, port: 22, password: 123456 }
- { name: dev server with key path, user: appuser, host: 192.168.8.35, port: 22, keypath: /root/.ssh/id_rsa }
- { name: dev server with passphrase key, user: appuser, host: 192.168.8.35, port: 22, keypath: /root/.ssh/id_rsa, passphrase: abcdefghijklmn}
- { name: dev server without port, user: appuser, host: 192.168.8.35 }
- { name: dev server without user, host: 192.168.8.35 }
- { name: dev server without password, host: 192.168.8.35 }
- { name: ⚡️ server with emoji name, host: 192.168.8.35 }
- { name: server with alias, alias: dev, host: 192.168.8.35 }
- name: server with jump
  user: appuser
  host: 192.168.8.35
  port: 22
  password: 123456
  jump:
  - user: appuser
    host: 192.168.8.36
    port: 2222


# server group 1
- name: server group 1
  children:
  - { name: server 1, user: root, host: 192.168.1.2 }
  - { name: server 2, user: root, host: 192.168.1.3 }
  - { name: server 3, user: root, host: 192.168.1.4 }

# server group 2
- name: server group 2
  children:
  - { name: server 1, user: root, host: 192.168.2.2 }
  - { name: server 2, user: root, host: 192.168.3.3 }
  - { name: server 3, user: root, host: 192.168.4.4 }
```

# callback

<!-- prettier-ignore -->
```yaml
- name: dev server fully configured
  user: appuser
  host: 192.168.8.35
  port: 22
  password: 123456
  callback-shells:
    - { cmd: 2 }
    - { delay: 1500, cmd: 0 }
    - { cmd: "echo 1" }
```

## Managing hosts in the TUI

This fork adds in-TUI host management. While the host picker is open, the
following keys are available (shown in the footer bar):

| Key | Action |
|-----|--------|
| `^A` | Add a new host (opens a form) |
| `^E` | Edit the selected host or folder |
| `^D` | Delete the selected host or folder (with confirmation) |
| `^O` | Open the options hub (master password, share, FileZilla import/export) |
| `^S` | Share — copy a safe summary of the selected host to the clipboard |

Changes are written back to `~/.sshw.yml` automatically (atomic write with a
`.bak` backup left alongside).

### Master-password encryption

Master-password protection is **off by default**. Enable it via `^O` →
"Master Password". When enabled, all `password` fields in `~/.sshw.yml` are
stored as self-describing `enc:` strings (argon2id key-derivation +
XChaCha20-Poly1305 AEAD). You are prompted for the master password once per
session on first decrypt; the derived key is cached in memory for the run.

Disabling the master password decrypts all fields back to plaintext in
`~/.sshw.yml`.

> **Note:** if you use an external tool to sync `sites.json` → `~/.sshw.yml`,
> that sync must be updated to understand the `enc:` format — prompt for the
> master password at sync time; leaving it blank writes plaintext.

### FileZilla import / export

Via `^O` → "FileZilla Import" or "FileZilla Export":

- **Import** reads a `sitemanager.xml` file and merges SFTP entries into
  `~/.sshw.yml` (existing hosts with the same folder-path + name are
  left untouched; new ones are appended). FTP-only entries are skipped.
- **Export** writes the current hosts to a `sitemanager.xml` compatible with
  FileZilla's Site Manager.

### Host-key fingerprint pinning

Add a `fingerprint` field to any host entry to enable verified first-connect:

```yaml
- name: production
  host: 203.0.113.10
  user: deploy
  password: secret
  fingerprint: "SHA256:xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

When `fingerprint` is set, the presented host key is checked against it before
the usual `~/.ssh/known_hosts` trust-on-first-use logic. A mismatch aborts the
connection. Leave the field absent to use plain TOFU (the default).

The fingerprint for a server can be obtained with:

```
ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub
```

or from the AWS EC2 console System Log on first boot.

## Credits

This repository is a vendored fork maintained by [WebDevBar](https://github.com/WebDevBar) for internal use. Full credit to the upstream authors:

- **Original author** — [yinheli](https://github.com/yinheli) · [yinheli/sshw](https://github.com/yinheli/sshw)
- **Fork improvements** — [vaska94](https://github.com/vaska94) · [vaska94/sshw](https://github.com/vaska94/sshw)
  (modernized deprecated APIs, custom terminal select widget, case-insensitive search, `copy-id` support, dependency cleanup)
- **This fork** — [WebDevBar/sshw](https://github.com/WebDevBar/sshw) — tracks `vaska94/sshw` upstream; updates synced and reviewed manually.
  WebDevBar enhancements: host-key verification (trust-on-first-use via `~/.ssh/known_hosts`, with `HostKeyAlgorithms` pinned to the trusted key type), global cross-folder search in the host picker, case-insensitively sorted folders/hosts, in-TUI host management (`^A`/`^E`/`^D`), master-password encryption (`enc:` format, argon2id + XChaCha20-Poly1305), FileZilla import/export, host-key fingerprint pinning, and a gated clipboard share (`^S`). These were developed with [Claude Code](https://claude.com/claude-code) (Anthropic).

Licensed under the [MIT License](./LICENSE) © 2018–2026 yinheli (me@yinheli.com).

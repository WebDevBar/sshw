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

## Credits

This repository is a vendored fork maintained by [WebDevBar](https://github.com/WebDevBar) for internal use. Full credit to the upstream authors:

- **Original author** — [yinheli](https://github.com/yinheli) · [yinheli/sshw](https://github.com/yinheli/sshw)
- **Fork improvements** — [vaska94](https://github.com/vaska94) · [vaska94/sshw](https://github.com/vaska94/sshw)
  (modernized deprecated APIs, custom terminal select widget, case-insensitive search, `copy-id` support, dependency cleanup)
- **This fork** — [WebDevBar/sshw](https://github.com/WebDevBar/sshw) — tracks `vaska94/sshw` upstream; updates synced and reviewed manually.

Licensed under the [MIT License](./LICENSE) © 2018–2026 yinheli (me@yinheli.com).

# TODO / Wishlist (WebDevBar fork)

- [ ] **Non-interactive CLI import / add-host.** Importing FileZilla sites currently
  requires the interactive TUI (`^O` -> Import from FileZilla). Add CLI flags so hosts
  can be imported or added without launching the picker, e.g.:
    - `sshw -import-filezilla <path/to/sitemanager.xml>` - merge SFTP entries into `~/.sshw.yml`
    - `sshw -add <user>@<host>[:port] [-name NAME] [-folder FOLDER]` - add a single host
  Rationale: scripted setup on a fresh machine, and adding a one-off host (e.g. a
  Tailscale desktop) from the command line instead of the TUI.

package sshw

import (
	"fmt"
	"strings"
)

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ShareText renders a host's non-empty fields + a safe ssh command line.
func ShareText(n *Node) string {
	var b strings.Builder
	add := func(label, v string) {
		if v != "" {
			fmt.Fprintf(&b, "%-9s %s\n", label+":", v)
		}
	}
	add("name", n.Name)
	add("host", n.Host)
	add("user", n.User)
	if n.Port != 0 {
		add("port", fmt.Sprintf("%d", n.Port))
	}
	add("alias", n.Alias)
	add("password", n.Password)
	add("passphrase", n.Passphrase)
	add("keypath", n.KeyPath)

	// ssh line: refuse if host/user/keypath could be parsed as an option
	unsafe := strings.HasPrefix(n.Host, "-") || strings.HasPrefix(n.User, "-") || strings.HasPrefix(n.KeyPath, "-")
	if n.Host != "" && !unsafe {
		parts := []string{"ssh"}
		if n.Port != 0 && n.Port != 22 {
			parts = append(parts, "-p", fmt.Sprintf("%d", n.Port))
		}
		if n.KeyPath != "" {
			parts = append(parts, "-i", shellQuote(n.KeyPath))
		}
		dest := shellQuote(n.Host)
		if n.User != "" {
			dest = shellQuote(n.User) + "@" + shellQuote(n.Host)
		}
		parts = append(parts, "--", dest)
		b.WriteString("\n" + strings.Join(parts, " ") + "\n")
	}
	return b.String()
}

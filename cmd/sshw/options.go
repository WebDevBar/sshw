package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/yinheli/sshw"
	"golang.org/x/term"
)

// optItem is one row in the options hub menu.
type optItem struct {
	label string
	help  string
}

var optItems = []optItem{
	{"Add host", "add a new SSH host to the config"},
	{"Master password", "set, change, or clear at-rest encryption"},
	{"Enable Share", "toggle clipboard ^S share (shows credentials)"},
	{"Export to FileZilla", "write sitemanager.xml for FileZilla import"},
	{"Import from FileZilla", "merge sitemanager.xml into this config"},
	{"Back", "return to the host picker"},
}

// runOptions shows the options hub and dispatches to sub-flows.
// Replaces the Task-6 stub.
func runOptions() {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return
	}
	defer term.Restore(fd, oldState)

	fmt.Fprint(os.Stderr, ansiHideCur)
	defer fmt.Fprint(os.Stderr, ansiShowCur)

	cursor := 0
	renderedLines := 0

	render := func() {
		for i := 0; i < renderedLines; i++ {
			fmt.Fprint(os.Stderr, ansiCursorUp+ansiClearLine)
		}
		var buf strings.Builder
		buf.WriteString(ansiClearLine + ansiFaint + "Use arrows to navigate, Enter to select, Esc to go back" + ansiReset + "\r\n")
		buf.WriteString(ansiClearLine + "✨ " + ansiGreen + "Options" + ansiReset + "\r\n")
		lines := 2
		for i, item := range optItems {
			buf.WriteString(ansiClearLine)
			label := item.label
			// Annotate Enable Share with current state
			if item.label == "Enable Share" {
				if shareEnabled {
					label = "Disable Share (currently ON)"
				} else {
					label = "Enable Share (currently OFF)"
				}
			}
			// Annotate Master password with current state
			if item.label == "Master password" {
				if settings != nil && settings.MasterPassword.Enabled {
					label = "Master password (currently ON)"
				} else {
					label = "Master password (currently OFF)"
				}
			}
			if i == cursor {
				buf.WriteString("  " + ansiCyan + "➤ " + label + ansiReset + "\r\n")
			} else {
				buf.WriteString("    " + label + "\r\n")
			}
			lines++
			buf.WriteString(ansiClearLine + "      " + ansiFaint + item.help + ansiReset + "\r\n")
			lines++
		}
		fmt.Fprint(os.Stderr, buf.String())
		renderedLines = lines
	}

	render()

	b := make([]byte, 64)
	for {
		n, err := os.Stdin.Read(b)
		if err != nil {
			break
		}
		input := b[:n]
		switch {
		case n == 1 && input[0] == 3: // ^C
			clearScreen(renderedLines)
			return
		case n == 1 && input[0] == 27: // Esc
			clearScreen(renderedLines)
			return
		case n == 3 && input[0] == 27 && input[1] == 91 && input[2] == 65: // Up
			if cursor > 0 {
				cursor--
			}
		case n == 3 && input[0] == 27 && input[1] == 91 && input[2] == 66: // Down
			if cursor < len(optItems)-1 {
				cursor++
			}
		case n == 1 && input[0] == 13: // Enter
			clearScreen(renderedLines)
			term.Restore(fd, oldState)
			switch optItems[cursor].label {
			case "Add host":
				// Re-enter raw mode for the form, then restore for picker
				term.Restore(fd, oldState)
				return
			case "Master password":
				runMasterPassword()
			case "Enable Share":
				runShareToggle()
			case "Export to FileZilla":
				runExportFileZilla()
			case "Import from FileZilla":
				runImportFileZilla()
			case "Back":
				return
			}
			// Re-enter raw for hub (unless we returned above)
			newState, rerr := term.MakeRaw(fd)
			if rerr != nil {
				return
			}
			oldState = newState
			fmt.Fprint(os.Stderr, ansiHideCur)
			renderedLines = 0
			render()
			continue
		default:
			render()
			continue
		}
		render()
	}
}

// runMasterPassword handles set/change/clear of the master password.
func runMasterPassword() {
	cfg := sshw.GetConfig()
	enabled := settings != nil && settings.MasterPassword.Enabled
	verifier := ""
	if settings != nil {
		verifier = settings.MasterPassword.Verifier
	}

	if !enabled && !sshw.AnyEncrypted(cfg) {
		// --- SET (from off) ---
		fmt.Fprint(os.Stderr, "\r\n  Set master password\r\n")
		pw, ok := readNewPasswordTwice()
		if !ok {
			fmt.Fprintln(os.Stderr, "  Cancelled.")
			return
		}
		salt := sshw.OperativeSalt(cfg, "")
		if err := sshw.EncryptAll(cfg, pw, salt); err != nil {
			fmt.Fprintf(os.Stderr, "  encrypt error: %v\r\n", err)
			return
		}
		sshw.SetConfig(cfg)
		if err := sshw.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "  save error: %v\r\n", err)
			return
		}
		v, err := sshw.MakeVerifier(pw, salt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  verifier error: %v\r\n", err)
			return
		}
		if settings == nil {
			settings = &sshw.Settings{}
		}
		settings.MasterPassword.Enabled = true
		settings.MasterPassword.Verifier = v
		if serr := sshw.SaveSettings(settings); serr != nil {
			fmt.Fprintf(os.Stderr, "  settings save error: %v\r\n", serr)
			return
		}
		cachedMasterPassword = pw
		fmt.Fprintln(os.Stderr, "  Master password set. Config is now encrypted.")
		return
	}

	// Has a master password already — offer change or clear
	fmt.Fprint(os.Stderr, "\r\n  Master password options:\r\n")
	fmt.Fprint(os.Stderr, "  [c] change   [x] clear   [Esc] cancel\r\n")

	fd := int(os.Stdin.Fd())
	oldState, _ := term.MakeRaw(fd)
	buf := make([]byte, 4)
	n, _ := os.Stdin.Read(buf)
	term.Restore(fd, oldState)
	fmt.Fprintln(os.Stderr)

	if n == 0 {
		return
	}
	ch := buf[0]

	switch {
	case ch == 'c' || ch == 'C':
		// --- CHANGE ---
		fmt.Fprint(os.Stderr, "  Enter current master password: ")
		current, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return
		}
		currentPw := string(current)
		if !sshw.ValidatePassword(cfg, verifier, currentPw) {
			fmt.Fprintln(os.Stderr, "  Wrong current master password.")
			return
		}
		pw, ok := readNewPasswordTwice()
		if !ok {
			fmt.Fprintln(os.Stderr, "  Cancelled.")
			return
		}
		salt := sshw.OperativeSalt(cfg, verifier)
		if err := sshw.DecryptAll(cfg, currentPw); err != nil {
			fmt.Fprintf(os.Stderr, "  decrypt error: %v\r\n", err)
			return
		}
		if err := sshw.EncryptAll(cfg, pw, salt); err != nil {
			fmt.Fprintf(os.Stderr, "  encrypt error: %v\r\n", err)
			return
		}
		sshw.SetConfig(cfg)
		if err := sshw.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "  save error: %v\r\n", err)
			return
		}
		v, err := sshw.MakeVerifier(pw, salt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  verifier error: %v\r\n", err)
			return
		}
		settings.MasterPassword.Verifier = v
		if serr := sshw.SaveSettings(settings); serr != nil {
			fmt.Fprintf(os.Stderr, "  settings save error: %v\r\n", serr)
			return
		}
		cachedMasterPassword = pw
		fmt.Fprintln(os.Stderr, "  Master password changed.")

	case ch == 'x' || ch == 'X':
		// --- CLEAR ---
		fmt.Fprint(os.Stderr, "  Enter current master password to confirm clear: ")
		current, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return
		}
		currentPw := string(current)
		if !sshw.ValidatePassword(cfg, verifier, currentPw) {
			fmt.Fprintln(os.Stderr, "  Wrong master password.")
			return
		}
		// Spec §4.1 ordering: drop verifier FIRST, then save plaintext.
		if settings == nil {
			settings = &sshw.Settings{}
		}
		settings.MasterPassword.Enabled = false
		settings.MasterPassword.Verifier = ""
		if serr := sshw.SaveSettings(settings); serr != nil {
			fmt.Fprintf(os.Stderr, "  settings save error: %v\r\n", serr)
			return
		}
		if err := sshw.DecryptAll(cfg, currentPw); err != nil {
			fmt.Fprintf(os.Stderr, "  decrypt error: %v\r\n", err)
			return
		}
		sshw.SetConfig(cfg)
		if err := sshw.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "  save error: %v\r\n", err)
			return
		}
		cachedMasterPassword = ""
		fmt.Fprintln(os.Stderr, "  Master password cleared. Config is now plaintext.")

	default:
		fmt.Fprintln(os.Stderr, "  Cancelled.")
	}
}

// readNewPasswordTwice prompts for a new password twice and returns it, or ("", false) on mismatch/cancel.
func readNewPasswordTwice() (string, bool) {
	fmt.Fprint(os.Stderr, "  New password: ")
	b1, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil || len(b1) == 0 {
		return "", false
	}
	fmt.Fprint(os.Stderr, "  Confirm password: ")
	b2, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", false
	}
	if string(b1) != string(b2) {
		fmt.Fprintln(os.Stderr, "  Passwords do not match.")
		return "", false
	}
	return string(b1), true
}

// runShareToggle flips the share enabled flag.
func runShareToggle() {
	if settings == nil {
		settings = &sshw.Settings{}
	}
	settings.Share.Enabled = !settings.Share.Enabled
	if err := sshw.SaveSettings(settings); err != nil {
		fmt.Fprintf(os.Stderr, "  settings save error: %v\r\n", err)
		return
	}
	shareEnabled = settings.Share.Enabled
	if shareEnabled {
		fmt.Fprintln(os.Stderr, "  Share enabled. ^S will copy host credentials to clipboard.")
	} else {
		fmt.Fprintln(os.Stderr, "  Share disabled.")
	}
}

// deepCopyConfig returns a deep clone of the node tree via yaml round-trip.
// Used by runExportFileZilla to decrypt a copy without touching the live tree.
func deepCopyConfig(nodes []*sshw.Node) ([]*sshw.Node, error) {
	b, err := sshw.MarshalNodes(nodes)
	if err != nil {
		return nil, err
	}
	return sshw.UnmarshalNodes(b)
}

// countLeaves returns the number of host leaf nodes in the tree.
func countLeaves(nodes []*sshw.Node) int {
	count := 0
	for _, n := range nodes {
		if n.Host != "" {
			count++
		}
		count += countLeaves(n.Children)
		count += countLeaves(n.Jump)
	}
	return count
}

// runExportFileZilla prompts for an output path, warns about plaintext and
// confirms before decrypting (never the live tree), then writes sitemanager.xml.
func runExportFileZilla() {
	cfg := sshw.GetConfig()

	// --- 1. Prompt for output path ---
	fmt.Fprint(os.Stderr, "  Output path [sitemanager.xml]: ")
	var outPath string
	fmt.Fscan(os.Stdin, &outPath)
	if outPath == "" {
		outPath = "sitemanager.xml"
	}
	fmt.Fprintln(os.Stderr)

	// --- 2. Plaintext-export warning + confirm (BEFORE any decrypt/master-password prompt) ---
	fmt.Fprintln(os.Stderr, "  \u26a0  WARNING: the exported file will contain passwords in PLAINTEXT.")
	fmt.Fprint(os.Stderr, "  Continue? [y/N]: ")

	b := make([]byte, 4)
	n, _ := os.Stdin.Read(b)
	fmt.Fprintln(os.Stderr)
	if n == 0 || (b[0] != 'y' && b[0] != 'Y') {
		fmt.Fprintln(os.Stderr, "  Export cancelled.")
		return
	}

	needDecrypt := (settings != nil && settings.MasterPassword.Enabled) || sshw.AnyEncrypted(cfg)

	// --- 3. Decrypt a deep copy if needed ---
	var exportNodes []*sshw.Node
	if needDecrypt {
		verifier := ""
		if settings != nil {
			verifier = settings.MasterPassword.Verifier
		}
		enabled := settings != nil && settings.MasterPassword.Enabled
		pw, err := ensureMaster(cfg, verifier, enabled)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  master password error: %v\r\n", err)
			return
		}
		cp, err := deepCopyConfig(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  copy error: %v\r\n", err)
			return
		}
		if err := sshw.DecryptAll(cp, pw); err != nil {
			fmt.Fprintf(os.Stderr, "  decrypt error: %v\r\n", err)
			return
		}
		exportNodes = cp
	} else {
		exportNodes = cfg
	}

	// --- 4. Export and write ---
	xmlData, err := sshw.ExportFileZilla(exportNodes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  export error: %v\r\n", err)
		return
	}
	if err := os.WriteFile(outPath, xmlData, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "  write error: %v\r\n", err)
		return
	}
	count := countLeaves(exportNodes)
	fmt.Fprintf(os.Stderr, "  Exported %d host(s) to %s\r\n", count, outPath)
}

// runImportFileZilla is a placeholder — implemented in Task 15.
func runImportFileZilla() {
	fmt.Fprint(os.Stderr, "  Path to sitemanager.xml [sitemanager.xml]: ")
	var inPath string
	fmt.Fscan(os.Stdin, &inPath)
	if inPath == "" {
		inPath = "sitemanager.xml"
	}
	fmt.Fprintln(os.Stderr)

	b, err := os.ReadFile(inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  read error: %v\r\n", err)
		return
	}

	hosts, err := sshw.ParseFileZilla(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  parse error: %v\r\n", err)
		return
	}

	cfg := sshw.GetConfig()
	res := sshw.MergeImported(&cfg, hosts)

	if settings != nil && settings.MasterPassword.Enabled {
		verifier := settings.MasterPassword.Verifier
		pw, merr := ensureMaster(cfg, verifier, true)
		if merr != nil {
			fmt.Fprintf(os.Stderr, "  master password error: %v\r\n", merr)
			return
		}
		salt := sshw.OperativeSalt(cfg, verifier)
		if eerr := sshw.EncryptAll(cfg, pw, salt); eerr != nil {
			fmt.Fprintf(os.Stderr, "  encrypt error: %v\r\n", eerr)
			return
		}
	}

	sshw.SetConfig(cfg)
	if serr := sshw.Save(); serr != nil {
		fmt.Fprintf(os.Stderr, "  save error: %v\r\n", serr)
		return
	}
	fmt.Fprintf(os.Stderr, "  Import done: %d added, %d updated, %d skipped (FTP).\r\n",
		res.Added, res.Updated, res.Skipped)
}

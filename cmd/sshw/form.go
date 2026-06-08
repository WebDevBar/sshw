package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/yinheli/sshw"
	"golang.org/x/term"
)

// formField is one editable row with its help line and whether it's masked.
type formField struct {
	label  string
	help   string
	masked bool
}

var formFields = []formField{
	{"Name", "label shown in the picker list", false},
	{"Folder", `group path; "/" = sub-levels; blank = top level`, false},
	{"Host", "hostname or IP to connect to", false},
	{"User", "ssh user; blank defaults to root", false},
	{"Port", "ssh port; blank defaults to 22", false},
	{"Alias", "enables `sshw <alias>` direct connect; optional", false},
	{"Key path", "private key for key auth; blank = use password", false},
	{"Passphrase", "unlocks an encrypted key; optional", true},
	{"Password", "password auth; optional, key tried first", true},
	{"Fingerprint", "pin SHA256:… for verified first-connect; blank = TOFU", false},
}

type formModel struct {
	values   map[string]string
	editing  *sshw.Node // non-nil = edit mode (move/update in place)
	origPath string     // folder path the edited node lived in
	cursor   int
}

func newFormModel(src *sshw.Node, path string) *formModel {
	f := &formModel{values: map[string]string{}, origPath: path}
	f.values["Folder"] = path // pre-fill from the picker's current folder (add) or the node's folder (edit)
	if src != nil {
		f.editing = src
		f.values["Name"] = src.Name
		f.values["Host"] = src.Host
		f.values["User"] = src.User
		if src.Port != 0 {
			f.values["Port"] = strconv.Itoa(src.Port)
		}
		f.values["Alias"] = src.Alias
		f.values["Key path"] = src.KeyPath
		f.values["Passphrase"] = src.Passphrase
		f.values["Password"] = src.Password
		f.values["Fingerprint"] = src.Fingerprint
	}
	return f
}

func (f *formModel) set(label, v string)     { f.values[label] = v }
func (f *formModel) get(label string) string { return f.values[label] }

// toNode builds a Node from the form values + returns the target folder path.
func (f *formModel) toNode() (*sshw.Node, string, error) {
	name := f.values["Name"]
	if name == "" {
		return nil, "", fmt.Errorf("Name is required")
	}
	port := 0
	if p := f.values["Port"]; p != "" {
		var err error
		port, err = strconv.Atoi(p)
		if err != nil {
			return nil, "", fmt.Errorf("Port must be a number")
		}
	}
	n := &sshw.Node{}
	if f.editing != nil {
		cp := *f.editing // edit mode: keep advanced fields (AgentPath, CallbackShells, Children, Jump)
		n = &cp
	}
	n.Name = name
	n.Host = f.values["Host"]
	n.User = f.values["User"]
	n.Port = port
	n.Alias = f.values["Alias"]
	n.KeyPath = f.values["Key path"]
	n.Passphrase = f.values["Passphrase"]
	n.Password = f.values["Password"]
	n.Fingerprint = f.values["Fingerprint"]
	return n, f.values["Folder"], nil
}

// runForm renders an add/edit form in raw-terminal mode.
// Returns true when the user saves (Enter on valid input), false on Esc.
func runForm(f *formModel) bool {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return false
	}
	defer term.Restore(fd, oldState)

	fmt.Fprint(os.Stderr, ansiHideCur)
	defer fmt.Fprint(os.Stderr, ansiShowCur)

	title := "Add host"
	if f.editing != nil {
		title = "Edit host"
	}

	errMsg := ""
	renderedLines := 0

	render := func() {
		for i := 0; i < renderedLines; i++ {
			fmt.Fprint(os.Stderr, ansiCursorUp+ansiClearLine)
		}
		var buf strings.Builder
		lines := 0

		// Header
		buf.WriteString(ansiClearLine + ansiFaint + "Tab/↑/↓ move  Enter save  Esc cancel" + ansiReset + "\r\n")
		buf.WriteString(ansiClearLine + "✨ " + ansiGreen + title + ansiReset + "\r\n")
		lines += 2

		// Error line
		if errMsg != "" {
			buf.WriteString(ansiClearLine + ansiYellow + "  ✖ " + errMsg + ansiReset + "\r\n")
		} else {
			buf.WriteString(ansiClearLine + "\r\n")
		}
		lines++

		// Fields
		for i, ff := range formFields {
			val := f.values[ff.label]
			display := val
			if ff.masked && val != "" {
				display = strings.Repeat("•", len([]rune(val)))
			}

			buf.WriteString(ansiClearLine)
			if i == f.cursor {
				buf.WriteString("  " + ansiCyan + ff.label + ": " + ansiReset + display + " " + ansiCyan + "▌" + ansiReset)
			} else {
				buf.WriteString("  " + ansiFaint + ff.label + ": " + ansiReset + display)
			}
			buf.WriteString("\r\n")
			lines++

			// Help line
			buf.WriteString(ansiClearLine + "    " + ansiFaint + ff.help + ansiReset + "\r\n")
			lines++
		}

		fmt.Fprint(os.Stderr, buf.String())
		renderedLines = lines
	}

	render()

	buf := make([]byte, 64)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return false
		}
		b := buf[:n]

		switch {
		case n == 1 && b[0] == 27: // Esc
			clearScreen(renderedLines)
			return false

		case n == 1 && b[0] == 13: // Enter — validate + save
			_, _, verr := f.toNode()
			if verr != nil {
				errMsg = verr.Error()
				render()
				continue
			}
			clearScreen(renderedLines)
			return true

		case n == 1 && b[0] == 9: // Tab — move down
			f.cursor = (f.cursor + 1) % len(formFields)
			errMsg = ""

		case n == 3 && b[0] == 27 && b[1] == 91 && b[2] == 65: // Up arrow
			if f.cursor > 0 {
				f.cursor--
			}
			errMsg = ""

		case n == 3 && b[0] == 27 && b[1] == 91 && b[2] == 66: // Down arrow
			if f.cursor < len(formFields)-1 {
				f.cursor++
			}
			errMsg = ""

		case n == 1 && b[0] == 127: // Backspace
			label := formFields[f.cursor].label
			v := f.values[label]
			if len(v) > 0 {
				// trim last rune (safe for multibyte)
				r := []rune(v)
				f.values[label] = string(r[:len(r)-1])
			}
			errMsg = ""

		case n == 1 && b[0] >= 32 && b[0] < 127: // Printable ASCII
			label := formFields[f.cursor].label
			f.values[label] += string(b[0])
			errMsg = ""

		default:
			continue
		}

		render()
	}
}

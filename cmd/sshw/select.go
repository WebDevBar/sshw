package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/yinheli/sshw"
	"golang.org/x/term"
)

// ANSI escape sequences
const (
	ansiReset     = "\033[0m"
	ansiCyan      = "\033[1;36m"
	ansiYellow    = "\033[1;33m"
	ansiGreen     = "\033[1;32m"
	ansiFaint     = "\033[2m"
	ansiClearLine = "\033[2K"
	ansiCursorUp  = "\033[A"
	ansiHideCur   = "\033[?25l"
	ansiShowCur   = "\033[?25h"
)

// matchText reports whether content matches the query (case-insensitive;
// space-separated terms are ANDed; empty query matches everything).
func matchText(input, content string) bool {
	input = strings.ToLower(input)
	content = strings.ToLower(content)
	if strings.Contains(input, " ") {
		for _, key := range strings.Split(input, " ") {
			key = strings.TrimSpace(key)
			if key != "" && !strings.Contains(content, key) {
				return false
			}
		}
		return true
	}
	return strings.Contains(content, input)
}

func matchNode(input string, node *sshw.Node) bool {
	return matchText(input, node.Name+" "+node.User+" "+node.Host)
}

func formatActive(e entry) string {
	n := e.node
	s := "  ➤ " + ansiCyan + n.Name + ansiReset
	if n.Alias != "" {
		s += "(" + ansiYellow + n.Alias + ansiReset + ")"
	}
	if n.Host != "" {
		s += " "
		if n.User != "" {
			s += ansiFaint + n.User + "@" + ansiReset
		}
		s += ansiFaint + n.Host + ansiReset
	}
	if e.path != "" {
		s += ansiFaint + "  ‹ " + e.path + ansiReset
	}
	return s
}

func formatInactive(e entry) string {
	n := e.node
	s := "    " + n.Name
	if n.Alias != "" {
		s += "(" + n.Alias + ")"
	}
	if n.Host != "" {
		s += " "
		if n.User != "" {
			s += ansiFaint + n.User + "@" + ansiReset
		}
		s += n.Host
	}
	if e.path != "" {
		s += ansiFaint + "  ‹ " + e.path + ansiReset
	}
	return s
}

func selectNode(label string, items []*sshw.Node, leaves []leaf, size int) (*sshw.Node, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	defer term.Restore(fd, oldState)

	fmt.Fprint(os.Stderr, ansiHideCur)
	defer fmt.Fprint(os.Stderr, ansiShowCur)

	cursor := 0
	search := ""
	entries := viewEntries(search, items, leaves)
	renderedLines := 0

	render := func() {
		for i := 0; i < renderedLines; i++ {
			fmt.Fprint(os.Stderr, ansiCursorUp+ansiClearLine)
		}
		var buf strings.Builder
		buf.WriteString(ansiClearLine + ansiFaint + "Use the arrow keys to navigate: ↓ ↑" + ansiReset + "\r\n")
		buf.WriteString(ansiClearLine + "✨ " + ansiGreen + label + ansiReset + "\r\n")
		lines := 2
		if search != "" {
			buf.WriteString(ansiClearLine + ansiFaint + "search: " + ansiReset + search + "\r\n")
			lines++
			if len(entries) == 0 {
				buf.WriteString(ansiClearLine + ansiFaint + "  no matches" + ansiReset + "\r\n")
				lines++
			}
		}
		count := len(entries)
		visible := size
		if count < visible {
			visible = count
		}
		start := 0
		if cursor >= visible {
			start = cursor - visible + 1
		}
		for i := 0; i < size; i++ {
			idx := start + i
			buf.WriteString(ansiClearLine)
			if idx < count {
				if idx == cursor {
					buf.WriteString(formatActive(entries[idx]))
				} else {
					buf.WriteString(formatInactive(entries[idx]))
				}
			}
			buf.WriteString("\r\n")
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
			return nil, err
		}
		b := buf[:n]
		switch {
		case n == 1 && b[0] == 3: // Ctrl+C
			clearScreen(renderedLines)
			return nil, nil
		case n == 1 && b[0] == 13: // Enter
			if len(entries) == 0 {
				continue
			}
			clearScreen(renderedLines)
			return entries[cursor].node, nil
		case n == 1 && b[0] == 127: // Backspace
			if len(search) > 0 {
				search = search[:len(search)-1]
				entries = viewEntries(search, items, leaves)
				cursor = 0
			} else if len(items) > 0 && items[0].Name == prev {
				// empty query while inside a folder -> go up one level
				clearScreen(renderedLines)
				return items[0], nil
			}
		case n == 1 && b[0] == 27: // Escape
			clearScreen(renderedLines)
			if len(items) > 0 && items[0].Name == prev {
				return items[0], nil // inside a folder -> up one level
			}
			return nil, nil // at root -> quit
		case n == 3 && b[0] == 27 && b[1] == 91 && b[2] == 65: // Up
			if cursor > 0 {
				cursor--
			}
		case n == 3 && b[0] == 27 && b[1] == 91 && b[2] == 66: // Down
			if cursor < len(entries)-1 {
				cursor++
			}
		case n == 1 && b[0] >= 32 && b[0] < 127: // Printable
			search += string(b[0])
			entries = viewEntries(search, items, leaves)
			cursor = 0
		default:
			continue
		}
		render()
	}
}

func clearScreen(lines int) {
	for i := 0; i < lines; i++ {
		fmt.Fprint(os.Stderr, ansiCursorUp+ansiClearLine)
	}
}

// leaf is a connectable host plus its "/"-joined folder path.
type leaf struct {
	node *sshw.Node
	path string
}

// flattenLeaves walks the tree and returns every connectable leaf host with its
// breadcrumb path. A node is a searchable leaf iff it has no children, has a
// Host, and is not the synthetic "-parent-" sentinel.
func flattenLeaves(nodes []*sshw.Node, prefix string) []leaf {
	var out []leaf
	for _, n := range nodes {
		if len(n.Children) > 0 {
			child := n.Name
			if prefix != "" {
				child = prefix + "/" + n.Name
			}
			out = append(out, flattenLeaves(n.Children, child)...)
			continue
		}
		if n.Host == "" || n.Name == prev {
			continue
		}
		out = append(out, leaf{node: n, path: prefix})
	}
	return out
}

// leafContent is the text searched for a leaf: path + name + user + host.
func leafContent(l leaf) string {
	return l.path + " " + l.node.Name + " " + l.node.User + " " + l.node.Host
}

// entry is one displayable row: a node, plus an optional breadcrumb path
// (non-empty only for global search results).
type entry struct {
	node *sshw.Node
	path string
}

// viewEntries returns the rows to display: the current level when search is
// empty, otherwise the global leaves filtered by the query (matched on
// path+name+user+host).
func viewEntries(search string, items []*sshw.Node, leaves []leaf) []entry {
	if search == "" {
		es := make([]entry, len(items))
		for i, n := range items {
			es[i] = entry{node: n}
		}
		return es
	}
	var es []entry
	for _, l := range leaves {
		if matchText(search, leafContent(l)) {
			es = append(es, entry{node: l.node, path: l.path})
		}
	}
	return es
}

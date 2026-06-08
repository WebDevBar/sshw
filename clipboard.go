package sshw

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"runtime"
)

// clipboardCmd picks a platform clipboard command; have(name) reports availability.
func clipboardCmd(goos string, have func(string) bool) []string {
	switch goos {
	case "darwin":
		if have("pbcopy") {
			return []string{"pbcopy"}
		}
	case "windows":
		if have("clip") {
			return []string{"clip"}
		}
	default: // linux, bsd, …
		if have("wl-copy") {
			return []string{"wl-copy"}
		}
		if have("xclip") {
			return []string{"xclip", "-selection", "clipboard"}
		}
		if have("xsel") {
			return []string{"xsel", "-b"}
		}
		if have("clip.exe") { // WSL
			return []string{"clip.exe"}
		}
	}
	return nil
}

func osc52(s string) string {
	return fmt.Sprintf("\x1b]52;c;%s\x07", base64.StdEncoding.EncodeToString([]byte(s)))
}

// Copy puts s on the clipboard. Returns (true, "") on a tool success; (false,
// "osc52") when it fell back to the best-effort terminal escape; error only when
// neither path is possible.
func Copy(s string, emit func(string)) (toolOK bool, method string, err error) {
	have := func(name string) bool { _, e := exec.LookPath(name); return e == nil }
	if cmd := clipboardCmd(runtime.GOOS, have); cmd != nil {
		c := exec.Command(cmd[0], cmd[1:]...)
		stdin, _ := c.StdinPipe()
		if e := c.Start(); e == nil {
			_, _ = stdin.Write([]byte(s))
			stdin.Close()
			if c.Wait() == nil {
				return true, cmd[0], nil
			}
		}
	}
	emit(osc52(s)) // best-effort; no ack
	return false, "osc52", nil
}

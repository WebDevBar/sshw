package sshw

import (
	"strings"
	"testing"
)

func TestExportFileZilla(t *testing.T) {
	nodes := []*Node{
		{Name: "grp", Children: []*Node{
			{Name: "db", Host: "10.0.0.5", User: "root", Port: 22, Password: "pw"},
		}},
	}
	xml, err := ExportFileZilla(nodes)
	if err != nil {
		t.Fatal(err)
	}
	s := string(xml)
	for _, want := range []string{"<Name>db</Name>", "<Host>10.0.0.5</Host>", "<Port>22</Port>",
		"<User>root</User>", `<Folder`, "<Protocol>1</Protocol>"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in:\n%s", want, s)
		}
	}
	// password is base64 of "pw"
	if !strings.Contains(s, `encoding="base64"`) || !strings.Contains(s, "cHc=") {
		t.Fatalf("password not base64-encoded:\n%s", s)
	}
}

package main

import (
	"testing"

	"github.com/yinheli/sshw"
)

func TestFormModelToNode(t *testing.T) {
	f := newFormModel(nil, "") // add mode
	f.set("Name", "db")
	f.set("Folder", "prod/db")
	f.set("Host", "10.0.0.5")
	f.set("Port", "2222")
	f.set("Password", "pw")
	n, folder, err := f.toNode()
	if err != nil {
		t.Fatal(err)
	}
	if n.Name != "db" || n.Host != "10.0.0.5" || n.Port != 2222 || n.Password != "pw" {
		t.Fatalf("bad node: %+v", n)
	}
	if folder != "prod/db" {
		t.Fatalf("bad folder: %q", folder)
	}
}

func TestFormValidation(t *testing.T) {
	f := newFormModel(nil, "")
	f.set("Host", "h") // no name
	if _, _, err := f.toNode(); err == nil {
		t.Fatal("expected name-required error")
	}
	f.set("Name", "x")
	f.set("Port", "notaport")
	if _, _, err := f.toNode(); err == nil {
		t.Fatal("expected port-numeric error")
	}
}

func TestFormFromNodePrefills(t *testing.T) {
	src := &sshw.Node{Name: "db", Host: "h", Port: 22, KeyPath: "~/k"}
	f := newFormModel(src, "prod")
	if f.get("Name") != "db" || f.get("Key path") != "~/k" || f.get("Folder") != "prod" {
		t.Fatalf("prefill failed: %q %q", f.get("Name"), f.get("Key path"))
	}
}

func TestFormEditPreservesAdvancedFields(t *testing.T) {
	src := &sshw.Node{Name: "db", Host: "h", AgentPath: "/a",
		Jump: []*sshw.Node{{Name: "j", Host: "jh"}}, CallbackShells: []*sshw.CallbackShell{{Cmd: "x"}}}
	f := newFormModel(src, "prod")
	f.set("Host", "h2") // edit a mapped field
	n, _, err := f.toNode()
	if err != nil {
		t.Fatal(err)
	}
	if n.Host != "h2" || n.AgentPath != "/a" || len(n.Jump) != 1 || len(n.CallbackShells) != 1 {
		t.Fatalf("advanced fields not preserved on edit: %+v", n)
	}
}

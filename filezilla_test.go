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

func TestParseFileZilla(t *testing.T) {
	// Build a FileZilla XML using the exporter for the SFTP portion,
	// then inject a raw FTP server to verify FTP flagging.
	nodes := []*Node{
		{Name: "grp", Children: []*Node{
			{Name: "db", Host: "10.0.0.5", User: "root", Port: 22, Password: "pw"},
		}},
	}
	xmlBytes, err := ExportFileZilla(nodes)
	if err != nil {
		t.Fatal(err)
	}
	// Inject an FTP entry (Protocol=0) into the XML.
	ftpEntry := `  <Server>
    <Name>ftp-server</Name>
    <Host>ftp.example.com</Host>
    <Port>21</Port>
    <Protocol>0</Protocol>
    <Logontype>1</Logontype>
    <User>ftpuser</User>
  </Server>`
	xmlStr := strings.Replace(string(xmlBytes), "</Servers>", ftpEntry+"\n</Servers>", 1)

	hosts, err := ParseFileZilla([]byte(xmlStr))
	if err != nil {
		t.Fatalf("ParseFileZilla error: %v", err)
	}

	// Find the round-tripped SFTP host
	var dbHost *importedHost
	var ftpHost *importedHost
	for i := range hosts {
		h := &hosts[i]
		if h.Name == "db" {
			dbHost = h
		}
		if h.Name == "ftp-server" {
			ftpHost = h
		}
	}

	if dbHost == nil {
		t.Fatal("db host not found in parsed output")
	}
	if dbHost.Host != "10.0.0.5" {
		t.Errorf("host mismatch: got %q want %q", dbHost.Host, "10.0.0.5")
	}
	if dbHost.Port != 22 {
		t.Errorf("port mismatch: got %d want 22", dbHost.Port)
	}
	if dbHost.User != "root" {
		t.Errorf("user mismatch: got %q want %q", dbHost.User, "root")
	}
	if dbHost.Password != "pw" {
		t.Errorf("password mismatch: got %q want %q", dbHost.Password, "pw")
	}
	if dbHost.Path != "grp" {
		t.Errorf("path mismatch: got %q want %q", dbHost.Path, "grp")
	}
	if dbHost.FTP {
		t.Error("SFTP host should not be flagged as FTP")
	}

	if ftpHost == nil {
		t.Fatal("ftp-server host not found in parsed output")
	}
	if !ftpHost.FTP {
		t.Error("FTP host (Protocol=0) should be flagged as FTP")
	}
}

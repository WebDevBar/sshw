package sshw

import (
	"encoding/base64"
	"encoding/xml"
	"strings"
)

// fzPass is a <Pass encoding="base64"> element.
type fzPass struct {
	Encoding string `xml:"encoding,attr"`
	Value    string `xml:",chardata"`
}

// fzServer maps to a <Server> element in the FileZilla sitemanager.xml.
type fzServer struct {
	XMLName   xml.Name `xml:"Server"`
	Name      string   `xml:"Name"`
	Host      string   `xml:"Host"`
	Port      int      `xml:"Port"`
	Protocol  int      `xml:"Protocol"`  // 1 = SFTP
	Logontype int      `xml:"Logontype"` // 0=anonymous, 1=normal, 5=key-file
	User      string   `xml:"User"`
	Pass      *fzPass  `xml:"Pass,omitempty"`
	Keyfile   string   `xml:"Keyfile,omitempty"`
}

// fzFolder maps to a <Folder> element.
type fzFolder struct {
	XMLName xml.Name   `xml:"Folder"`
	Name    string     `xml:",chardata"`
	Servers []fzServer `xml:"Server"`
	Folders []fzFolder `xml:"Folder"`
}

// fzServers is the <Servers> container.
type fzServers struct {
	XMLName xml.Name   `xml:"Servers"`
	Servers []fzServer `xml:"Server"`
	Folders []fzFolder `xml:"Folder"`
}

// fzRoot is the top-level <FileZilla3> element.
type fzRoot struct {
	XMLName xml.Name  `xml:"FileZilla3"`
	Servers fzServers `xml:"Servers"`
}

// ExportFileZilla converts the node tree into a FileZilla sitemanager.xml byte
// slice. Only fields that FileZilla understands are emitted; sshw-only fields
// (Alias, Passphrase, Fingerprint, Jump, CallbackShells) are silently dropped.
func ExportFileZilla(nodes []*Node) ([]byte, error) {
	root := fzRoot{}
	root.Servers.Servers, root.Servers.Folders = walkNodes(nodes)
	out, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}

// walkNodes converts a slice of Nodes into parallel slices of fzServer and
// fzFolder values (leaves and groups respectively).
func walkNodes(nodes []*Node) ([]fzServer, []fzFolder) {
	var servers []fzServer
	var folders []fzFolder
	for _, n := range nodes {
		if len(n.Children) > 0 {
			// Group node → Folder
			f := fzFolder{Name: n.Name}
			f.Servers, f.Folders = walkNodes(n.Children)
			folders = append(folders, f)
		} else if n.Host != "" {
			servers = append(servers, nodeToServer(n))
		}
	}
	return servers, folders
}

// logontype returns the FileZilla Logontype integer for a node:
//
//	5 = key-file (KeyPath set)
//	1 = normal password
//	0 = anonymous / no credentials
func logontype(n *Node) int {
	if n.KeyPath != "" {
		return 5
	}
	if n.Password != "" {
		return 1
	}
	return 0
}

// nodeToServer converts a leaf Node to an fzServer.
func nodeToServer(n *Node) fzServer {
	port := n.Port
	if port <= 0 {
		port = 22
	}
	s := fzServer{
		Name:      n.Name,
		Host:      n.Host,
		Port:      port,
		Protocol:  1, // SFTP
		Logontype: logontype(n),
		User:      n.User,
		Keyfile:   n.KeyPath,
	}
	if n.Password != "" {
		enc := base64.StdEncoding.EncodeToString([]byte(n.Password))
		s.Pass = &fzPass{Encoding: "base64", Value: enc}
	}
	return s
}

// ParseFileZilla reads a FileZilla sitemanager.xml and returns a slice of
// importedHost values. Folders are walked recursively to build folder Path
// strings (slash-joined). Passwords are base64-decoded. Entries with
// Protocol=0 (plain FTP) are included but flagged with FTP=true so that
// callers (e.g. MergeImported) can skip them.
func ParseFileZilla(data []byte) ([]importedHost, error) {
	var root fzRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	var hosts []importedHost
	collectServers(root.Servers.Servers, "", &hosts)
	collectFolders(root.Servers.Folders, "", &hosts)
	return hosts, nil
}

func collectServers(servers []fzServer, path string, out *[]importedHost) {
	for _, s := range servers {
		h := importedHost{
			Path: path,
			Name: s.Name,
			Host: s.Host,
			Port: s.Port,
			User: s.User,
			FTP:  s.Protocol == 0,
		}
		if s.Pass != nil && s.Pass.Encoding == "base64" && s.Pass.Value != "" {
			if decoded, err := base64.StdEncoding.DecodeString(s.Pass.Value); err == nil {
				h.Password = string(decoded)
			}
		}
		if s.Keyfile != "" {
			h.KeyPath = s.Keyfile
		}
		*out = append(*out, h)
	}
}

func collectFolders(folders []fzFolder, parentPath string, out *[]importedHost) {
	for _, f := range folders {
		name := strings.TrimSpace(f.Name)
		var path string
		if parentPath == "" {
			path = name
		} else {
			path = parentPath + "/" + name
		}
		collectServers(f.Servers, path, out)
		collectFolders(f.Folders, path, out)
	}
}

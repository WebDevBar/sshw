package sshw

type importedHost struct {
	Path     string // folder path, "/"-joined
	Name     string
	Host     string
	User     string
	Port     int
	Password string
	KeyPath  string
	FTP      bool // skip on import
}

type MergeResult struct{ Added, Updated, Skipped int }

func MergeImported(root *[]*Node, hosts []importedHost) MergeResult {
	var r MergeResult
	for _, h := range hosts {
		if h.FTP {
			r.Skipped++
			continue
		}
		if existing := FindLeaf(*root, h.Path, h.Name); existing != nil {
			// update ONLY mapped fields; never blank, never touch sshw-only fields
			if h.Host != "" {
				existing.Host = h.Host
			}
			if h.User != "" {
				existing.User = h.User
			}
			if h.Port != 0 {
				existing.Port = h.Port
			}
			if h.Password != "" {
				existing.Password = h.Password
			}
			if h.KeyPath != "" {
				existing.KeyPath = h.KeyPath
			}
			r.Updated++
		} else {
			*root = InsertNode(*root, h.Path, &Node{
				Name: h.Name, Host: h.Host, User: h.User, Port: h.Port,
				Password: h.Password, KeyPath: h.KeyPath,
			})
			r.Added++
		}
	}
	return r
}

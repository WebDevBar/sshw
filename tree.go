package sshw

import "strings"

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

// ensureFolder returns the []*Node slice at a folder path, creating folders as
// needed, via a setter so callers can write the (possibly new) root back.
func ensureFolder(nodes []*Node, parts []string) ([]*Node, func([]*Node) []*Node) {
	if len(parts) == 0 {
		return nodes, func(n []*Node) []*Node { return n }
	}
	head := parts[0]
	for _, n := range nodes {
		if n.Name == head && n.Host == "" {
			inner, setInner := ensureFolder(n.Children, parts[1:])
			return inner, func(x []*Node) []*Node {
				n.Children = setInner(x)
				return nodes
			}
		}
	}
	// create the folder node
	folder := &Node{Name: head}
	nodes = append(nodes, folder)
	inner, setInner := ensureFolder(folder.Children, parts[1:])
	return inner, func(x []*Node) []*Node {
		folder.Children = setInner(x)
		return nodes
	}
}

// InsertNode adds leaf at folder path (creating folders), returns new root.
func InsertNode(root []*Node, path string, leaf *Node) []*Node {
	children, set := ensureFolder(root, splitPath(path))
	children = append(children, leaf)
	root = set(children)
	sortNodes(root)
	return root
}

// FindFolder returns the children slice at path, or nil if absent.
func FindFolder(root []*Node, path string) []*Node {
	cur := root
	for _, part := range splitPath(path) {
		var next []*Node
		found := false
		for _, n := range cur {
			if n.Name == part && n.Host == "" {
				next = n.Children
				found = true
				break
			}
		}
		if !found {
			return nil
		}
		cur = next
	}
	return cur
}

// FindLeaf returns the host node named name directly under folder path.
func FindLeaf(root []*Node, path, name string) *Node {
	for _, n := range FindFolder(root, path) {
		if n.Name == name && n.Host != "" {
			return n
		}
	}
	return nil
}

// DeleteNode removes the node named name directly under folder path and returns
// the (possibly new) root. Works at the root level too (returns a fresh slice).
func DeleteNode(root []*Node, path, name string) ([]*Node, bool) {
	return deleteRec(root, splitPath(path), name)
}

func deleteRec(nodes []*Node, parts []string, name string) ([]*Node, bool) {
	if len(parts) == 0 {
		out := make([]*Node, 0, len(nodes))
		removed := false
		for _, n := range nodes {
			if n.Name == name {
				removed = true
				continue
			}
			out = append(out, n)
		}
		return out, removed
	}
	for _, n := range nodes {
		if n.Name == parts[0] && n.Host == "" {
			newChildren, removed := deleteRec(n.Children, parts[1:], name)
			n.Children = newChildren
			return nodes, removed
		}
	}
	return nodes, false
}

// MoveNode moves a leaf from one folder path to another.
func MoveNode(root []*Node, fromPath, name, toPath string) []*Node {
	leaf := FindLeaf(root, fromPath, name)
	if leaf == nil {
		return root
	}
	cp := *leaf
	root, _ = DeleteNode(root, fromPath, name)
	return InsertNode(root, toPath, &cp)
}

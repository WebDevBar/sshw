package sshw

import "testing"

func sample() []*Node {
	return []*Node{
		{Name: "prod", Children: []*Node{{Name: "db", Host: "1.1.1.1"}}},
		{Name: "h0", Host: "0.0.0.0"},
	}
}

func TestInsertCreatesFolders(t *testing.T) {
	root := sample()
	// "prod" is a folder (db is a host leaf inside it); inserting into "prod" → 2 leaves
	root = InsertNode(root, "prod", &Node{Name: "db2", Host: "2.2.2.2"})
	prod := FindFolder(root, "prod")
	if prod == nil || len(prod) != 2 {
		t.Fatalf("expected 2 leaves under prod, got %v", prod)
	}
	root = InsertNode(root, "new/grp", &Node{Name: "x", Host: "9"})
	if FindFolder(root, "new/grp") == nil {
		t.Fatal("nested folders not created")
	}
}

func TestDeleteByPathName(t *testing.T) {
	root := sample()
	root, ok := DeleteNode(root, "prod", "db")
	if !ok || FindLeaf(root, "prod", "db") != nil {
		t.Fatal("delete failed")
	}
}

func TestMoveNode(t *testing.T) {
	root := sample()
	root = MoveNode(root, "prod", "db", "archive")
	if FindLeaf(root, "prod", "db") != nil || FindLeaf(root, "archive", "db") == nil {
		t.Fatal("move failed")
	}
}

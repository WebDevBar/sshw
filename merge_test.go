package sshw

import "testing"

func TestMergeAddUpdateNeverDelete(t *testing.T) {
	existing := []*Node{
		{Name: "grp", Children: []*Node{
			{Name: "db", Host: "old", User: "root", Alias: "keepme", Fingerprint: "SHA256:keep"},
		}},
		{Name: "lonely", Host: "x"}, // absent from import -> must survive
	}
	imported := []importedHost{
		{Path: "grp", Name: "db", Host: "new", User: "admin", Port: 22}, // update (mapped fields only)
		{Path: "grp", Name: "extra", Host: "e", User: "u", Port: 22},    // add
	}
	res := MergeImported(&existing, imported)
	db := FindLeaf(existing, "grp", "db")
	if db.Host != "new" || db.User != "admin" {
		t.Fatalf("mapped fields not updated: %+v", db)
	}
	if db.Alias != "keepme" || db.Fingerprint != "SHA256:keep" {
		t.Fatalf("sshw-only fields wiped: %+v", db)
	}
	if FindLeaf(existing, "grp", "extra") == nil {
		t.Fatal("missing entry not added")
	}
	if FindLeaf(existing, "", "lonely") == nil {
		t.Fatal("absent entry wrongly deleted")
	}
	if res.Added != 1 || res.Updated != 1 {
		t.Fatalf("bad counts: %+v", res)
	}
}

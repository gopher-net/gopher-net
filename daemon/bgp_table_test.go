package daemon

import (
	"net"
	"testing"
)

var db *BgpDatabase

func TestAddBgpTableEntry(t *testing.T) {
	db = &BgpDatabase{
		Table: make(map[string]BgpDbTable),
	}

	ip := net.ParseIP("192.168.224.1")
	prefixLen := 24

	entry := &BgpDbEntry{
		Prefix:         ip,
		PrefixLen:      prefixLen,
		PathAttributes: nil,
	}

	db.AddEntry(net.ParseIP("10.10.10.10"), entry)

	if len(db.Table) != 1 {
		t.Error("Table should have 1 entry")
	}

	if len(db.Table["10.10.10.10"]) != 1 {
		t.Error("Table for peer 10.10.10.10 should have 1 entry")
	}

	b := [16]byte{95, 102, 88, 236, 64, 214, 160, 46, 71, 136, 150, 245, 134, 214, 9, 149}

	dbEntry, ok := db.Table["10.10.10.10"][b]

	if !ok {
		t.Error("Row with correct hash not found")
	}

	if !dbEntry.Prefix.Equal(ip) {
		t.Error("Prefix address is incorrect")
	}

	if dbEntry.PrefixLen != prefixLen {
		t.Error("Prefix length is incorrect")
	}
}

func TestRemoveBgpTableEntry(t *testing.T) {
	entry := &BgpDbEntry{
		Prefix:         net.ParseIP("192.168.224.1"),
		PrefixLen:      24,
		PathAttributes: nil,
	}
	db.RemoveEntry(net.ParseIP("10.10.10.10"), entry)

	if len(db.Table) != 1 {
		t.Error("Table should have 1 entry")
	}

	if len(db.Table["10.10.10.10"]) != 0 {
		t.Error("Table for peer 10.10.10.10 should have 0 entries")
	}

}

func TestBgpTableHashing(t *testing.T) {
	// Baseline
	entry1 := &BgpDbEntry{
		Prefix:         net.ParseIP("192.168.224.1"),
		PrefixLen:      24,
		PathAttributes: nil,
	}
	// Same as Baseline
	entry2 := &BgpDbEntry{
		Prefix:         net.ParseIP("192.168.224.1"),
		PrefixLen:      24,
		PathAttributes: nil,
	}

	// Different PrefixLen
	entry3 := &BgpDbEntry{
		Prefix:         net.ParseIP("192.168.224.1"),
		PrefixLen:      32,
		PathAttributes: nil,
	}

	// Different Prefix
	entry4 := &BgpDbEntry{
		Prefix:         net.ParseIP("1.1.1.1"),
		PrefixLen:      24,
		PathAttributes: nil,
	}

	if entry1.Hash() != entry2.Hash() {
		t.Error("entry1 and entry2 should provide the same hash")
	}

	if entry1.Hash() == entry3.Hash() {
		t.Error("entry1 and entry3 should yield different hashes")
	}

	if entry1.Hash() == entry4.Hash() {
		t.Error("entry1 and entry4 should yield different hashes")
	}
}

package daemon

import (
	"crypto/md5"
	"encoding/binary"
	"net"

	bgp "github.com/gopher-net/gopher-net/third-party/github.com/gobgp/packet"
)

// BgpDbTable contains a map of BgpDbEntry keyed on a hash of Prefix and PrefixLen
type BgpDbTable map[[16]byte]BgpDbEntry

// BgpDbEntry is an entry in the Bgp Database for a specific peer
type BgpDbEntry struct {
	Prefix         net.IP
	PrefixLen      int
	PathAttributes []bgp.PathAttributeInterface
}

// Hash returns the MD5sum of the Prefix and PrefixLen
func (e *BgpDbEntry) Hash() [16]byte {
	p1 := e.Prefix.To4()
	p2 := make([]byte, 2)
	binary.BigEndian.PutUint16(p2, uint16(e.PrefixLen))
	data := append(p1, p2...)
	return md5.Sum(data)
}

// BgpDatabase is the database of all learned routes
type BgpDatabase struct {
	Table map[string]BgpDbTable
}

func NewBgpDatabase() *BgpDatabase {
	return &BgpDatabase{
		map[string]BgpDbTable{},
	}
}

func (db *BgpDatabase) AddEntry(peer net.IP, entry *BgpDbEntry) {
	_, ok := db.Table[peer.String()]

	if !ok {
		db.Table[peer.String()] = BgpDbTable{}
	}

	table := db.Table[peer.String()]
	table[entry.Hash()] = *entry
}

func (db *BgpDatabase) RemoveEntry(peer net.IP, entry *BgpDbEntry) error {
	_, ok := db.Table[peer.String()]

	if !ok {
		// error
	}

	table := db.Table[peer.String()]
	delete(table, entry.Hash())

	return nil
}

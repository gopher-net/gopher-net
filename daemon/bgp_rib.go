package daemon

import (
	"fmt"
	"net"

	bgp "github.com/gopher-net/gopher-net/third-party/github.com/gobgp/packet"
)

func NewRibEntry() RibEntry {
	rib := RibEntry{
		NextHop:        nil,
		NetworkSegment: 0,
		DestIpPrefix:   nil,
		Length:         0,
		Gateway:        nil,
	}
	return rib
}

// RibEntry is an entry in the RoutingTable
type RibEntry struct {
	NextHop        net.IP
	NetworkSegment int
	DestIpPrefix   net.IP
	Length         int
	Gateway        net.IP
	Interface      net.IP
	Metric         int
}

// RoutingTable represents a RoutingInformationBase
type RoutingTable struct {
	// ToDo: we should replace this with a Radix tree for LPM
	Rib []RibEntry
}

// Add a new Rib entry
func (r *RoutingTable) Add(entry *RibEntry) {
	r.Rib = append(r.Rib, *entry)
}

// Print displays the content of the Routing Table
func (r *RoutingTable) Print() string {
	result := fmt.Sprintf("%+v\n", r.Rib)
	return result
}

func Ipv4PrefixConcat(p *bgp.NLRInfo) string {
	return fmt.Sprintf("%v/%v", p.Prefix.String(), p.Length)
}

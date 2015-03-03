package daemon

import (
	"log"
	"net"
	"testing"
)

func TestRibPrint(t *testing.T) {
	rib := &RoutingTable{}
	newEntry := &RibEntry{
		DestIpPrefix: net.ParseIP("10.10.10.10"),
		Length:       24,
		Gateway:      net.ParseIP("10.10.10.254"),
		Interface:    net.ParseIP("127.0.0.1"),
		Metric:       10,
	}

	newEntry2 := &RibEntry{
		DestIpPrefix: net.ParseIP("10.10.10.11"),
		Length:       24,
		Gateway:      net.ParseIP("10.10.10.254"),
		Interface:    net.ParseIP("127.0.0.1"),
		Metric:       10,
	}

	rib.Add(newEntry)
	rib.Add(newEntry2)
	log.Print(rib.Print())
}

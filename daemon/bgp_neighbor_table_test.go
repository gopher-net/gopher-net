package daemon

import (
	"testing"
)

func TestBgpEntry(t *testing.T) {

	n1 := new(BgpNeighbor)
	n1.NeighborName = "router1"
	n1.Id = "169.1.1.10"
	if n1.NeighborName != "router1" {
		t.Errorf("Bad neighbor entry %v", n1)
	}

	n2 := new(BgpNeighbor)
	n2.NeighborName = "router2"
	n2.Id = "169.1.1.11"
	n2.As = 10
	if n2.NeighborName != "router2" {
		t.Errorf("Bad neighbor entry %v", n2)
	}
}

func TestBgpDbEntry(t *testing.T) {

	// instantiate a new neighbor
	p := new(BgpNeighbor)
	p.NeighborName = "router1"
	p.Id = "169.1.1.10"
	// instantiate a new neighbor DB
	bgpDb := NewBgpNeighborDb()
	bgpDb.AddBgpNeighbor("foo3", *p)
	bgpDb.AddBgpNeighbor("169.1.1.12", *p)
	neighbor := bgpDb.GetBgpNeighbor("foo3")

	if neighbor.Version > 0 {
		t.Error("failed to return bgp neighbors")
	}

	n1, ok := bgpDb.AddNeighbor("169.1.1.14")

	if !ok {
		t.Error("failed to add a neighbor w/ a constructor: ", n1)
	}

	if n1.ConnectRetryTime < 1 {
		t.Error("constructor values failed: ", n1.ConnectRetryTime)
	}

	neighborlist := bgpDb.GetBgpNeighborList()
	if len(neighborlist) < 1 {
		t.Error("failed to list bgp neighbors")
	}
}

func TestBgpNeighborRemove(t *testing.T) {
	bgpDb := NewBgpNeighborDb()
	n1, ok := bgpDb.AddNeighbor("169.1.1.14")
	if !ok {
		t.Error("failed to add a neighbor w/ a constructor: ", n1)
	}

	if n1.Version < 1 {
		t.Error("AddNeighbor constructor failed for: ", n1)
	}

	n2 := new(BgpNeighbor)
	n2.NeighborName = "router2"
	n2.Id = "router-id-169.1.1.11"
	n2.As = 10
	if n2.As != 10 {
		t.Error("New neighbor failed for: ", n2)
	}

	bgpDb.AddBgpNeighbor("169.1.1.15", *n2)
	p := bgpDb.GetBgpNeighbor("169.1.1.15")
	if p.As > 1 && p.NeighborName != "router2" {
		t.Error("GetBgpNeighbor failed")
	}

	bgpDb.RemoveBgpNeighbor("169.1.1.15")
	p = bgpDb.GetBgpNeighbor("169.1.1.15")

	if _, ok := bgpDb.bgpNeighbors["169.1.1.15"]; ok {
		t.Error("RemoveBgpNeighbor failed")
	}

	bgpDb.GetBgpNeighbor("169.1.1.14")
	if !ok {
		t.Error("Remove neighbor deleted too many neighbors")
	}
}

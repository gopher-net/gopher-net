package daemon

import (
	"fmt"
	"net"
	"sync"
)

type BgpNeighbor struct {
	Version             int
	As                  int
	MessagesReceived    int
	MessagesSent        int
	TableVersion        int
	FsmState            string
	ConnectRetryCounter int
	ConnectRetryTimer   int
	ConnectRetryTime    int
	HoldTimer           int
	HoldTime            int
	KeepaliveTimer      int
	KeepaliveTime       int
	NeighborName        string
	Id                  string
	IpAddr              string
	Conn                net.Conn
}

// NewBgpNeighbor returns a BgpNeighbor object with values initialized at their defaults according to RFC 4271
func NewBgpNeighbor(nodeId string) (BgpNeighbor, bool) {
	neighbor := BgpNeighbor{
		Version:             4,
		As:                  0,
		MessagesReceived:    0,
		MessagesSent:        0,
		TableVersion:        0,
		FsmState:            "IDLE",
		ConnectRetryCounter: 0,
		ConnectRetryTimer:   0,
		ConnectRetryTime:    120,
		HoldTimer:           0,
		HoldTime:            90,
		KeepaliveTimer:      0,
		KeepaliveTime:       30,
		NeighborName:        "",
		Id:                  "",
	}
	return neighbor, true
}

// NeighborDatabase is a map of neighbor IP addresses to a NeighborDetails
type BgpNeighborDb struct {
	lock         *sync.RWMutex
	bgpNeighbors map[string]BgpNeighbor // Ip address of the neighbor
}

// Map the IP address key and bgp neighbor details as the value
func NewBgpNeighborDb() *BgpNeighborDb {
	return &BgpNeighborDb{
		bgpNeighbors: make(map[string]BgpNeighbor),
		lock:         new(sync.RWMutex)}
}

// Add a neighbor with neighbor constructor
func (db *BgpNeighborDb) AddNeighbor(neighborIp string) (BgpNeighbor, bool) {
	db.lock.Lock()
	defer db.lock.Unlock()
	if neighbor, ok := db.bgpNeighbors[neighborIp]; ok {
		return neighbor, true
	}
	neighbor, ok := NewBgpNeighbor(neighborIp)
	if ok {
		db.bgpNeighbors[neighborIp] = neighbor
		return neighbor, true
	}
	return neighbor, false
}

// Add a neighbor without neighbor the constructor
func (db *BgpNeighborDb) AddBgpNeighbor(neighborIp string, bgpNeighbors BgpNeighbor) {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.bgpNeighbors[neighborIp] = bgpNeighbors
}

// Remove a BGP peer from the BgpNeighborDb neighbor map
func (db *BgpNeighborDb) RemoveBgpNeighbor(neighborIp string) {
	db.lock.Lock()
	defer db.lock.Unlock()
	if _, ok := db.bgpNeighbors[neighborIp]; ok {
		delete(db.bgpNeighbors, neighborIp)
	} else {
		fmt.Printf("bgpNeighbor %v doesnt exist", neighborIp)
	}
}

// Return a single neighbor from the key(neighbor ip)
func (db *BgpNeighborDb) GetBgpNeighbor(neighborIp string) BgpNeighbor {
	db.lock.Lock()
	defer db.lock.Unlock()
	r := db.bgpNeighbors[neighborIp]
	return r
}

// Return all BGP neighbor IPs
func (g *BgpNeighborDb) GetBgpNeighborList() []string {
	nodelist := []string{}
	// get all neighbots and append them to the list.
	for neighbor := range g.bgpNeighbors {
		nodelist = append(nodelist, neighbor)
	}
	return nodelist
}

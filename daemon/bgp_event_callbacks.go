package daemon

import (
	"fmt"

	log "github.com/gopher-net/gopher-net/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/gopher-net/gopher-net/configuration"
	bgp "github.com/gopher-net/gopher-net/third-party/github.com/gobgp/packet"
	"github.com/gopher-net/gopher-net/third-party/github.com/gobgp/table"
)

type NodeEvent int

const (
	_ NodeEvent = iota
	EVENT_PREFIX_BEST
	EVENT_PREFIX_WITHDRAWN
	EVENT_NODE_JOIN
	EVENT_NODE_REMOVE
)

// todo: add to mathod calls
func (t NodeEvent) String() string {
	switch t {
	case EVENT_PREFIX_BEST:
		return "prefix_added"
	case EVENT_PREFIX_WITHDRAWN:
		return "prefix_withdrawn"
	case EVENT_NODE_JOIN:
		return "node_joined"
	case EVENT_NODE_REMOVE:
		return "node_removed"
	default:
		panic(fmt.Sprintf("unknown event: [ %d ]", t))
	}
}

func ContainerPrefixEvent(routes []table.Path, bgpMsgBody *bgp.BGPUpdate) {
	for _, route := range routes {
		jsn, _ := route.MarshalJSON()
		log.Debug("Container Event: Route Added Notification")
		log.Debugf("Container Event: Prefix Added: -> [ %s ]", route.GetPrefix())
		log.Debugf("Container Event: Prefix Nexthop: -> [ %s ]", route.GetNexthop())
		log.Debugf("Container Event: All NLRI -> [ %s ]", jsn)

		if route.IsWithdraw() {
			wd := route.GetNlri().(*bgp.WithdrawnRoute)
			log.Debugln("Container Event: Route Withdraw Notification")
			log.Debugf("Container Event: Prefix Withdrawn -> [ %s ]/[ %d ]", wd.IPAddrPrefix.String(), wd.Length)
		}
	}
}

func (d *FSMHandler) NodeAddedFSMEvent(nConf *configuration.NeighborType) {
	log.Debugln("Container Event: New Neighbor Added")
	log.Debugf("Container Event: Established Neighbor IP address -> [ %d ]", nConf.PeerAs)
	log.Debugf("Container Event: Established Neighbor IP address -> [ %s ]", nConf.NeighborAddress)
	log.Debugf("Container Event: Established Neighbor FSM State -> [ %s ]", d.fsm.state.String())
}

func (d *FSMHandler) NodeRemovedFSMEvent(nConf *configuration.NeighborType) {
	log.Debugln("Container Event: Neighbor Removed (FSM Idle)")
	log.Debugf("Container Event: Idle Neighbor IP address -> [ %d ]", nConf.PeerAs)
	log.Debugf("Container Event: Neighbor Removed Idle Neighbor IP address -> [ [ %s ] ]", nConf.NeighborAddress)

}

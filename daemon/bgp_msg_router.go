package daemon

import (
	"encoding/json"
	"fmt"
	"github.com/gopher-net/gopher-net/api"
	"github.com/gopher-net/gopher-net/configuration"
	"net"

	log "github.com/gopher-net/gopher-net/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	bgp "github.com/gopher-net/gopher-net/third-party/github.com/gobgp/packet"
	"github.com/gopher-net/gopher-net/third-party/github.com/gobgp/table"
)

const AS_SEQUENCE uint8 = 2

type RestRoute struct {
	IP4prefix    string `json:"ip_prefix"`
	NextHop      net.IP `json:"ip_nexthop"`
	AS           uint32 `json:"source_as"`
	RF           string `json:"route_family"`
	NeighborAddr net.IP `json:"neighbor_ip"`
	RouterId     net.IP `json:"neighbor_router_id"`
	LocalId      net.IP `json:"local_router_id"`
	ExCommunity  string `json:"extended_community"`
}

func (daemon *Daemon) handleRest(restReq *api.RestRequest) {
	switch restReq.RequestType {

	case api.API_NEIGHBORS:
		result := &api.RestResponse{}
		neighborList := make([]*Neighbor, 0)
		for _, info := range daemon.neighborMap {
			neighborList = append(neighborList, info.neighbor)
		}
		j, _ := json.MarshalIndent(neighborList, "", "\t")
		result.Data = j
		restReq.ResponseCh <- result
		close(restReq.ResponseCh)

	case api.API_NEIGHBOR:
		remoteAddr := restReq.RemoteAddr
		result := &api.RestResponse{}
		info, found := daemon.neighborMap[remoteAddr]
		if found {
			j, _ := json.MarshalIndent(info.neighbor, "", "\t")
			result.Data = j
		} else {
			result.ResponseErr = fmt.Errorf("Neighbor that has [ %s ] does not exist.", remoteAddr)
		}
		restReq.ResponseCh <- result
		close(restReq.ResponseCh)

	case api.API_CONF_GLOBAL:
		result := &api.RestResponse{}
		j, _ := json.MarshalIndent(daemon.bgpConfig, "", "\t")
		result.Data = j
		restReq.ResponseCh <- result
		close(restReq.ResponseCh)

	case api.API_CONF_NEIGHBORS:
		result := &api.RestResponse{}
		var neighborList []*configuration.NeighborType
		for _, neighbor := range daemon.neighborMap {
			neighborList = append(neighborList, &neighbor.neighbor.neighborConfig)
		}
		j, _ := json.MarshalIndent(neighborList, "", "\t")
		result.Data = j
		restReq.ResponseCh <- result
		close(restReq.ResponseCh)

	case api.API_ADJ_RIB_LOCAL:
		remoteAddr := restReq.RemoteAddr
		result := &api.RestResponse{}
		info, found := daemon.neighborMap[remoteAddr]
		if found {
			msg := &daemonMsg{
				msgType: SRV_MSG_API,
				msgData: restReq,
			}
			info.neighbor.daemonMsgCh <- msg
		} else {
			result.ResponseErr = fmt.Errorf("Neighbor that has [ %s ] does not exist.", remoteAddr)
			restReq.ResponseCh <- result
			close(restReq.ResponseCh)
		}

	case api.API_ROUTES:
		result := &api.RestResponse{}
		var routeTables []*RestRoute
		for _, peer := range daemon.neighborMap {
			routes := peer.neighbor.adjRib.GetInPathList(bgp.RF_IPv4_UC)
			for i, _ := range routes {
				prefix := routes[i].GetNlri().(*bgp.NLRInfo).IPAddrPrefix.Prefix
				mask := routes[i].GetNlri().(*bgp.NLRInfo).Length
				nexthop := routes[i].GetNexthop()
				ipRoutes := new(RestRoute)
				ipRoutes.IP4prefix = CidrToString(prefix, mask)
				routeTables = append(routeTables,
					&RestRoute{
						IP4prefix:    CidrToString(prefix, mask),
						AS:           peer.neighbor.neighborInfo.AS,
						RouterId:     peer.neighbor.neighborInfo.ID,
						RF:           peer.neighbor.neighborInfo.RF.String(),
						NextHop:      nexthop,
						NeighborAddr: peer.neighbor.fsm.neighborConfig.NeighborAddress,
						LocalId:      peer.neighbor.neighborInfo.LocalID,
					})
			}
		}
		j, _ := json.MarshalIndent(routeTables, "", "\t")
		result.Data = j
		restReq.ResponseCh <- result
		close(restReq.ResponseCh)

	case api.API_LOCAL_RIB:
		result := &api.RestResponse{}
		for _, peer := range daemon.neighborMap {
			tables, _ := peer.neighbor.rib.Tables[bgp.RF_IPv4_UC].MarshalJSON()
			result.Data = tables
		}
		restReq.ResponseCh <- result
		close(restReq.ResponseCh)

	case api.API_RIB_OUT:
		result := &api.RestResponse{}
		ribOutList := make([]table.Path, 0)
		for _, peer := range daemon.neighborMap {
			out := peer.neighbor.adjRib.GetOutPathList(peer.neighbor.rf)
			for _, ribOut := range out {
				ribOutList = append(ribOutList, ribOut)
			}
		}
		j, _ := json.MarshalIndent(ribOutList, "", "\t")
		result.Data = j
		restReq.ResponseCh <- result
		close(restReq.ResponseCh)

	case api.API_RIB_IN:
		result := &api.RestResponse{}
		ribInList := make([]table.Path, 0)
		for _, peer := range daemon.neighborMap {
			in := peer.neighbor.adjRib.GetInPathList(peer.neighbor.rf)
			for _, ribIn := range in {
				ribInList = append(ribInList, ribIn)
			}
		}
		j, _ := json.MarshalIndent(ribInList, "", "\t")
		result.Data = j
		restReq.ResponseCh <- result
		close(restReq.ResponseCh)

	case api.API_ADD_NEIGHBOR:
		result := &api.RestResponse{}
		neighborAddr := restReq.NodeConfig.NeighborAddress.String()
		ok := daemon.checkBgpPeerAddr(neighborAddr)
		if !ok {
			log.Debugf("Specified neighbor IP [%s] to add is bound to the local machine", neighborAddr)
		} else {
			ok = daemon.checkIfPeerExists(neighborAddr)
			if !ok {
				log.Debugf("Specified neighbor IP config [%s] already exists as a bgp neighbor", neighborAddr)
				result.ResponseErr = fmt.Errorf("Specified neighbor IP config [%s] already exists", neighborAddr)
			} else {
				log.Infof("Initiating peer to neighbor at: %s", neighborAddr)
				configuration.SetNeighborTypeDefault(&restReq.NodeConfig)
				daemon.NeighborAdd(restReq.NodeConfig)
				uuid, err := CreateUUID()
				if err != nil {
					log.Errorln("error generating a node uuid", err)
				}
				restReq.NodeConfig.Description = uuid
			}
		}
		j, _ := json.MarshalIndent(fmt.Sprint("Neighbor successfully added"), "", "\t")

		result.Data = j
		restReq.ResponseCh <- result
		close(restReq.ResponseCh)

	case api.API_DEL_NEIGHBOR:
		result := &api.RestResponse{}
		_, exists := daemon.neighborMap[restReq.NodeConfig.NeighborAddress.String()]
		if !exists {
			result.ResponseErr = fmt.Errorf(
				"Failed to delete the node, an active node with the address [ %s ] was not found",
				restReq.NodeConfig.NeighborAddress.String())
		}
		log.Debugf("Attempting to build BGP peering to neighbor",
			restReq.NodeConfig.NeighborAddress.String())
		configuration.SetNeighborTypeDefault(&restReq.NodeConfig)
		daemon.NeighborDelete(restReq.NodeConfig)
		returnMsg := fmt.Sprintf("Node delete requested")
		j, _ := json.MarshalIndent(returnMsg, "", "\t")
		result.Data = j
		restReq.ResponseCh <- result
		close(restReq.ResponseCh)

	case api.API_ADD_ROUTE:
		// TODO routes are not inserted if a host is not conencted
		result := &api.RestResponse{}
		log.Debugf("Adding route:  [ Prefix: %s , Netmask: %d , Nexthop: %s ]",
			restReq.RestRoute.IpPrefix, restReq.RestRoute.PrefixMask, restReq.RestRoute.NextHop)

		if restReq.RestRoute.IpPrefix == "" || !isValidIp(restReq.RestRoute.NextHop) {
			log.Errorln("Error adding route: IP Prefix, Mask and IP Nexthop are mandatory.")
			result.ResponseErr = fmt.Errorf("IP Prefix, Mask and IP Nexthop are mandatory.")
			return
		}
		if restReq.RestRoute.NextHop == "" || !isValidIp(restReq.RestRoute.NextHop) {
			log.Errorln("Error adding route: IP Prefix, Mask and IP Nexthop are mandatory.")
			result.ResponseErr = fmt.Errorf("IP Prefix, Mask and IP Nexthop are mandatory.")
			return
		}
		if restReq.RestRoute.PrefixMask == 0 { // TODO What about default routes?
			log.Errorln("Error adding route: IP Prefix, Mask and IP Nexthop are mandatory.")
			result.ResponseErr = fmt.Errorf("IP Prefix, Mask and IP Nexthop are mandatory.")
			return
		}
		for _, p := range daemon.neighborMap {
			if p.neighbor.neighborConfig.BgpNeighborCommonState.State != uint32(bgp.BGP_FSM_ESTABLISHED) {
				continue
			}
			origin := bgp.NewPathAttributeOrigin(0)
			aspathParam := []bgp.AsPathParamInterface{}
			aspath := bgp.NewPathAttributeAsPath(aspathParam)
			nexthop := bgp.NewPathAttributeNextHop(restReq.RestRoute.NextHop)
			med := bgp.NewPathAttributeMultiExitDisc(0)
			localpref := bgp.NewPathAttributeLocalPref(100)
			pathAttributes := []bgp.PathAttributeInterface{
				origin,
				aspath,
				nexthop,
				med,
				localpref,
			}
			if restReq.RestRoute.ExCommunity != "" && len(restReq.RestRoute.ExCommunity) < 8 {
				exCommStr := restReq.RestRoute.ExCommunity
				exCommunity := []byte(exCommStr)
				exComm := []bgp.ExtendedCommunityInterface{
					&bgp.OpaqueExtended{Value: exCommunity},
				}
				excommunity := bgp.NewPathAttributeExtendedCommunities(exComm)
				pathAttributes = []bgp.PathAttributeInterface{
					origin,
					aspath,
					nexthop,
					med,
					localpref,
					excommunity,
				}
			}
			prefix := *bgp.NewNLRInfo(restReq.RestRoute.PrefixMask, restReq.RestRoute.IpPrefix)
			nlri := []bgp.NLRInfo{prefix}
			withdrawnRoutes := []bgp.WithdrawnRoute{}
			updateMsg := bgp.NewBGPUpdateMessage(withdrawnRoutes, pathAttributes, nlri)
			p.neighbor.outgoing <- updateMsg
			returnMsg := fmt.Sprintf("Added prefix [ Prefix: %s , Netmask: %d, Nexthop: %s ]",
				restReq.RestRoute.IpPrefix, restReq.RestRoute.PrefixMask, restReq.RestRoute.NextHop)
			j, _ := json.MarshalIndent(returnMsg, "", "\t")
			result.Data = j
			restReq.ResponseCh <- result
			defer close(restReq.ResponseCh)
		}

	case api.API_DEL_ROUTE:
		// TODO: Verify if route exists in RIB
		result := &api.RestResponse{} // Todo condense and cleanup null checks
		log.Debugf("Deleting route:  [ Prefix: %s , Netmask: %d ]",
			restReq.RestRoute.IpPrefix, restReq.RestRoute.PrefixMask)

		if restReq.RestRoute.IpPrefix == "" || !isValidIp(restReq.RestRoute.NextHop) {
			log.Errorln("Error adding route: IP Prefix, Mask and Nexthop are mandatory.")
			result.ResponseErr = fmt.Errorf("IP Prefix, Mask and IP Nexthop are mandatory.")
			return
		}
		if restReq.RestRoute.PrefixMask == 0 {
			log.Errorln("Error adding route: IP Prefix, Mask and Nexthop are mandatory.")
			result.ResponseErr = fmt.Errorf("IP Prefix, Mask and Nexthop are mandatory.")
		}
		for _, p := range daemon.neighborMap {
			if p.neighbor.neighborConfig.BgpNeighborCommonState.State != uint32(bgp.BGP_FSM_ESTABLISHED) {
				continue
			}
			origin := bgp.NewPathAttributeOrigin(0)
			aspathParam := []bgp.AsPathParamInterface{}
			aspath := bgp.NewPathAttributeAsPath(aspathParam)
			nexthop := bgp.NewPathAttributeNextHop("172.16.86.190")
			med := bgp.NewPathAttributeMultiExitDisc(0)
			localpref := bgp.NewPathAttributeLocalPref(100)
			pathAttributes := []bgp.PathAttributeInterface{
				origin,
				aspath,
				nexthop,
				med,
				localpref,
			}
			nlri := []bgp.NLRInfo{}
			w := bgp.WithdrawnRoute{*bgp.NewIPAddrPrefix(restReq.RestRoute.PrefixMask, restReq.RestRoute.IpPrefix)}
			withdrawnRoutes := []bgp.WithdrawnRoute{w}
			updateMsg := bgp.NewBGPUpdateMessage(withdrawnRoutes, pathAttributes, nlri)
			p.neighbor.outgoing <- updateMsg
			returnMsg := fmt.Sprintf("Deleting prefix [ Prefix: %s , Netmask: %d ]",
				restReq.RestRoute.IpPrefix, restReq.RestRoute.PrefixMask)
			j, _ := json.MarshalIndent(returnMsg, "", "\t")
			result.Data = j
			restReq.ResponseCh <- result
			defer close(restReq.ResponseCh)
		}
	}
}

func (restRoutes *RestRoute) String() string {
	str := fmt.Sprintf("BGP_AS Source: %d, ", restRoutes.AS)
	str = str + fmt.Sprintf(" IP_PREFIX: %s, ", restRoutes.IP4prefix)
	str = str + fmt.Sprintf(" IP_NEXTHOP: %s, ", restRoutes.NextHop)
	str = str + fmt.Sprintf(" BGP_ID: %s, ", restRoutes.RouterId)
	return str
}

func CidrToString(ipaddress net.IP, mask uint8) string {
	return fmt.Sprintf("%s/%d", ipaddress, mask)
}

func ResolveIp(ipStr string) (*net.IPAddr, error) {
	ipaddr, err := net.ResolveIPAddr("ip", ipStr)
	return ipaddr, err
}

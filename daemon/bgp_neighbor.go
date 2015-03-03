package daemon

import (
	"encoding/json"
	"github.com/gopher-net/gopher-net/api"
	"github.com/gopher-net/gopher-net/configuration"
	"net"
	"time"

	log "github.com/gopher-net/gopher-net/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	tomb "github.com/gopher-net/gopher-net/Godeps/_workspace/src/gopkg.in/tomb.v2"
	bgp "github.com/gopher-net/gopher-net/third-party/github.com/gobgp/packet"
	"github.com/gopher-net/gopher-net/third-party/github.com/gobgp/table"
)

type neighborMsgType int

const (
	_ neighborMsgType = iota
	PEER_MSG_PATH
	PEER_MSG_PEER_DOWN
)

const (
	FSM_CHANNEL_LENGTH = 1024
	FLOP_THRESHOLD     = time.Second * 30
)

type neighborMsg struct {
	msgType neighborMsgType
	msgData interface{}
}

type Neighbor struct {
	t              tomb.Tomb
	globalConfig   configuration.GlobalType
	neighborConfig configuration.NeighborType
	acceptedConnCh chan *net.TCPConn
	daemonMsgCh    chan *daemonMsg
	neighborMsgCh  chan *neighborMsg
	fsm            *FSM
	adjRib         *table.AdjRib
	rib            *table.TableManager
	rf             bgp.RouteFamily
	capMap         map[bgp.BGPCapabilityCode]bgp.ParameterCapabilityInterface
	neighborInfo   *table.PeerInfo
	siblings       map[string]*daemonMsgDataNeighbor
	outgoing       chan *bgp.BGPMessage
}

func NewNeighbor(g configuration.GlobalType, neighbor configuration.NeighborType, daemonMsgCh chan *daemonMsg, neighborMsgCh chan *neighborMsg, neighborList []*daemonMsgDataNeighbor) *Neighbor {
	p := &Neighbor{
		globalConfig:   g,
		neighborConfig: neighbor,
		acceptedConnCh: make(chan *net.TCPConn),
		daemonMsgCh:    daemonMsgCh,
		neighborMsgCh:  neighborMsgCh,
		capMap:         make(map[bgp.BGPCapabilityCode]bgp.ParameterCapabilityInterface),
	}
	p.siblings = make(map[string]*daemonMsgDataNeighbor)
	for _, s := range neighborList {
		p.siblings[s.address.String()] = s
	}
	p.fsm = NewFSM(&g, &neighbor, p.acceptedConnCh)
	neighbor.BgpNeighborCommonState.State = uint32(bgp.BGP_FSM_IDLE)
	neighbor.BgpNeighborCommonState.Downtime = time.Now()
	if neighbor.NeighborAddress.To4() != nil {
		p.rf = bgp.RF_IPv4_UC
	} else {
		p.rf = bgp.RF_IPv6_UC
	}
	p.neighborInfo = &table.PeerInfo{
		AS:      neighbor.PeerAs,
		LocalID: g.RouterId,
		RF:      p.rf,
		Address: neighbor.NeighborAddress,
	}
	p.adjRib = table.NewAdjRib()
	p.rib = table.NewTableManager()
	p.t.Go(p.loop)
	return p
}

func (neighbor *Neighbor) handleBGPmessage(m *bgp.BGPMessage) {
	log.WithFields(log.Fields{
		"Topic": "Neighbor",
		"Key":   neighbor.neighborConfig.NeighborAddress,
		"data":  m,
	}).Debug("received")

	switch m.Header.Type {
	case bgp.BGP_MSG_OPEN:
		body := m.Body.(*bgp.BGPOpen)
		neighbor.neighborInfo.ID = m.Body.(*bgp.BGPOpen).ID
		for _, p := range body.OptParams {
			paramCap, y := p.(*bgp.OptionParameterCapability)
			if !y {
				continue
			}
			for _, c := range paramCap.Capability {
				neighbor.capMap[c.Code()] = c
			}
		}

	case bgp.BGP_MSG_ROUTE_REFRESH:
		pathList := neighbor.adjRib.GetOutPathList(neighbor.rf)
		neighbor.sendMessages(table.CreateUpdateMsgFromPaths(pathList))
	case bgp.BGP_MSG_UPDATE:
		neighbor.neighborConfig.BgpNeighborCommonState.UpdateRecvTime = time.Now()
		body := m.Body.(*bgp.BGPUpdate)

		table.UpdatePathAttrs4ByteAs(body)
		msg := table.NewProcessMessage(m, neighbor.neighborInfo)
		pathList := msg.ToPathList()
		if len(pathList) == 0 {
			return
		}
		neighbor.adjRib.UpdateIn(pathList)

		// Container Events Call docker_updates.go
		ContainerPrefixEvent(pathList, body)

		pm := &neighborMsg{
			msgType: PEER_MSG_PATH,
			msgData: pathList,
		}
		for _, s := range neighbor.siblings {
			if s.rf != neighbor.rf {
				continue
			}
			s.neighborMsgCh <- pm
		}
	}
}

func (neighbor *Neighbor) sendMessages(msgs []*bgp.BGPMessage) {
	for _, m := range msgs {
		if neighbor.neighborConfig.BgpNeighborCommonState.State != uint32(bgp.BGP_FSM_ESTABLISHED) {
			continue
		}

		if m.Header.Type != bgp.BGP_MSG_UPDATE {
			log.Fatal("not update message ", m.Header.Type)
		}

		_, y := neighbor.capMap[bgp.BGP_CAP_FOUR_OCTET_AS_NUMBER]
		if !y {
			log.WithFields(log.Fields{
				"Topic": "Neighbor",
				"Key":   neighbor.neighborConfig.NeighborAddress,
				"data":  m,
			}).Debug("update for 2byte AS neighbor")
			table.UpdatePathAttrs2ByteAs(m.Body.(*bgp.BGPUpdate))
		}

		neighbor.outgoing <- m
	}
}

func (neighbor *Neighbor) handleREST(restReq *api.RestRequest) {
	result := &api.RestResponse{}
	j, _ := json.Marshal(neighbor.rib.Tables[neighbor.rf])
	result.Data = j
	restReq.ResponseCh <- result
	close(restReq.ResponseCh)
}

func (neighbor *Neighbor) sendUpdateMsgFromPaths(pList []table.Path, wList []table.Path) {
	pathList := append([]table.Path(nil), pList...)
	pathList = append(pathList, wList...)

	for _, p := range wList {
		if !p.IsWithdraw() {
			log.Fatal("withdraw pathlist has non withdraw path")
		}
	}
	neighbor.adjRib.UpdateOut(pathList)
	neighbor.sendMessages(table.CreateUpdateMsgFromPaths(pathList))
}

func (neighbor *Neighbor) handleNeighborMsg(m *neighborMsg) {
	switch m.msgType {
	case PEER_MSG_PATH:
		pList, wList, _ := neighbor.rib.ProcessPaths(m.msgData.([]table.Path))
		neighbor.sendUpdateMsgFromPaths(pList, wList)
	case PEER_MSG_PEER_DOWN:
		pList, wList, _ := neighbor.rib.DeletePathsforPeer(m.msgData.(*table.PeerInfo))
		neighbor.sendUpdateMsgFromPaths(pList, wList)
	}
}

func (neighbor *Neighbor) handleServerMsg(m *daemonMsg) {
	switch m.msgType {
	case SRV_MSG_PEER_ADDED:
		d := m.msgData.(*daemonMsgDataNeighbor)
		neighbor.siblings[d.address.String()] = d
		pathList := neighbor.adjRib.GetInPathList(d.rf)

		if len(pathList) == 0 {
			return
		}
		pm := &neighborMsg{
			msgType: PEER_MSG_PATH,
			msgData: pathList,
		}
		for _, s := range neighbor.siblings {
			if s.rf != neighbor.rf {
				continue
			}
			s.neighborMsgCh <- pm
		}
	case SRV_MSG_PEER_DELETED:

		d := m.msgData.(*table.PeerInfo)
		_, found := neighbor.siblings[d.Address.String()]
		if found {
			delete(neighbor.siblings, d.Address.String())
			pList, wList, _ := neighbor.rib.DeletePathsforPeer(d)
			neighbor.sendUpdateMsgFromPaths(pList, wList)
		} else {
			log.Warning("can not find neighbor: ", d.Address.String())
		}
	case SRV_MSG_API:
		neighbor.handleREST(m.msgData.(*api.RestRequest))
	default:
		log.Fatal("unknown daemon msg type ", m.msgType)
	}
}

// this goroutine handles routing table operations
func (peer *Neighbor) loop() error {
	for {
		//		h := NewFSMHandler(neighbor.fsm)
		incoming := make(chan *fsmMsg, FSM_CHANNEL_LENGTH)
		peer.outgoing = make(chan *bgp.BGPMessage, FSM_CHANNEL_LENGTH)

		h := NewFSMHandler(peer.fsm, incoming, peer.outgoing)
		sameState := true
		for sameState {
			select {
			case <-peer.t.Dying():
				close(peer.acceptedConnCh)
				peer.outgoing <- bgp.NewBGPNotificationMessage(bgp.BGP_ERROR_CEASE, bgp.BGP_ERROR_SUB_PEER_DECONFIGURED, nil)
				h.Wait()
				return nil
			case e := <-incoming:
				switch e.MsgType {
				case FSM_MSG_STATE_CHANGE:
					nextState := e.MsgData.(bgp.FSMState)
					// waits for all goroutines created for the current state
					h.Wait()
					oldState := bgp.FSMState(peer.neighborConfig.BgpNeighborCommonState.State)
					peer.neighborConfig.BgpNeighborCommonState.State = uint32(nextState)
					peer.fsm.StateChange(nextState)
					sameState = false
					if nextState == bgp.BGP_FSM_ESTABLISHED {
						pathList := peer.adjRib.GetOutPathList(peer.rf)
						peer.sendMessages(table.CreateUpdateMsgFromPaths(pathList))
						peer.fsm.neighborConfig.BgpNeighborCommonState.Uptime = time.Now()
						peer.fsm.neighborConfig.BgpNeighborCommonState.EstablishedCount++
					}
					if oldState == bgp.BGP_FSM_ESTABLISHED {
						t := time.Now()
						peer.fsm.neighborConfig.BgpNeighborCommonState.Downtime = t
						if t.Sub(peer.fsm.neighborConfig.BgpNeighborCommonState.Uptime) < FLOP_THRESHOLD {
							peer.fsm.neighborConfig.BgpNeighborCommonState.Flops++
						}
						peer.adjRib.DropAllIn(peer.rf)
						pm := &neighborMsg{
							msgType: PEER_MSG_PEER_DOWN,
							msgData: peer.neighborInfo,
						}
						for _, s := range peer.siblings {
							s.neighborMsgCh <- pm
						}
					}
				case FSM_MSG_BGP_MESSAGE:
					switch m := e.MsgData.(type) {
					case *bgp.MessageError:
						peer.outgoing <- bgp.NewBGPNotificationMessage(m.TypeCode, m.SubTypeCode, m.Data)
					case *bgp.BGPMessage:
						peer.handleBGPmessage(m)
					default:
						log.WithFields(log.Fields{
							"Topic": "Neighbor",
							"Key":   peer.neighborConfig.NeighborAddress,
							"Data":  e.MsgData,
						}).Panic("unknonw msg type")
					}
				}
			case m := <-peer.daemonMsgCh:
				peer.handleServerMsg(m)
			case m := <-peer.neighborMsgCh:
				peer.handleNeighborMsg(m)
			}
		}
	}
}

func (neighbor *Neighbor) Stop() error {
	neighbor.t.Kill(nil)
	return neighbor.t.Wait()
}

func (neighbor *Neighbor) PassConn(conn *net.TCPConn) {
	neighbor.acceptedConnCh <- conn
}

func (neighbor *Neighbor) MarshalJSON() ([]byte, error) {

	f := neighbor.fsm
	c := f.neighborConfig

	p := make(map[string]interface{})
	capList := make([]int, 0)
	for k, _ := range neighbor.capMap {
		capList = append(capList, int(k))
	}

	p["conf"] = struct {
		RemoteIP           string `json:"remote_ip"`
		Id                 string `json:"id"`
		RemoteAS           uint32 `json:"remote_as"`
		CapRefresh         bool   `json:"cap_refresh"`
		CapEnhancedRefresh bool   `json:"cap_enhanced_refresh"`
		RemoteCap          []int
		LocalCap           []int
	}{
		RemoteIP: c.NeighborAddress.String(),
		Id:       neighbor.neighborInfo.ID.To4().String(),
		//Description: "",
		RemoteAS:  c.PeerAs,
		RemoteCap: capList,
		LocalCap:  []int{int(bgp.BGP_CAP_MULTIPROTOCOL), int(bgp.BGP_CAP_ROUTE_REFRESH), int(bgp.BGP_CAP_FOUR_OCTET_AS_NUMBER)},
	}

	s := c.BgpNeighborCommonState

	uptime := float64(0)
	if !s.Uptime.IsZero() {
		uptime = time.Now().Sub(s.Uptime).Seconds()
	}
	downtime := float64(0)
	if !s.Downtime.IsZero() {
		downtime = time.Now().Sub(s.Downtime).Seconds()
	}

	p["info"] = struct {
		BgpState                  string  `json:"bgp_state"`
		FsmEstablishedTransitions uint32  `json:"fsm_established_transitions"`
		TotalMessageOut           uint32  `json:"total_message_out"`
		TotalMessageIn            uint32  `json:"total_message_in"`
		UpdateMessageOut          uint32  `json:"update_message_out"`
		UpdateMessageIn           uint32  `json:"update_message_in"`
		KeepAliveMessageOut       uint32  `json:"keepalive_message_out"`
		KeepAliveMessageIn        uint32  `json:"keepalive_message_in"`
		OpenMessageOut            uint32  `json:"open_message_out"`
		OpenMessageIn             uint32  `json:"open_message_in"`
		NotificationOut           uint32  `json:"notification_out"`
		NotificationIn            uint32  `json:"notification_in"`
		RefreshMessageOut         uint32  `json:"refresh_message_out"`
		RefreshMessageIn          uint32  `json:"refresh_message_in"`
		Uptime                    float64 `json:"uptime"`
		Downtime                  float64 `json:"downtime"`
		LastError                 string  `json:"last_error"`
		Received                  uint32
		Accepted                  uint32
		Advertized                uint32
		OutQ                      int
		Flops                     uint32
	}{

		BgpState:                  f.state.String(),
		FsmEstablishedTransitions: s.EstablishedCount,
		TotalMessageOut:           s.TotalOut,
		TotalMessageIn:            s.TotalIn,
		UpdateMessageOut:          s.UpdateOut,
		UpdateMessageIn:           s.UpdateIn,
		KeepAliveMessageOut:       s.KeepaliveOut,
		KeepAliveMessageIn:        s.KeepaliveIn,
		OpenMessageOut:            s.OpenOut,
		OpenMessageIn:             s.OpenIn,
		NotificationOut:           s.NotifyOut,
		NotificationIn:            s.NotifyIn,
		RefreshMessageOut:         s.RefreshOut,
		RefreshMessageIn:          s.RefreshIn,
		Uptime:                    uptime,
		Downtime:                  downtime,
		Received:                  uint32(neighbor.adjRib.GetInCount(neighbor.rf)),
		Accepted:                  uint32(neighbor.adjRib.GetInCount(neighbor.rf)),
		Advertized:                uint32(neighbor.adjRib.GetOutCount(neighbor.rf)),
		OutQ:                      len(neighbor.outgoing),
		Flops:                     s.Flops,
	}

	return json.Marshal(p)
}

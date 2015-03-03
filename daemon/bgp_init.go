package daemon

import (
	"fmt"
	"github.com/gopher-net/gopher-net/api"
	"github.com/gopher-net/gopher-net/configuration"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/gopher-net/gopher-net/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	bgp "github.com/gopher-net/gopher-net/third-party/github.com/gobgp/packet"
)

type daemonMsgType int

const (
	_ daemonMsgType = iota
	SRV_MSG_PEER_ADDED
	SRV_MSG_PEER_DELETED
	SRV_MSG_API
)

type daemonMsg struct {
	msgType daemonMsgType
	msgData interface{}
}

type daemonMsgDataNeighbor struct {
	neighborMsgCh chan *neighborMsg
	address       net.IP
	rf            bgp.RouteFamily
}

type neighborMapInfo struct {
	neighbor        *Neighbor
	daemonMsgCh     chan *daemonMsg
	neighborMsgCh   chan *neighborMsg
	neighborMsgData *daemonMsgDataNeighbor
}

type Daemon struct {
	bgpConfig         configuration.BgpType
	globalTypeCh      chan configuration.GlobalType
	addedNeighborCh   chan configuration.NeighborType
	deletedNeighborCh chan configuration.NeighborType
	RestReqCh         chan *api.RestRequest
	listenPort        int
	neighborMap       map[string]neighborMapInfo
}

func NewBgpDaemon(port int) *Daemon {
	b := Daemon{}
	b.globalTypeCh = make(chan configuration.GlobalType)
	b.addedNeighborCh = make(chan configuration.NeighborType)
	b.deletedNeighborCh = make(chan configuration.NeighborType)
	b.RestReqCh = make(chan *api.RestRequest, 1)
	b.listenPort = port
	return &b
}

func InitDialNeighbor(neighborAddr string, acceptCh chan *net.TCPConn) {
	neighborIpPort := fmt.Sprint(neighborAddr, ":", bgp.BGP_PORT)
	log.Infof("Initiating peer to configured neighbor: %s", neighborIpPort)
	go func() {
		conn, err := net.DialTimeout("tcp", neighborIpPort, 20*time.Second)
		if err != nil {
			log.Errorf("%s verify a BGP listener is running a bgp service and is reachable on port 179 \n", err)
		} else {
			acceptCh <- conn.(*net.TCPConn)
		}
	}()
}

func listenAndAccept(proto string, port int, ch chan *net.TCPConn) (*net.TCPListener, error) {
	service := ":" + strconv.Itoa(port)
	addr, _ := net.ResolveTCPAddr(proto, service)
	l, err := net.ListenTCP(proto, addr)
	if err != nil {
		log.Info(err)
		return nil, err
	}
	go func() {
		for {
			conn, err := l.AcceptTCP()
			if err != nil {
				log.Info(err)
				continue
			}
			ch <- conn
		}
	}()
	return l, nil
}

func (daemon *Daemon) Serve() {
	daemon.bgpConfig.Global = <-daemon.globalTypeCh
	listenerMap := make(map[string]*net.TCPListener)
	acceptCh := make(chan *net.TCPConn)
	l4, err1 := listenAndAccept("tcp4", daemon.listenPort, acceptCh)
	listenerMap["tcp4"] = l4
	if listenerMap["tcp4"] == l4 {
		for _, peerinfo := range daemon.neighborMap {
			neighborAddr := peerinfo.neighbor.neighborConfig.NeighborAddress.String()
			ok := daemon.checkBgpPeerAddr(neighborAddr)
			if !ok {
				log.Errorf("Specified neighbor IP [%s] to add is in use by the local machine", neighborAddr)
			} else {
				log.Debugf("Initiating peer to neighbor at: %s", neighborAddr)
				go InitDialNeighbor(neighborAddr, acceptCh)
			}
		}

	}

	l6, err2 := listenAndAccept("tcp6", daemon.listenPort, acceptCh)
	listenerMap["tcp6"] = l6
	if err1 != nil && err2 != nil {
		log.Fatal("can't listen either v4 and v6")
		os.Exit(1)
	}
	daemon.neighborMap = make(map[string]neighborMapInfo)
	for {
		select {
		case conn := <-acceptCh:
			remoteAddr := func(addrPort string) string {
				if strings.Index(addrPort, "[") == -1 {
					return strings.Split(addrPort, ":")[0]
				}
				idx := strings.LastIndex(addrPort, ":")
				return addrPort[1 : idx-1]
			}(conn.RemoteAddr().String())
			info, found := daemon.neighborMap[remoteAddr]
			if found {
				log.Info("accepted a new connection from ", remoteAddr)
				info.neighbor.PassConn(conn)
			} else {
				log.Info("can't find configuration for a bgp neighbor from ", remoteAddr)
				conn.Close()
			}
		case neighbor := <-daemon.addedNeighborCh:
			sch := make(chan *daemonMsg, 8)
			pch := make(chan *neighborMsg, 4096)
			l := make([]*daemonMsgDataNeighbor, len(daemon.neighborMap))
			i := 0
			for _, v := range daemon.neighborMap {
				l[i] = v.neighborMsgData
				i++
			}
			p := NewNeighbor(daemon.bgpConfig.Global, neighbor, sch, pch, l)
			d := &daemonMsgDataNeighbor{
				address:       neighbor.NeighborAddress,
				neighborMsgCh: pch,
				rf:            p.neighborInfo.RF,
			}
			msg := &daemonMsg{
				msgType: SRV_MSG_PEER_ADDED,
				msgData: d,
			}
			sendServerMsgToAll(daemon.neighborMap, msg)
			daemon.neighborMap[neighbor.NeighborAddress.String()] = neighborMapInfo{
				neighbor:        p,
				daemonMsgCh:     sch,
				neighborMsgData: d,
			}
			neighborAddr := neighbor.NeighborAddress.String()
			ok := daemon.checkBgpPeerAddr(neighborAddr)
			if !ok {
				log.Debugf("Specified neighbor IP [%s] to add is bound to the local machine", neighborAddr)
			} else {
				log.Debugf("Initiating peer to neighbor at: %s", neighborAddr)
				go InitDialNeighbor(neighborAddr, acceptCh)
			}

		case neighbor := <-daemon.deletedNeighborCh:
			addr := neighbor.NeighborAddress.String()
			info, found := daemon.neighborMap[addr]
			if found {
				log.Info("Deleting peer configuration for ", addr)
				info.neighbor.Stop()
				delete(daemon.neighborMap, addr)
				msg := &daemonMsg{
					msgType: SRV_MSG_PEER_DELETED,
					msgData: info.neighbor.neighborInfo,
				}
				sendServerMsgToAll(daemon.neighborMap, msg)
			} else {
				log.Info("Can't delete a peer configuration for ", addr)
			}
		case restReq := <-daemon.RestReqCh:
			go daemon.handleRest(restReq)
		}
	}
}

func sendServerMsgToAll(neighborMap map[string]neighborMapInfo, msg *daemonMsg) {
	for _, info := range neighborMap {
		info.daemonMsgCh <- msg
	}
}

func (daemon *Daemon) SetGlobalType(g configuration.GlobalType) {
	daemon.globalTypeCh <- g
}

func (daemon *Daemon) NeighborAdd(neighbor configuration.NeighborType) {
	ok := daemon.checkBgpPeerAddr(neighbor.NeighborAddress.String())
	if !ok {
		log.Debugf("Specified neighbor IP [%s] to add is bound to the local machine", neighbor.NeighborAddress)
	} else {
		log.Debugf("Added neighbor configuration: %s", neighbor.NeighborAddress)
		daemon.addedNeighborCh <- neighbor
	}
}

func (daemon *Daemon) NeighborDelete(neighbor configuration.NeighborType) {
	log.Debugf("Deleting neighbor %s", neighbor.NeighborAddress)
	daemon.deletedNeighborCh <- neighbor
}

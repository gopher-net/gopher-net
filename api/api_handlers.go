package api

import (
	"encoding/json"
	"github.com/gopher-net/gopher-net/configuration"
	"net/http"

	log "github.com/gopher-net/gopher-net/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/gopher-net/gopher-net/Godeps/_workspace/src/github.com/docker/libchan"
	"github.com/gopher-net/gopher-net/Godeps/_workspace/src/github.com/gorilla/mux"
)

// Get a specific BGP Neighbor state
// curl -X "GET" "http://127.0.0.1:8080/v1/bgp/neighbor/<neighbor_ip>"
func (rs *RestServer) GetNeighbor(w http.ResponseWriter, r *http.Request) {
	arg := mux.Vars(r)
	remoteAddr, found := arg[NEIGHBOR_ADDR]
	if !found {
		errStr := "neighbor address is not specified"
		log.Debug(errStr)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	log.Debugf("Look up neighbor with the remote address : %v", remoteAddr)
	req := NewRestRequest(API_NEIGHBOR, remoteAddr)
	rs.bgpServerCh <- req
	res := <-req.ResponseCh
	if e := res.Err(); e != nil {
		log.Debug(e.Error())
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(res.Data)
}

// Get all BGP Neighbors
// curl -X "GET" "http://127.0.0.1:8080/v1/bgp/neighbors"
func (rs *RestServer) GetNeighbors(w http.ResponseWriter, r *http.Request) {
	req := NewRestRequest(API_NEIGHBORS, "")
	rs.bgpServerCh <- req
	res := <-req.ResponseCh
	if e := res.Err(); e != nil {
		log.Debug(e.Error())
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	log.Debugf("REST Response NEIGHBORS:  %s", res)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(res.Data)
}

// curl -i -X GET http://127.0.0.1:8080/v1/bgp/routes/local-rib
func (rs *RestServer) GetRibLocalHandler(w http.ResponseWriter, r *http.Request) {
	req := NewRestRequest(API_LOCAL_RIB, "")
	rs.bgpServerCh <- req
	res := <-req.ResponseCh
	if e := res.Err(); e != nil {
		log.Debug(e.Error())
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(res.Data)
}

//curl -i -X GET http://127.0.0.1:8080/v1/bgp/routes/adj-rib-local/172.16.86.135
func (rs *RestServer) GetNeighborLocalRib(w http.ResponseWriter, r *http.Request) {

	arg := mux.Vars(r)
	remoteAddr, found := arg[NEIGHBOR_ADDR]
	if !found {
		errStr := "neighbor address is not specified"
		log.Debug(errStr)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	log.Debugf("Look up neighbor with the remote address: %v", remoteAddr)
	req := NewRestRequest(API_ADJ_RIB_LOCAL, remoteAddr)
	rs.bgpServerCh <- req
	res := <-req.ResponseCh
	if e := res.Err(); e != nil {
		log.Debug(e.Error())
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(res.Data)
}

func (rs *RestServer) GetNeighborsConf(w http.ResponseWriter, r *http.Request) {
	req := NewRestRequest(API_CONF_NEIGHBORS, "")
	rs.bgpServerCh <- req
	res := <-req.ResponseCh
	if e := res.Err(); e != nil {
		log.Debug(e.Error())
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	log.Debugf("REST response configurations: %s", res)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(res.Data)
}

func (rs *RestServer) GetGlobalConfig(w http.ResponseWriter, r *http.Request) {
	req := NewRestRequest(API_CONF_GLOBAL, "")
	rs.bgpServerCh <- req
	res := <-req.ResponseCh
	if e := res.Err(); e != nil {
		log.Debug(e.Error())
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	log.Debugf("REST response configurations: %s", res)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(res.Data)
}

// curl -i -X GET http://72.16.86.1:8080/v1/bgp/routes
func (rs *RestServer) GetRouteTables(w http.ResponseWriter, r *http.Request) {
	req := NewRestRequest(API_ROUTES, "")
	rs.bgpServerCh <- req
	res := <-req.ResponseCh
	if e := res.Err(); e != nil {
		log.Debug(e.Error())
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	log.Debugf("REST Response routing tables: %s", res)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(res.Data)
}

// curl -i -X GET http://72.16.86.1:8080/v1/bgp/routes-out
func (rs *RestServer) GetRibOut(w http.ResponseWriter, r *http.Request) {
	req := NewRestRequest(API_RIB_OUT, "")
	rs.bgpServerCh <- req
	res := <-req.ResponseCh
	if e := res.Err(); e != nil {
		log.Debug(e.Error())
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	log.Debugf("REST Response rib-out table: %s", res)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(res.Data)
}

// curl -i -X GET http://72.16.86.1:8080/v1/bgp/routes/routes-in
func (rs *RestServer) GetRibIn(w http.ResponseWriter, r *http.Request) {
	req := NewRestRequest(API_RIB_IN, "")
	rs.bgpServerCh <- req
	res := <-req.ResponseCh
	if e := res.Err(); e != nil {
		log.Debug(e.Error())
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	log.Debugf("REST Response rib-in table: %s", res)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(res.Data)
}

/*
curl -X "POST" "http://127.0.0.1:8080/v1/bgp/routes/add" \
	-d $'{
	"ip_prefix": "6.2.6.0",
	"ip_nexthop": "172.16.86.13",
	"ip_mask": 24,
	"source_as": 7675,
	"route_family": "RF_IPv4_UC",
  	"opaque": "POLICY"
}'
*/
func (rs *RestServer) PostNewRoute(w http.ResponseWriter, r *http.Request) {
	var route RestRoute
	err := json.NewDecoder(r.Body).Decode(&route)
	if err != nil {
		http.Error(w, "HTTP decoding error", 500)
		return
	}
	req := RouteRequest(API_ADD_ROUTE, route)
	rs.bgpServerCh <- req
	res := <-req.ResponseCh
	if e := res.Err(); e != nil {
		log.Debug(e.Error())
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	log.Debugf("REST Response post new route: %s", res)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(res.Data)
}

func (rs *RestServer) PostDelRoute(w http.ResponseWriter, r *http.Request) {
	var route RestRoute
	err := json.NewDecoder(r.Body).Decode(&route)
	if err != nil {
		http.Error(w, "HTTP decoding error", 500)
		return
	}
	req := RouteRequest(API_DEL_ROUTE, route)
	rs.bgpServerCh <- req

	res := <-req.ResponseCh

	if e := res.Err(); e != nil {
		log.Debug(e.Error())
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	log.Debugf("REST Response post delete route: %s", res)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(res.Data)
}

// Add a neighbor
// curl -X POST http://127.0.0.1:8080/v1/bgp/neighbor/add -d
// '{"neighbor_as":7675,"neighbor_ip":"172.16.86.134"}'
func (rs *RestServer) PostNewNeighbor(w http.ResponseWriter, r *http.Request) {
	var config configuration.NeighborType
	err := json.NewDecoder(r.Body).Decode(&config)
	if err != nil {
		http.Error(w, "HTTP decoding error", 500)
		return
	}
	req := NodeRequest(API_ADD_NEIGHBOR, config)
	rs.bgpServerCh <- req
	res := <-req.ResponseCh
	if e := res.Err(); e != nil {
		log.Debug(e.Error())
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	log.Debugf("REST Response post new bgp neighbor: %s", res)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(res.Data)
}

// Add a neighbor
// curl -X POST http://127.0.0.1:8080/v1/bgp/neighbor -d
// '{"neighbor_as":7675,"neighbor_ip":"172.16.86.134"}'
func (rs *RestServer) PostDelNeighbor(w http.ResponseWriter, r *http.Request) {
	var config configuration.NeighborType
	err := json.NewDecoder(r.Body).Decode(&config)
	if err != nil {
		http.Error(w, "HTTP decoding error", 500)
		return
	}
	req := NodeRequest(API_DEL_NEIGHBOR, config)
	rs.bgpServerCh <- req
	res := <-req.ResponseCh
	if e := res.Err(); e != nil {
		log.Debug(e.Error())
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	log.Debugf("REST Response delete new neighbor: %s", res)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(res.Data)
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

// TODO move below, cyclic deps
type RemoteCommand struct {
	Cmd        string
	Args       []string
	StatusChan libchan.Sender
}

type CommandResponse struct {
	Data   interface{}
	Status int
}

// Commented to avoid cyclic dep
//func ShowRoutes(daemon *daemon.Daemon) (int, interface{}) {
//	result := daemon.BgpDb
//	return 0, result
//}

func AdvertiseRoute() int {
	log.Print("advertsing")
	return 0
}

func WithdrawRoute() int {
	log.Print("withdrawing")
	return 0
}

func AddPeer() int {
	log.Print("adding peer")
	return 0
}

func RemovePeer() int {
	log.Print("remove peer")
	return 0
}

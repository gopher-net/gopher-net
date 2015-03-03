package api

import (
	"github.com/gopher-net/gopher-net/configuration"
	"net/http"
	"strconv"

	"github.com/gopher-net/gopher-net/Godeps/_workspace/src/github.com/gorilla/mux"
)

const (
	_ = iota
	API_NEIGHBOR
	API_NEIGHBORS
	API_LOCAL_RIB
	API_ADJ_RIB_LOCAL
	API_NEIGHBOR_ADD
	API_ROUTES
	API_ADD_ROUTE
	API_DEL_ROUTE
	API_ADD_NEIGHBOR
	API_CONF_GLOBAL
	API_CONF_NEIGHBORS
	API_DEL_NEIGHBOR
	API_RIB_IN
	API_RIB_OUT
	API_NEIGHBOR_SHUTDOWN
	API_NEIGHBOR_RESET
	API_NEIGHBOR_SOFT_RESET
	API_NEIGHBOR_SOFT_RESET_IN
	API_NEIGHBOR_SOFT_RESET_OUT
)

const (
	BASE_VERSION       = "/v1"
	ROUTES             = "/bgp/routes"
	ADJ_RIB_LOCAL      = "/adj-rib-local"
	RIB_LOCAL          = "/local-rib"
	NEIGHBOR_ADDR      = "remotePeerAddr"
	REMOTE_AS_ARG      = "remoteAS"
	REMOTE_NEIGHBOR_AS = "/neighbor-as"
	GLOBAL_CONF        = "/bgp/conf/global"
	NEIGHBORS_CONF     = "/bgp/conf/neighbors"
	ADD                = "/add"
	DEL                = "/delete"
	RIB_OUT_PREFIX     = "/routes-out"
	RIB_IN_PREFIX      = "/routes-in"
	NEIGHBOR_PREFIX    = "/bgp/neighbor"
	NEIGHBORS_PREFIX   = "/bgp/neighbors"
	NEIGHBOR           = BASE_VERSION + NEIGHBOR_PREFIX
	NEIGHBORS          = BASE_VERSION + NEIGHBORS_PREFIX
	ROUTE_TABLES       = BASE_VERSION + ROUTES
	GLOBAL_CONFIG      = BASE_VERSION + GLOBAL_CONF
	NEIGHBORS_CONFIG   = BASE_VERSION + NEIGHBORS_CONF
	RIB_IN             = ROUTE_TABLES + RIB_IN_PREFIX
	RIB_OUT            = ROUTE_TABLES + RIB_OUT_PREFIX
	REST_PORT          = 8080
)

type RestRequest struct {
	RequestType int
	RemoteAddr  string
	ResponseCh  chan *RestResponse
	NodeConfig  configuration.NeighborType
	RestRoute   RestRoute
	Err         error
}

type RestResponse struct {
	ResponseErr error
	Data        []byte
}

type RestServer struct {
	port        int
	bgpServerCh chan *RestRequest
}

type RestResponseDefault struct {
	ResponseErr error
}

type RestResponseNeighbors struct {
	RestResponseDefault
	Neighbors []string
}

// Response struct for Neighbor
type RestResponseNeighbor struct {
	RestResponseDefault
	RemoteAddr    string
	RemoteAs      uint32
	NeighborState string
	UpdateCount   int
}

// Response struct for Rib
type RestResponseRib struct {
	RestResponseDefault
	RemoteAddr string
	RemoteAs   uint32
	RibInfo    []string
}

func (r *RestResponse) Err() error {
	return r.ResponseErr
}

func NewRestServer(port int, bgpServerCh chan *RestRequest) *RestServer {
	rs := &RestServer{
		port:        port,
		bgpServerCh: bgpServerCh,
	}
	return rs
}

func NodeRequest(reqType int, config configuration.NeighborType) *RestRequest {
	r := &RestRequest{
		RequestType: reqType,
		NodeConfig:  config,
		ResponseCh:  make(chan *RestResponse),
	}
	return r
}

func RouteRequest(reqType int, route RestRoute) *RestRequest {
	r := &RestRequest{
		RequestType: reqType,
		RestRoute:   route,
		ResponseCh:  make(chan *RestResponse),
	}
	return r
}

func NewRestRequest(reqType int, remoteAddr string) *RestRequest {
	r := &RestRequest{
		RequestType: reqType,
		RemoteAddr:  remoteAddr,
		ResponseCh:  make(chan *RestResponse),
	}
	return r
}

type RestRoute struct {
	IpPrefix    string `json:"ip_prefix"`
	PrefixMask  uint8  `json:"ip_mask"`
	NextHop     string `json:"ip_nexthop"`
	AS          uint32 `json:"source_as"`
	LocalPref   uint32 `json:"local_pref"`
	RF          string `json:"route_family"`
	ExCommunity string `json:"opaque"`
}

func (rs *RestServer) Serve() {

	r := mux.NewRouter()
	// add/delete/get routes
	r.HandleFunc(ROUTE_TABLES+ADJ_RIB_LOCAL+"/{"+NEIGHBOR_ADDR+"}", rs.GetNeighborLocalRib).Methods("GET")
	r.HandleFunc(ROUTE_TABLES+RIB_LOCAL, rs.GetRibLocalHandler).Methods("GET")
	r.HandleFunc(ROUTE_TABLES, rs.GetRouteTables).Methods("GET")
	r.HandleFunc(ROUTE_TABLES+RIB_OUT_PREFIX, rs.GetRibOut).Methods("GET")
	r.HandleFunc(ROUTE_TABLES+RIB_IN_PREFIX, rs.GetRibIn).Methods("GET")
	r.HandleFunc(ROUTE_TABLES+ADD, rs.PostNewRoute).Methods("POST")
	r.HandleFunc(ROUTE_TABLES+DEL, rs.PostDelRoute).Methods("POST")

	// add/delete/get neighbors
	r.HandleFunc(NEIGHBOR+"/{"+NEIGHBOR_ADDR+"}", rs.GetNeighbor).Methods("GET")
	r.HandleFunc(NEIGHBORS, rs.GetNeighbors).Methods("GET")
	r.HandleFunc(NEIGHBOR+ADD, rs.PostNewNeighbor).Methods("POST")
	r.HandleFunc(NEIGHBOR+DEL, rs.PostDelNeighbor).Methods("POST")

	// Get node and global configuration
	r.HandleFunc(GLOBAL_CONFIG, rs.GetGlobalConfig).Methods("GET")
	r.HandleFunc(NEIGHBORS_CONFIG, rs.GetNeighborsConf).Methods("GET")

	// handle 404
	r.NotFoundHandler = http.HandlerFunc(NotFoundHandler)
	http.Handle("/", r)
	http.ListenAndServe(":"+strconv.Itoa(rs.port), nil)
}

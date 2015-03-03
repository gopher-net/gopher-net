
gopher-net
==========

A collection of routing daemons written in Go.

## Configure and Run

The packet library for the bgp portion uses the great work from the osrg monitoring bgp implementation along with the config options. This enables speaker initiation and ephemeral route caching. Still needs lots of work, feel free to run with any pieces of it. Also need a Dijkstra algorithm to be implemented (e.g. OSPF, ISIS best path selection).

This project isn't affiliated w/ any company, just network nerds hacking on prototypes using routing protocols in Go.

You can either define peers using the sample configuration in the root ./bgpd.conf or add the neighbor using the API listed below.

Run w/ sudo privileges or as root in order to bind to 179

	sudo -E go run main.go -f bgpd.conf

To run with the logs set to debug use:

    sudo -E go run main.go -f bgpd.conf -l debug

You can point two bgpd.go processes at one another or run against Quagga.

## Example API usage

A Postman import is located in the ./api directory.

##### Get Neighbor State Info

    curl -i -X GET http://127.0.0.1:8080/v1/bgp/neighbor/172.16.86.134

##### Get all Remote Routes (Destinations)

    curl -i -X GET http://172.16.86.135:8080/v1/bgp/routes

##### Get all Remote Routes (Destinations)

    curl -i -X GET http://127.0.0.1:8080/v1/bgp/routes

##### Get Local_RIB from a Peer (Destinations)

    curl -i -X GET http://127.0.0.1:8080/v1/bgp/neighbor/172.16.86.134/local-rib

##### Get Remote Destination Prefixes

    curl -i -X GET http://127.0.0.1:8080/v1/bgp/routes/local-rib

##### Get RIB_IN  (Incoming Routes)

    curl -i -X GET http://172.16.86.1:8080/v1/bgp/routes/routes-in

##### Get RIB_OUT (Outgoing Routes)

    curl -i -X GET http://127.0.0.1:8080/v1/bgp/routes/routes-out

##### Add a Route

    curl -X "POST" "http://172.16.86.1:8080/v1/bgp/routes/add" \
	    -d $'{
	    "ip_prefix": "1.1.1.1",
	    "ip_nexthop": "172.16.86.13",
	    "ip_mask": 32,
	    "source_as": 7675,
	    "route_family": "RF_IPv4_UC"
    }'

##### Add a Route

    curl -X "POST" "http://172.16.86.135:8080/v1/bgp/routes/add" \
	    -d $'{
	    "ip_prefix": "2.2.2.2",
	    "ip_nexthop": "172.16.86.13",
	    "ip_mask": 32,
	    "source_as": 7675,
	    "route_family": "RF_IPv4_UC"
    }'


##### Add a Route with an Opaque Community


    curl -X "POST" "http://172.16.86.135:8080/v1/bgp/routes/add" \
	    -d $'{
	    "ip_prefix": "2.2.2.2",
	    "ip_nexthop": "172.16.86.13",
	    "ip_mask": 32,
	    "source_as": 7675,
	    "extended_community": "POLICY"
    }'

##### Add add a neighbor

There is a race condition bug here that will blow up the FSM in some cases :)

    curl -X "POST" "http://127.0.0.1:8080/v1/bgp/neighbor/add" \
	    -d $'{
    "neighbor_as":7675,
    "neighbor_ip":"172.16.86.135"
    }'

##### Add a Neighbor and a Route 


    curl -X "POST" "http://127.0.0.1:8080/v1/bgp/neighbor/add" \
	    -d $'{
    "neighbor_as":7675,
    "neighbor_ip":"172.16.86.135"
    }'
    
    curl -X "POST" "http://172.16.86.1:8080/v1/bgp/routes/add" \
	    -d $'{
	    "ip_prefix": "2.2.2.2",
	    "ip_nexthop": "172.16.86.13",
	    "ip_mask": 32,
	    "source_as": 7675,
	    "route_family": "RF_IPv4_UC"
    }'


## BGP Prefix Update Events and BGP Node Events

These callbacks are located in bgp_event_callbacks.go as an example of how to get notified of a new prefix (or MAC if we get around to evpn etc) or even opaque community strings eventually would be cool. This could be a new VM, new container or anything else being advertised in BGP updates. Would ideally be migrated to an interface watch pub/sub as for a more decoupled update notification.

A New Node/Neighbor goes into an Established BGP FSM state. This means its ready to send and receive prefix (container updates).

    INFO[0001] Container Event: New Neighbor Added
    INFO[0001] Container Event: Established Neighbor IP address -> [ 7675 ]
    INFO[0001] Container Event: Established Neighbor IP address -> [ 172.16.86.134 ]
    INFO[0001] Container Event: Established Neighbor FSM State -> [ BGP_FSM_ESTABLISHED ]

An existing Node/Neighbor is no longer in an Established BGP FSM state. This means it is no longer receiving prefix (container) updates.

    INFO[0163] Container Event: Neighbor Removed (FSM Idle)
    INFO[0163] Container Event: Idle Neighbor IP address -> [ 7675 ]
    INFO[0163] Container Event: Neighbor Removed Idle Neighbor IP address -> [ [ 172.16.86.134 ] ]


New Prefix/Containers are updated. A forwarding entry needs to be added to a TEP for the new Prefix using the route.GetNexthop() value as the destination TEP.

    INFO[0002] Container Event: Route Added Notification
    INFO[0002] Container Event: Prefix Added: -> [ 10.201.108.0/24 ]
    INFO[0002] Container Event: Prefix Nexthop: -> [ 172.16.86.134 ]
    INFO[0002] Container Event: All NLRI:
    [
        {
            "Network": "10.201.108.0/24",
            "Nexthop": "172.16.86.134",
            "Attrs": [
            {
                "Type": "BGP_ATTR_TYPE_ORIGIN",
                "Value": 0
            },
            {
                "Type": "BGP_ATTR_TYPE_AS_PATH",
                "AsPath": [
                ]
            },
            {
                "Type": "BGP_ATTR_TYPE_NEXT_HOP",
                "Nexthop": "172.16.86.134"
            },
            {
                "Type": "BGP_ATTR_TYPE_MULTI_EXIT_DISC",
                "Metric": 0
            },
            {
                "Type": "BGP_ATTR_TYPE_LOCAL_PREF",
                "Pref": 100
            }
            ],
            "Best": "false"
        }
    ]

Existing Prefix/Containers are Withdrawn/Removed from the BGP tables. Existing entries needs to be removed for the Prefix

    INFO[0065] Container Event: Route Added Notification
    INFO[0065] Container Event: Prefix Added: -> [ 10.205.222.5/32 ]
    INFO[0065] Container Event: Prefix Nexthop: -> [ <nil> ]
    INFO[0065] Container Event: All NLRI:
    [
    	{
        "Network": "10.205.222.5/32",
        "Nexthop": "\u003cnil\u003e",
        "Attrs": null,
        "Best": "false"
    	}
    ]

## Running the tests

To run only unit tests:

    make test

## Dependency Management

We use [godep](https://github.com/tools/godep) for dependency management with rewritten import paths.
This allows the repo to be `go get`able.

To bump the version of a dependency, follow these [instructions](https://github.com/tools/godep#update-a-dependency)

// Copyright (C) 2014 Nippon Telegraph and Telephone Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bgp

import "fmt"

const AS_TRANS = 23456

const BGP_PORT = 179

type FSMState int

const (
	_ FSMState = iota
	BGP_FSM_IDLE
	BGP_FSM_CONNECT
	BGP_FSM_ACTIVE
	BGP_FSM_OPENSENT
	BGP_FSM_OPENCONFIRM
	BGP_FSM_ESTABLISHED
)

const (
	_BGPAttrType_name_0 = "BGP_ATTR_TYPE_ORIGINBGP_ATTR_TYPE_AS_PATHBGP_ATTR_TYPE_NEXT_HOPBGP_ATTR_TYPE_MULTI_EXIT_DISCBGP_ATTR_TYPE_LOCAL_PREFBGP_ATTR_TYPE_ATOMIC_AGGREGATEBGP_ATTR_TYPE_AGGREGATORBGP_ATTR_TYPE_COMMUNITIESBGP_ATTR_TYPE_ORIGINATOR_IDBGP_ATTR_TYPE_CLUSTER_LIST"
	_BGPAttrType_name_1 = "BGP_ATTR_TYPE_MP_REACH_NLRIBGP_ATTR_TYPE_MP_UNREACH_NLRIBGP_ATTR_TYPE_EXTENDED_COMMUNITIESBGP_ATTR_TYPE_AS4_PATHBGP_ATTR_TYPE_AS4_AGGREGATOR"
)

var (
	_BGPAttrType_index_0 = [...]uint8{0, 20, 41, 63, 92, 116, 146, 170, 195, 222, 248}
	_BGPAttrType_index_1 = [...]uint8{0, 27, 56, 90, 112, 140}
)

func (i BGPAttrType) String() string {
	switch {
	case 1 <= i && i <= 10:
		i -= 1
		return _BGPAttrType_name_0[_BGPAttrType_index_0[i]:_BGPAttrType_index_0[i+1]]
	case 14 <= i && i <= 18:
		i -= 14
		return _BGPAttrType_name_1[_BGPAttrType_index_1[i]:_BGPAttrType_index_1[i+1]]
	default:
		return fmt.Sprintf("BGPAttrType(%d)", i)
	}
}

const _FSMState_name = "BGP_FSM_IDLEBGP_FSM_CONNECTBGP_FSM_ACTIVEBGP_FSM_OPENSENTBGP_FSM_OPENCONFIRMBGP_FSM_ESTABLISHED"

var _FSMState_index = [...]uint8{0, 12, 27, 41, 57, 76, 95}

func (i FSMState) String() string {
	i -= 1
	if i < 0 || i+1 >= FSMState(len(_FSMState_index)) {
		return fmt.Sprintf("FSMState(%d)", i+1)
	}
	return _FSMState_name[_FSMState_index[i]:_FSMState_index[i+1]]
}

const (
	_RouteFamily_name_0 = "RF_IPv4_UC"
	_RouteFamily_name_1 = "RF_IPv4_MPLS"
	_RouteFamily_name_2 = "RF_IPv4_VPN"
	_RouteFamily_name_3 = "RF_RTC_UC"
	_RouteFamily_name_4 = "RF_IPv6_UC"
	_RouteFamily_name_5 = "RF_IPv6_MPLS"
	_RouteFamily_name_6 = "RF_IPv6_VPN"
)

var (
	_RouteFamily_index_0 = [...]uint8{0, 10}
	_RouteFamily_index_1 = [...]uint8{0, 12}
	_RouteFamily_index_2 = [...]uint8{0, 11}
	_RouteFamily_index_3 = [...]uint8{0, 9}
	_RouteFamily_index_4 = [...]uint8{0, 10}
	_RouteFamily_index_5 = [...]uint8{0, 12}
	_RouteFamily_index_6 = [...]uint8{0, 11}
)

func (i RouteFamily) String() string {
	switch {
	case i == 65537:
		return _RouteFamily_name_0
	case i == 65540:
		return _RouteFamily_name_1
	case i == 65664:
		return _RouteFamily_name_2
	case i == 65668:
		return _RouteFamily_name_3
	case i == 131073:
		return _RouteFamily_name_4
	case i == 131076:
		return _RouteFamily_name_5
	case i == 131200:
		return _RouteFamily_name_6
	default:
		return fmt.Sprintf("RouteFamily(%d)", i)
	}
}

const (
	_BGPCapabilityCode_name_0 = "BGP_CAP_MULTIPROTOCOLBGP_CAP_ROUTE_REFRESH"
	_BGPCapabilityCode_name_1 = "BGP_CAP_CARRYING_LABEL_INFO"
	_BGPCapabilityCode_name_2 = "BGP_CAP_GRACEFUL_RESTARTBGP_CAP_FOUR_OCTET_AS_NUMBER"
	_BGPCapabilityCode_name_3 = "BGP_CAP_ENHANCED_ROUTE_REFRESH"
	_BGPCapabilityCode_name_4 = "BGP_CAP_ROUTE_REFRESH_CISCO"
)

var (
	_BGPCapabilityCode_index_0 = [...]uint8{0, 21, 42}
	_BGPCapabilityCode_index_1 = [...]uint8{0, 27}
	_BGPCapabilityCode_index_2 = [...]uint8{0, 24, 52}
	_BGPCapabilityCode_index_3 = [...]uint8{0, 30}
	_BGPCapabilityCode_index_4 = [...]uint8{0, 27}
)

func (i BGPCapabilityCode) String() string {
	switch {
	case 1 <= i && i <= 2:
		i -= 1
		return _BGPCapabilityCode_name_0[_BGPCapabilityCode_index_0[i]:_BGPCapabilityCode_index_0[i+1]]
	case i == 4:
		return _BGPCapabilityCode_name_1
	case 64 <= i && i <= 65:
		i -= 64
		return _BGPCapabilityCode_name_2[_BGPCapabilityCode_index_2[i]:_BGPCapabilityCode_index_2[i+1]]
	case i == 70:
		return _BGPCapabilityCode_name_3
	case i == 128:
		return _BGPCapabilityCode_name_4
	default:
		return fmt.Sprintf("BGPCapabilityCode(%d)", i)
	}
}

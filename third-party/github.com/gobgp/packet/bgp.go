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

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"reflect"
)

// move somewhere else

const (
	AFI_IP  = 1
	AFI_IP6 = 2
)

const (
	SAFI_UNICAST                  = 1
	SAFI_MULTICAST                = 2
	SAFI_MPLS_LABEL               = 4
	SAFI_MPLS_VPN                 = 128
	SAFI_ROUTE_TARGET_CONSTRTAINS = 132
)

const (
	_ = iota
	BGP_MSG_OPEN
	BGP_MSG_UPDATE
	BGP_MSG_NOTIFICATION
	BGP_MSG_KEEPALIVE
	BGP_MSG_ROUTE_REFRESH
)

const (
	BGP_OPT_CAPABILITY = 2
)

type BGPCapabilityCode uint8

const (
	BGP_CAP_MULTIPROTOCOL          BGPCapabilityCode = 1
	BGP_CAP_ROUTE_REFRESH          BGPCapabilityCode = 2
	BGP_CAP_CARRYING_LABEL_INFO    BGPCapabilityCode = 4
	BGP_CAP_GRACEFUL_RESTART       BGPCapabilityCode = 64
	BGP_CAP_FOUR_OCTET_AS_NUMBER   BGPCapabilityCode = 65
	BGP_CAP_ENHANCED_ROUTE_REFRESH BGPCapabilityCode = 70
	BGP_CAP_ROUTE_REFRESH_CISCO    BGPCapabilityCode = 128
)

type ParameterCapabilityInterface interface {
	DecodeFromBytes([]byte) error
	Serialize() ([]byte, error)
	Len() int
	Code() BGPCapabilityCode
}

type DefaultParameterCapability struct {
	CapCode  BGPCapabilityCode
	CapLen   uint8
	CapValue []byte
}

func (c *DefaultParameterCapability) Code() BGPCapabilityCode {
	return c.CapCode
}

func (c *DefaultParameterCapability) DecodeFromBytes(data []byte) error {
	c.CapCode = BGPCapabilityCode(data[0])
	c.CapLen = data[1]
	if uint8(len(data)) < 2+c.CapLen {
		return fmt.Errorf("Not all OptionParameterCapability bytes available")
	}
	c.CapValue = data[2 : 2+c.CapLen]
	return nil
}

func (c *DefaultParameterCapability) Serialize() ([]byte, error) {
	c.CapLen = uint8(len(c.CapValue))
	buf := make([]byte, 2)
	buf[0] = uint8(c.CapCode)
	buf[1] = c.CapLen
	buf = append(buf, c.CapValue...)
	return buf, nil
}

func (c *DefaultParameterCapability) Len() int {
	return int(c.CapLen + 2)
}

type CapMultiProtocolValue struct {
	AFI  uint16
	SAFI uint8
}

type CapMultiProtocol struct {
	DefaultParameterCapability
	CapValue CapMultiProtocolValue
}

func (c *CapMultiProtocol) DecodeFromBytes(data []byte) error {
	c.DefaultParameterCapability.DecodeFromBytes(data)
	data = data[2:]
	if len(data) < 4 {
		return fmt.Errorf("Not all CapabilityMultiProtocol bytes available")
	}
	c.CapValue.AFI = binary.BigEndian.Uint16(data[0:2])
	c.CapValue.SAFI = data[3]
	return nil
}

func (c *CapMultiProtocol) Serialize() ([]byte, error) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint16(buf[0:], c.CapValue.AFI)
	buf[3] = c.CapValue.SAFI
	c.DefaultParameterCapability.CapValue = buf
	return c.DefaultParameterCapability.Serialize()
}

func NewCapMultiProtocol(afi uint16, safi uint8) *CapMultiProtocol {
	return &CapMultiProtocol{
		DefaultParameterCapability{
			CapCode: BGP_CAP_MULTIPROTOCOL,
		},
		CapMultiProtocolValue{
			AFI:  afi,
			SAFI: safi,
		},
	}
}

type CapRouteRefresh struct {
	DefaultParameterCapability
}

func NewCapRouteRefresh() *CapRouteRefresh {
	return &CapRouteRefresh{
		DefaultParameterCapability{
			CapCode: BGP_CAP_ROUTE_REFRESH,
		},
	}
}

type CapCarryingLabelInfo struct {
	DefaultParameterCapability
}

type CapGracefulRestartTuples struct {
	AFI   uint16
	SAFI  uint8
	Flags uint8
}

type CapGracefulRestartValue struct {
	Flags  uint8
	Time   uint16
	Tuples []CapGracefulRestartTuples
}

type CapGracefulRestart struct {
	DefaultParameterCapability
	CapValue CapGracefulRestartValue
}

func (c *CapGracefulRestart) DecodeFromBytes(data []byte) error {
	c.DefaultParameterCapability.DecodeFromBytes(data)
	data = data[2:]
	restart := binary.BigEndian.Uint16(data[0:2])
	c.CapValue.Flags = uint8(restart >> 12)
	c.CapValue.Time = restart & 0xfff
	data = data[2:]
	for len(data) >= 4 {
		t := CapGracefulRestartTuples{binary.BigEndian.Uint16(data[0:2]),
			data[2], data[3]}
		c.CapValue.Tuples = append(c.CapValue.Tuples, t)
		data = data[4:]
	}
	return nil
}

func (c *CapGracefulRestart) Serialize() ([]byte, error) {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf[0:], uint16(c.CapValue.Flags)<<12|c.CapValue.Time)
	for _, t := range c.CapValue.Tuples {
		tbuf := make([]byte, 4)
		binary.BigEndian.PutUint16(tbuf[0:2], t.AFI)
		tbuf[2] = t.SAFI
		tbuf[3] = t.Flags
		buf = append(buf, tbuf...)
	}
	c.DefaultParameterCapability.CapValue = buf
	return c.DefaultParameterCapability.Serialize()
}

func NewCapGracefulRestart(flags uint8, time uint16, tuples []CapGracefulRestartTuples) *CapGracefulRestart {
	return &CapGracefulRestart{
		DefaultParameterCapability{
			CapCode: BGP_CAP_GRACEFUL_RESTART,
		},
		CapGracefulRestartValue{
			flags,
			time,
			tuples,
		},
	}
}

type CapFourOctetASNumber struct {
	DefaultParameterCapability
	CapValue uint32
}

func (c *CapFourOctetASNumber) DecodeFromBytes(data []byte) error {
	c.DefaultParameterCapability.DecodeFromBytes(data)
	data = data[2:]
	if len(data) < 4 {
		return fmt.Errorf("Not all CapabilityMultiProtocol bytes available")
	}
	c.CapValue = binary.BigEndian.Uint32(data[0:4])
	return nil
}

func (c *CapFourOctetASNumber) Serialize() ([]byte, error) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, c.CapValue)
	c.DefaultParameterCapability.CapValue = buf
	return c.DefaultParameterCapability.Serialize()
}

func NewCapFourOctetASNumber(asnum uint32) *CapFourOctetASNumber {
	return &CapFourOctetASNumber{
		DefaultParameterCapability{
			CapCode: BGP_CAP_FOUR_OCTET_AS_NUMBER,
		},
		asnum,
	}
}

type CapEnhancedRouteRefresh struct {
	DefaultParameterCapability
}

func NewCapEnhancedRouteRefresh() *CapEnhancedRouteRefresh {
	return &CapEnhancedRouteRefresh{
		DefaultParameterCapability{
			CapCode: BGP_CAP_ENHANCED_ROUTE_REFRESH,
		},
	}
}

type CapRouteRefreshCisco struct {
	DefaultParameterCapability
}

func NewCapRouteRefreshCisco() *CapRouteRefreshCisco {
	return &CapRouteRefreshCisco{
		DefaultParameterCapability{
			CapCode: BGP_CAP_ROUTE_REFRESH_CISCO,
		},
	}
}

type CapUnknown struct {
	DefaultParameterCapability
}

type OptionParameterInterface interface {
	Serialize() ([]byte, error)
}

type OptionParameterCapability struct {
	ParamType  uint8
	ParamLen   uint8
	Capability []ParameterCapabilityInterface
}

func (o *OptionParameterCapability) DecodeFromBytes(data []byte) error {
	if uint8(len(data)) < o.ParamLen {
		return fmt.Errorf("Not all OptionParameterCapability bytes available")
	}
	for len(data) >= 2 {
		var c ParameterCapabilityInterface
		switch BGPCapabilityCode(data[0]) {
		case BGP_CAP_MULTIPROTOCOL:
			c = &CapMultiProtocol{}
		case BGP_CAP_ROUTE_REFRESH:
			c = &CapRouteRefresh{}
		case BGP_CAP_CARRYING_LABEL_INFO:
			c = &CapCarryingLabelInfo{}
		case BGP_CAP_GRACEFUL_RESTART:
			c = &CapGracefulRestart{}
		case BGP_CAP_FOUR_OCTET_AS_NUMBER:
			c = &CapFourOctetASNumber{}
		case BGP_CAP_ENHANCED_ROUTE_REFRESH:
			c = &CapEnhancedRouteRefresh{}
		case BGP_CAP_ROUTE_REFRESH_CISCO:
			c = &CapRouteRefreshCisco{}
		default:
			c = &CapUnknown{}
		}
		err := c.DecodeFromBytes(data)
		if err != nil {
			return nil
		}
		o.Capability = append(o.Capability, c)
		data = data[c.Len():]
	}
	return nil
}

func (o *OptionParameterCapability) Serialize() ([]byte, error) {
	buf := make([]byte, 2)
	buf[0] = o.ParamType
	//buf[1] = o.ParamLen
	for _, p := range o.Capability {
		pbuf, err := p.Serialize()
		if err != nil {
			return nil, err
		}
		buf = append(buf, pbuf...)
	}
	buf[1] = uint8(len(buf) - 2)
	return buf, nil
}

func NewOptionParameterCapability(capability []ParameterCapabilityInterface) *OptionParameterCapability {
	return &OptionParameterCapability{
		ParamType:  BGP_OPT_CAPABILITY,
		Capability: capability,
	}
}

type OptionParameterUnknown struct {
	ParamType uint8
	ParamLen  uint8
	Value     []byte
}

func (o *OptionParameterUnknown) Serialize() ([]byte, error) {
	buf := make([]byte, 2)
	buf[0] = o.ParamType
	if o.ParamLen == 0 {
		o.ParamLen = uint8(len(o.Value))
	}
	buf[1] = o.ParamLen
	return append(buf, o.Value...), nil
}

type BGPOpen struct {
	Version     uint8
	MyAS        uint16
	HoldTime    uint16
	ID          net.IP
	OptParamLen uint8
	OptParams   []OptionParameterInterface
}

func (msg *BGPOpen) DecodeFromBytes(data []byte) error {
	msg.Version = data[0]
	msg.MyAS = binary.BigEndian.Uint16(data[1:3])
	msg.HoldTime = binary.BigEndian.Uint16(data[3:5])
	msg.ID = data[5:9]
	msg.OptParamLen = data[9]
	data = data[10:]
	if uint8(len(data)) < msg.OptParamLen {
		return fmt.Errorf("Not all BGP Open message bytes available")
	}

	for rest := msg.OptParamLen; rest > 0; {
		paramtype := data[0]
		paramlen := data[1]
		rest -= paramlen + 2

		if paramtype == BGP_OPT_CAPABILITY {
			p := &OptionParameterCapability{}
			p.ParamType = paramtype
			p.ParamLen = paramlen
			p.DecodeFromBytes(data[2 : 2+paramlen])
			msg.OptParams = append(msg.OptParams, p)
		} else {
			p := &OptionParameterUnknown{}
			p.ParamType = paramtype
			p.ParamLen = paramlen
			p.Value = data[2 : 2+paramlen]
			msg.OptParams = append(msg.OptParams, p)
		}
		data = data[2+paramlen:]
	}
	return nil
}

func (msg *BGPOpen) Serialize() ([]byte, error) {
	buf := make([]byte, 10)
	buf[0] = msg.Version
	binary.BigEndian.PutUint16(buf[1:3], msg.MyAS)
	binary.BigEndian.PutUint16(buf[3:5], msg.HoldTime)
	copy(buf[5:9], msg.ID)
	pbuf := make([]byte, 0)
	for _, p := range msg.OptParams {
		onepbuf, err := p.Serialize()
		if err != nil {
			return nil, err
		}
		pbuf = append(pbuf, onepbuf...)
	}
	buf[9] = uint8(len(pbuf))
	return append(buf, pbuf...), nil
}

func NewBGPOpenMessage(myas uint16, holdtime uint16, id string, optparams []OptionParameterInterface) *BGPMessage {
	return &BGPMessage{
		Header: BGPHeader{Type: BGP_MSG_OPEN},
		Body:   &BGPOpen{4, myas, holdtime, net.ParseIP(id).To4(), 0, optparams},
	}
}

type AddrPrefixInterface interface {
	DecodeFromBytes([]byte) error
	Serialize() ([]byte, error)
	AFI() uint16
	SAFI() uint8
	Len() int
}

type IPAddrPrefixDefault struct {
	Length uint8
	Prefix net.IP
}

func (r *IPAddrPrefixDefault) decodePrefix(data []byte, bitlen uint8, addrlen uint8) error {
	bytelen := (bitlen + 7) / 8
	if len(data) < int(bytelen) {
		eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
		eSubCode := uint8(BGP_ERROR_SUB_MALFORMED_ATTRIBUTE_LIST)
		return NewMessageError(eCode, eSubCode, nil, "network bytes is short")
	}
	b := make([]byte, addrlen)
	copy(b, data[:bytelen])
	r.Prefix = b
	return nil
}

//func (r *IPAddrPrefixDefault) serializePrefix(bitlen uint8) ([]byte, error) {
//    bytelen := (bitlen + 7) / 8
//    buf := make([]byte, bytelen)
//    copy(buf, r.Prefix)
//    // clear trailing bits in the last byte. rfc doesn't require
//    // this though.
//    if bitlen%8 != 0 {
//        mask := 0xff00 >> (bitlen % 8)
//        last_byte_value := buf[bytelen-1] & byte(mask)
//        buf[bytelen-1] = last_byte_value
//    }
//    return buf, nil
//}

// New
//func (r *IPAddrPrefixDefault) serializePrefix(bitlen uint8) ([]byte, error) {
//    bytelen := (bitlen + 7) / 8
//    buf := make([]byte, bytelen)
//    copy(buf, r.Prefix)
//    // clear trailing bits in the last byte. rfc doesn't require
//    // this though.
//    if bitlen%8 != 0 {
//        mask := 0xff00 >> (bitlen % 8)
//        last_byte_value := buf[bytelen-1] & byte(mask)
//        buf[bytelen-1] = last_byte_value
//    }
//    return buf, nil
//}

// Major bug Above produces NLRI updates of 0.0.0.0 -Below resolves the issue
func (r *IPAddrPrefixDefault) serializePrefix(bitlen uint8) ([]byte, error) {
	bytelen := (bitlen + 7) / 8
	buf := make([]byte, bytelen)
	copy(buf, r.Prefix.To4())
	return buf, nil
}

func (r *IPAddrPrefixDefault) Len() int {
	return int(1 + ((r.Length + 7) / 8))
}

func (r *IPAddrPrefixDefault) String() string {
	return fmt.Sprintf("%s/%d", r.Prefix.String(), r.Length)
}

type IPAddrPrefix struct {
	IPAddrPrefixDefault
	addrlen uint8
}

func (r *IPAddrPrefix) DecodeFromBytes(data []byte) error {
	if len(data) < 1 {
		eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
		eSubCode := uint8(BGP_ERROR_SUB_MALFORMED_ATTRIBUTE_LIST)
		return NewMessageError(eCode, eSubCode, nil, "prefix misses length field")
	}
	r.Length = data[0]
	if r.addrlen == 0 {
		r.addrlen = 4
	}
	return r.decodePrefix(data[1:], r.Length, r.addrlen)
}

func (r *IPAddrPrefix) Serialize() ([]byte, error) {
	buf := make([]byte, 1)
	buf[0] = r.Length
	pbuf, err := r.serializePrefix(r.Length)
	if err != nil {
		return nil, err
	}
	return append(buf, pbuf...), nil
}

func (r *IPAddrPrefix) AFI() uint16 {
	return AFI_IP
}

func (r *IPAddrPrefix) SAFI() uint8 {
	return SAFI_UNICAST
}

type IPv6AddrPrefix struct {
	IPAddrPrefix
}

func NewIPAddrPrefix(length uint8, prefix string) *IPAddrPrefix {
	return &IPAddrPrefix{
		IPAddrPrefixDefault{length, net.ParseIP(prefix)},
		4,
	}
}

func (r *IPv6AddrPrefix) AFI() uint16 {
	return AFI_IP6
}

func NewIPv6AddrPrefix(length uint8, prefix string) *IPv6AddrPrefix {
	return &IPv6AddrPrefix{
		IPAddrPrefix{
			IPAddrPrefixDefault{length, net.ParseIP(prefix)},
			16,
		},
	}
}

type WithdrawnRoute struct {
	IPAddrPrefix
}

const (
	BGP_RD_TWO_OCTET_AS = iota
	BGP_RD_IPV4_ADDRESS
	BGP_RD_FOUR_OCTET_AS
)

type RouteDistinguisherInterface interface {
	DecodeFromBytes([]byte) error
	Serialize() ([]byte, error)
	Len() int
}

type DefaultRouteDistinguisher struct {
	Type  uint16
	Value []byte
}

func (rd *DefaultRouteDistinguisher) DecodeFromBytes(data []byte) error {
	rd.Type = binary.BigEndian.Uint16(data[0:2])
	rd.Value = data[2:8]
	return nil
}

func (rd *DefaultRouteDistinguisher) Serialize() ([]byte, error) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint16(buf, rd.Type)
	copy(buf[2:], rd.Value)
	return buf, nil
}

func (rd *DefaultRouteDistinguisher) Len() int { return 8 }

type RouteDistinguisherTwoOctetASValue struct {
	Admin    uint16
	Assigned uint32
}

type RouteDistinguisherTwoOctetAS struct {
	DefaultRouteDistinguisher
	Value RouteDistinguisherTwoOctetASValue
}

func (rd *RouteDistinguisherTwoOctetAS) Serialize() ([]byte, error) {
	buf := make([]byte, 6)
	binary.BigEndian.PutUint16(buf[0:], rd.Value.Admin)
	binary.BigEndian.PutUint32(buf[2:], rd.Value.Assigned)
	rd.DefaultRouteDistinguisher.Value = buf
	return rd.DefaultRouteDistinguisher.Serialize()
}

func NewRouteDistinguisherTwoOctetAS(admin uint16, assigned uint32) *RouteDistinguisherTwoOctetAS {
	return &RouteDistinguisherTwoOctetAS{
		DefaultRouteDistinguisher{
			Type: BGP_RD_TWO_OCTET_AS,
		},
		RouteDistinguisherTwoOctetASValue{
			Admin:    admin,
			Assigned: assigned,
		},
	}
}

type RouteDistinguisherIPAddressASValue struct {
	Admin    net.IP
	Assigned uint16
}

type RouteDistinguisherIPAddressAS struct {
	DefaultRouteDistinguisher
	Value RouteDistinguisherIPAddressASValue
}

func (rd *RouteDistinguisherIPAddressAS) Serialize() ([]byte, error) {
	buf := make([]byte, 6)
	copy(buf[0:], rd.Value.Admin)
	binary.BigEndian.PutUint16(buf[4:], rd.Value.Assigned)
	rd.DefaultRouteDistinguisher.Value = buf
	return rd.DefaultRouteDistinguisher.Serialize()
}

func NewRouteDistinguisherIPAddressAS(admin string, assigned uint16) *RouteDistinguisherIPAddressAS {
	return &RouteDistinguisherIPAddressAS{
		DefaultRouteDistinguisher{
			Type: BGP_RD_IPV4_ADDRESS,
		},
		RouteDistinguisherIPAddressASValue{
			Admin:    net.ParseIP(admin),
			Assigned: assigned,
		},
	}
}

type RouteDistinguisherFourOctetASValue struct {
	Admin    uint32
	Assigned uint16
}

type RouteDistinguisherFourOctetAS struct {
	DefaultRouteDistinguisher
	Value RouteDistinguisherFourOctetASValue
}

func (rd *RouteDistinguisherFourOctetAS) Serialize() ([]byte, error) {
	buf := make([]byte, 6)
	binary.BigEndian.PutUint32(buf[0:], rd.Value.Admin)
	binary.BigEndian.PutUint16(buf[4:], rd.Value.Assigned)
	rd.DefaultRouteDistinguisher.Value = buf
	return rd.DefaultRouteDistinguisher.Serialize()
}

func NewRouteDistinguisherFourOctetAS(admin uint32, assigned uint16) *RouteDistinguisherFourOctetAS {
	return &RouteDistinguisherFourOctetAS{
		DefaultRouteDistinguisher{
			Type: BGP_RD_FOUR_OCTET_AS,
		},
		RouteDistinguisherFourOctetASValue{
			Admin:    admin,
			Assigned: assigned,
		},
	}
}

type RouteDistinguisherUnknown struct {
	DefaultRouteDistinguisher
}

func getRouteDistinguisher(data []byte) RouteDistinguisherInterface {
	rdtype := binary.BigEndian.Uint16(data[0:2])
	switch rdtype {
	case BGP_RD_TWO_OCTET_AS:
		rd := &RouteDistinguisherTwoOctetAS{}
		rd.Type = rdtype
		rd.Value.Admin = binary.BigEndian.Uint16(data[2:4])
		rd.Value.Assigned = binary.BigEndian.Uint32(data[4:8])
		return rd
	case BGP_RD_IPV4_ADDRESS:
		rd := &RouteDistinguisherIPAddressAS{}
		rd.Type = rdtype
		rd.Value.Admin = data[2:6]
		rd.Value.Assigned = binary.BigEndian.Uint16(data[6:8])
		return rd
	case BGP_RD_FOUR_OCTET_AS:
		rd := &RouteDistinguisherFourOctetAS{}
		rd.Type = rdtype
		rd.Value.Admin = binary.BigEndian.Uint32(data[2:6])
		rd.Value.Assigned = binary.BigEndian.Uint16(data[6:8])
		return rd
	}
	rd := &RouteDistinguisherUnknown{}
	rd.Type = rdtype
	return rd
}

type Label struct {
	Labels []uint32
}

func (l *Label) DecodeFromBytes(data []byte) error {
	labels := []uint32{}
	foundBottom := false
	for len(data) >= 4 {
		label := uint32(data[0]<<16 | data[1]<<8 | data[2])
		data = data[3:]
		labels = append(labels, label>>4)
		if label&1 == 1 {
			foundBottom = true
			break
		}
	}
	if foundBottom == false {
		l.Labels = []uint32{}
		return nil
	}
	l.Labels = labels
	return nil
}

func (l *Label) Serialize() ([]byte, error) {
	buf := make([]byte, len(l.Labels)*3)
	for i, label := range l.Labels {
		label = label << 4
		buf[i*3] = byte((label >> 16) & 0xff)
		buf[i*3+1] = byte((label >> 8) & 0xff)
		buf[i*3+2] = byte(label & 0xff)
	}
	buf[len(buf)-1] |= 1
	return buf, nil
}

func (l *Label) Len() int { return 3 * len(l.Labels) }

func NewLabel(labels ...uint32) *Label {
	return &Label{labels}
}

type LabelledVPNIPAddrPrefix struct {
	IPAddrPrefixDefault
	Labels  Label
	RD      RouteDistinguisherInterface
	addrlen uint8
}

func (l *LabelledVPNIPAddrPrefix) DecodeFromBytes(data []byte) error {
	l.Length = uint8(data[0])
	data = data[1:]
	l.Labels.DecodeFromBytes(data)
	if int(l.Length)-8*(l.Labels.Len()) < 0 {
		l.Labels.Labels = []uint32{}
	}
	data = data[l.Labels.Len():]
	l.RD = getRouteDistinguisher(data)
	data = data[l.RD.Len():]
	restbits := int(l.Length) - 8*(l.Labels.Len()+l.RD.Len())
	l.decodePrefix(data, uint8(restbits), l.addrlen)
	return nil
}

func (l *LabelledVPNIPAddrPrefix) Serialize() ([]byte, error) {
	buf := make([]byte, 1)
	buf[0] = l.Length
	lbuf, err := l.Labels.Serialize()
	if err != nil {
		return nil, err
	}
	buf = append(buf, lbuf...)
	rbuf, err := l.RD.Serialize()
	if err != nil {
		return nil, err
	}
	buf = append(buf, rbuf...)
	restbits := int(l.Length) - 8*(l.Labels.Len()+l.RD.Len())
	pbuf, err := l.serializePrefix(uint8(restbits))
	if err != nil {
		return nil, err
	}
	buf = append(buf, pbuf...)
	return buf, nil
}

func (l *LabelledVPNIPAddrPrefix) AFI() uint16 {
	return AFI_IP
}

func (l *LabelledVPNIPAddrPrefix) SAFI() uint8 {
	return SAFI_MPLS_VPN
}

func NewLabelledVPNIPAddrPrefix(length uint8, prefix string, label Label, rd RouteDistinguisherInterface) *LabelledVPNIPAddrPrefix {
	rdlen := 0
	if rd != nil {
		rdlen = rd.Len()
	}
	return &LabelledVPNIPAddrPrefix{
		IPAddrPrefixDefault{length + uint8(8*(label.Len()+rdlen)), net.ParseIP(prefix)},
		label,
		rd,
		4,
	}
}

type LabelledVPNIPv6AddrPrefix struct {
	LabelledVPNIPAddrPrefix
}

func (l *LabelledVPNIPv6AddrPrefix) AFI() uint16 {
	return AFI_IP6
}

func NewLabelledVPNIPv6AddrPrefix(length uint8, prefix string, label Label, rd RouteDistinguisherInterface) *LabelledVPNIPv6AddrPrefix {
	rdlen := 0
	if rd != nil {
		rdlen = rd.Len()
	}
	return &LabelledVPNIPv6AddrPrefix{
		LabelledVPNIPAddrPrefix{
			IPAddrPrefixDefault{length + uint8(8*(label.Len()+rdlen)), net.ParseIP(prefix)},
			label,
			rd,
			16,
		},
	}
}

type LabelledIPAddrPrefix struct {
	IPAddrPrefixDefault
	Labels  Label
	addrlen uint8
}

func (r *LabelledIPAddrPrefix) AFI() uint16 {
	return AFI_IP
}

func (r *LabelledIPAddrPrefix) SAFI() uint8 {
	return SAFI_MPLS_LABEL
}

func (r *IPAddrPrefix) decodeNextHop(data []byte) net.IP {
	if r.addrlen == 0 {
		r.addrlen = 4
	}
	var next net.IP = data[0:r.addrlen]
	return next
}

func (r *LabelledVPNIPAddrPrefix) decodeNextHop(data []byte) net.IP {
	// skip rd
	var next net.IP = data[8 : 8+r.addrlen]
	return next
}

func (r *LabelledIPAddrPrefix) decodeNextHop(data []byte) net.IP {
	var next net.IP = data[0:r.addrlen]
	return next
}

func (l *LabelledIPAddrPrefix) DecodeFromBytes(data []byte) error {
	l.Length = uint8(data[0])
	data = data[1:]
	l.Labels.DecodeFromBytes(data)
	if int(l.Length)-8*(l.Labels.Len()) < 0 {
		l.Labels.Labels = []uint32{}
	}
	restbits := int(l.Length) - 8*(l.Labels.Len())
	data = data[l.Labels.Len():]
	l.decodePrefix(data, uint8(restbits), l.addrlen)
	return nil
}

func (l *LabelledIPAddrPrefix) Serialize() ([]byte, error) {
	buf := make([]byte, 1)
	buf[0] = l.Length
	restbits := int(l.Length) - 8*(l.Labels.Len())
	lbuf, err := l.Labels.Serialize()
	if err != nil {
		return nil, err
	}
	buf = append(buf, lbuf...)
	pbuf, err := l.serializePrefix(uint8(restbits))
	if err != nil {
		return nil, err
	}
	buf = append(buf, pbuf...)
	return buf, nil
}

func NewLabelledIPAddrPrefix(length uint8, prefix string, label Label) *LabelledIPAddrPrefix {
	return &LabelledIPAddrPrefix{
		IPAddrPrefixDefault{length + uint8(label.Len()*8), net.ParseIP(prefix)},
		label,
		4,
	}
}

type LabelledIPv6AddrPrefix struct {
	LabelledIPAddrPrefix
}

func NewLabelledIPv6AddrPrefix(length uint8, prefix string, label Label) *LabelledIPv6AddrPrefix {
	return &LabelledIPv6AddrPrefix{
		LabelledIPAddrPrefix{
			IPAddrPrefixDefault{length + uint8(label.Len()*8), net.ParseIP(prefix)},
			label,
			16,
		},
	}
}

type RouteTargetMembershipNLRI struct {
	AS          uint32
	RouteTarget ExtendedCommunityInterface
}

func (n *RouteTargetMembershipNLRI) DecodeFromBytes(data []byte) error {
	n.AS = binary.BigEndian.Uint32(data[0:4])
	n.RouteTarget = parseExtended(data[4:])
	return nil
}

func (n *RouteTargetMembershipNLRI) Serialize() ([]byte, error) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, n.AS)
	ebuf, err := n.RouteTarget.Serialize()
	if err != nil {
		return nil, err
	}
	return append(buf, ebuf...), nil
}

func (n *RouteTargetMembershipNLRI) AFI() uint16 {
	return AFI_IP
}

func (n *RouteTargetMembershipNLRI) SAFI() uint8 {
	return SAFI_ROUTE_TARGET_CONSTRTAINS
}

func (n *RouteTargetMembershipNLRI) Len() int { return 12 }

func rfshift(afi uint16, safi uint8) RouteFamily {
	return RouteFamily(int(afi)<<16 | int(safi))
}

type RouteFamily int

const (
	RF_IPv4_UC   RouteFamily = AFI_IP<<16 | SAFI_UNICAST
	RF_IPv6_UC   RouteFamily = AFI_IP6<<16 | SAFI_UNICAST
	RF_IPv4_VPN  RouteFamily = AFI_IP<<16 | SAFI_MPLS_VPN
	RF_IPv6_VPN  RouteFamily = AFI_IP6<<16 | SAFI_MPLS_VPN
	RF_IPv4_MPLS RouteFamily = AFI_IP<<16 | SAFI_MPLS_LABEL
	RF_IPv6_MPLS RouteFamily = AFI_IP6<<16 | SAFI_MPLS_LABEL
	RF_RTC_UC    RouteFamily = AFI_IP<<16 | SAFI_ROUTE_TARGET_CONSTRTAINS
)

func routeFamilyPrefix(afi uint16, safi uint8) (prefix AddrPrefixInterface, err error) {
	switch rfshift(afi, safi) {
	case RF_IPv4_UC:
		prefix = NewIPAddrPrefix(0, "")
	case RF_IPv6_UC:
		prefix = NewIPv6AddrPrefix(0, "")
	case RF_IPv4_VPN:
		prefix = NewLabelledVPNIPAddrPrefix(0, "", *NewLabel(), nil)
	case RF_IPv6_VPN:
		prefix = NewLabelledVPNIPv6AddrPrefix(0, "", *NewLabel(), nil)
	case RF_IPv4_MPLS:
		prefix = NewLabelledIPAddrPrefix(0, "", *NewLabel())
	case RF_IPv6_MPLS:
		prefix = NewLabelledIPv6AddrPrefix(0, "", *NewLabel())
	case RF_RTC_UC:
		prefix = &RouteTargetMembershipNLRI{}
	default:
		return nil, errors.New("unknown route family")
	}
	return prefix, nil
}

const (
	BGP_ATTR_FLAG_EXTENDED_LENGTH = 1 << 4
	BGP_ATTR_FLAG_PARTIAL         = 1 << 5
	BGP_ATTR_FLAG_TRANSITIVE      = 1 << 6
	BGP_ATTR_FLAG_OPTIONAL        = 1 << 7
)

type BGPAttrType uint8

const (
	_ BGPAttrType = iota
	BGP_ATTR_TYPE_ORIGIN
	BGP_ATTR_TYPE_AS_PATH
	BGP_ATTR_TYPE_NEXT_HOP
	BGP_ATTR_TYPE_MULTI_EXIT_DISC
	BGP_ATTR_TYPE_LOCAL_PREF
	BGP_ATTR_TYPE_ATOMIC_AGGREGATE
	BGP_ATTR_TYPE_AGGREGATOR
	BGP_ATTR_TYPE_COMMUNITIES
	BGP_ATTR_TYPE_ORIGINATOR_ID
	BGP_ATTR_TYPE_CLUSTER_LIST
	_
	_
	_
	BGP_ATTR_TYPE_MP_REACH_NLRI
	BGP_ATTR_TYPE_MP_UNREACH_NLRI
	BGP_ATTR_TYPE_EXTENDED_COMMUNITIES
	BGP_ATTR_TYPE_AS4_PATH
	BGP_ATTR_TYPE_AS4_AGGREGATOR
)

// NOTIFICATION Error Code  RFC 4271 4.5.
const (
	_ = iota
	BGP_ERROR_MESSAGE_HEADER_ERROR
	BGP_ERROR_OPEN_MESSAGE_ERROR
	BGP_ERROR_UPDATE_MESSAGE_ERROR
	BGP_ERROR_HOLD_TIMER_EXPIRED
	BGP_ERROR_FSM_ERROR
	BGP_ERROR_CEASE
)

// NOTIFICATION Error Subcode for BGP_ERROR_MESSAGE_HEADER_ERROR
const (
	_ = iota
	BGP_ERROR_SUB_CONNECTION_NOT_SYNCHRONIZED
	BGP_ERROR_SUB_BAD_MESSAGE_LENGTH
	BGP_ERROR_SUB_BAD_MESSAGE_TYPE
)

// NOTIFICATION Error Subcode for BGP_ERROR_OPEN_MESSAGE_ERROR
const (
	_ = iota
	BGP_ERROR_SUB_UNSUPPORTED_VERSION_NUMBER
	BGP_ERROR_SUB_BAD_PEER_AS
	BGP_ERROR_SUB_BAD_BGP_IDENTIFIER
	BGP_ERROR_SUB_UNSUPPORTED_OPTIONAL_PARAMETER
	BGP_ERROR_SUB_AUTHENTICATION_FAILURE
	BGP_ERROR_SUB_UNACCEPTABLE_HOLD_TIME
)

// NOTIFICATION Error Subcode for BGP_ERROR_UPDATE_MESSAGE_ERROR
const (
	_ = iota
	BGP_ERROR_SUB_MALFORMED_ATTRIBUTE_LIST
	BGP_ERROR_SUB_UNRECOGNIZED_WELL_KNOWN_ATTRIBUTE
	BGP_ERROR_SUB_MISSING_WELL_KNOWN_ATTRIBUTE
	BGP_ERROR_SUB_ATTRIBUTE_FLAGS_ERROR
	BGP_ERROR_SUB_ATTRIBUTE_LENGTH_ERROR
	BGP_ERROR_SUB_INVALID_ORIGIN_ATTRIBUTE
	BGP_ERROR_SUB_ROUTING_LOOP
	BGP_ERROR_SUB_INVALID_NEXT_HOP_ATTRIBUTE
	BGP_ERROR_SUB_OPTIONAL_ATTRIBUTE_ERROR
	BGP_ERROR_SUB_INVALID_NETWORK_FIELD
	BGP_ERROR_SUB_MALFORMED_AS_PATH
)

// NOTIFICATION Error Subcode for BGP_ERROR_HOLD_TIMER_EXPIRED
const (
	_ = iota
	BGP_ERROR_SUB_HOLD_TIMER_EXPIRED
)

// NOTIFICATION Error Subcode for BGP_ERROR_FSM_ERROR
const (
	_ = iota
	BGP_ERROR_SUB_FSM_ERROR
)

// NOTIFICATION Error Subcode for BGP_ERROR_CEASE  (RFC 4486)
const (
	_ = iota
	BGP_ERROR_SUB_MAXIMUM_NUMBER_OF_PREFIXES_REACHED
	BGP_ERROR_SUB_ADMINISTRATIVE_SHUTDOWN
	BGP_ERROR_SUB_PEER_DECONFIGURED
	BGP_ERROR_SUB_ADMINISTRATIVE_RESET
	BGP_ERROR_SUB_CONNECTION_RESET
	BGP_ERROR_SUB_OTHER_CONFIGURATION_CHANGE
	BGP_ERROR_SUB_CONNECTION_COLLISION_RESOLUTION
	BGP_ERROR_SUB_OUT_OF_RESOURCES
)

var pathAttrFlags map[BGPAttrType]uint8 = map[BGPAttrType]uint8{
	BGP_ATTR_TYPE_ORIGIN:               BGP_ATTR_FLAG_TRANSITIVE,
	BGP_ATTR_TYPE_AS_PATH:              BGP_ATTR_FLAG_TRANSITIVE,
	BGP_ATTR_TYPE_NEXT_HOP:             BGP_ATTR_FLAG_TRANSITIVE,
	BGP_ATTR_TYPE_MULTI_EXIT_DISC:      BGP_ATTR_FLAG_OPTIONAL,
	BGP_ATTR_TYPE_LOCAL_PREF:           BGP_ATTR_FLAG_TRANSITIVE,
	BGP_ATTR_TYPE_ATOMIC_AGGREGATE:     BGP_ATTR_FLAG_TRANSITIVE,
	BGP_ATTR_TYPE_AGGREGATOR:           BGP_ATTR_FLAG_TRANSITIVE | BGP_ATTR_FLAG_OPTIONAL,
	BGP_ATTR_TYPE_COMMUNITIES:          BGP_ATTR_FLAG_TRANSITIVE | BGP_ATTR_FLAG_OPTIONAL,
	BGP_ATTR_TYPE_ORIGINATOR_ID:        BGP_ATTR_FLAG_OPTIONAL,
	BGP_ATTR_TYPE_CLUSTER_LIST:         BGP_ATTR_FLAG_OPTIONAL,
	BGP_ATTR_TYPE_MP_REACH_NLRI:        BGP_ATTR_FLAG_OPTIONAL,
	BGP_ATTR_TYPE_MP_UNREACH_NLRI:      BGP_ATTR_FLAG_OPTIONAL,
	BGP_ATTR_TYPE_EXTENDED_COMMUNITIES: BGP_ATTR_FLAG_TRANSITIVE | BGP_ATTR_FLAG_OPTIONAL,
	BGP_ATTR_TYPE_AS4_PATH:             BGP_ATTR_FLAG_TRANSITIVE | BGP_ATTR_FLAG_OPTIONAL,
	BGP_ATTR_TYPE_AS4_AGGREGATOR:       BGP_ATTR_FLAG_TRANSITIVE | BGP_ATTR_FLAG_OPTIONAL,
}

type PathAttributeInterface interface {
	DecodeFromBytes([]byte) error
	Serialize() ([]byte, error)
	Len() int
	getFlags() uint8
	getType() BGPAttrType
}

type PathAttribute struct {
	Flags  uint8
	Type   BGPAttrType
	Length uint16
	Value  []byte
}

func (p *PathAttribute) Len() int {
	l := 2 + p.Length
	if p.Flags&BGP_ATTR_FLAG_EXTENDED_LENGTH != 0 {
		l += 2
	} else {
		l += 1
	}
	return int(l)
}

func (p *PathAttribute) getFlags() uint8 {
	return p.Flags
}

func (p *PathAttribute) getType() BGPAttrType {
	return p.Type
}

func (p *PathAttribute) DecodeFromBytes(data []byte) error {
	eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
	eSubCode := uint8(BGP_ERROR_SUB_ATTRIBUTE_LENGTH_ERROR)
	if len(data) < 2 {
		return NewMessageError(eCode, eSubCode, data, "attribute header length is short")
	}
	p.Flags = data[0]
	p.Type = BGPAttrType(data[1])

	if p.Flags&BGP_ATTR_FLAG_EXTENDED_LENGTH != 0 {
		if len(data) < 4 {
			return NewMessageError(eCode, eSubCode, data, "attribute header length is short")
		}
		p.Length = binary.BigEndian.Uint16(data[2:4])
		data = data[4:]
	} else {
		if len(data) < 3 {
			return NewMessageError(eCode, eSubCode, data, "attribute header length is short")
		}
		p.Length = uint16(data[2])
		data = data[3:]
	}
	if len(data) < int(p.Length) {
		return NewMessageError(eCode, eSubCode, data, "attribute value length is short")
	}
	p.Value = data[:p.Length]

	return nil
}

func (p *PathAttribute) Serialize() ([]byte, error) {
	p.Length = uint16(len(p.Value))
	if p.Length > 255 {
		p.Flags |= BGP_ATTR_FLAG_EXTENDED_LENGTH
	} else {
		p.Flags &^= BGP_ATTR_FLAG_EXTENDED_LENGTH
	}
	buf := make([]byte, p.Len())
	buf[0] = p.Flags
	buf[1] = uint8(p.Type)
	if p.Flags&BGP_ATTR_FLAG_EXTENDED_LENGTH != 0 {
		binary.BigEndian.PutUint16(buf[2:4], p.Length)
		copy(buf[4:], p.Value)
	} else {
		buf[2] = byte(p.Length)
		copy(buf[3:], p.Value)
	}
	return buf, nil
}

type PathAttributeOrigin struct {
	PathAttribute
}

func (p *PathAttributeOrigin) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type  string
		Value uint8
	}{
		Type:  p.Type.String(),
		Value: uint8(p.Value[0]),
	})
}

func NewPathAttributeOrigin(value uint8) *PathAttributeOrigin {
	t := BGP_ATTR_TYPE_ORIGIN
	return &PathAttributeOrigin{

		PathAttribute: PathAttribute{
			Flags: pathAttrFlags[t],
			Type:  t,
			Value: []byte{byte(value)},
		},
	}
}

type AsPathParam struct {
	Type uint8
	Num  uint8
	AS   []uint16
}

func (a *AsPathParam) Serialize() ([]byte, error) {
	buf := make([]byte, 2+len(a.AS)*2)
	buf[0] = uint8(a.Type)
	buf[1] = a.Num
	for j, as := range a.AS {
		binary.BigEndian.PutUint16(buf[2+j*2:], as)
	}
	return buf, nil
}

func (a *AsPathParam) DecodeFromBytes(data []byte) error {
	eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
	eSubCode := uint8(BGP_ERROR_SUB_MALFORMED_AS_PATH)
	if len(data) < 2 {
		return NewMessageError(eCode, eSubCode, nil, "AS param header length is short")
	}
	a.Type = data[0]
	a.Num = data[1]
	data = data[2:]
	if len(data) < int(a.Num*2) {
		return NewMessageError(eCode, eSubCode, nil, "AS param data length is short")
	}
	for i := 0; i < int(a.Num); i++ {
		a.AS = append(a.AS, binary.BigEndian.Uint16(data))
		data = data[2:]
	}
	return nil
}

func (a *AsPathParam) Len() int {
	return 2 + len(a.AS)*2
}

func (a *AsPathParam) ASLen() int {
	return len(a.AS)
}

func NewAsPathParam(segType uint8, as []uint16) *AsPathParam {
	return &AsPathParam{
		Type: segType,
		Num:  uint8(len(as)),
		AS:   as,
	}
}

type As4PathParam struct {
	Type uint8
	Num  uint8
	AS   []uint32
}

func (a *As4PathParam) Serialize() ([]byte, error) {
	buf := make([]byte, 2+len(a.AS)*4)
	buf[0] = a.Type
	buf[1] = a.Num
	for j, as := range a.AS {
		binary.BigEndian.PutUint32(buf[2+j*4:], as)
	}
	return buf, nil
}

func (a *As4PathParam) DecodeFromBytes(data []byte) error {
	eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
	eSubCode := uint8(BGP_ERROR_SUB_MALFORMED_AS_PATH)
	if len(data) < 2 {
		return NewMessageError(eCode, eSubCode, nil, "AS4 param header length is short")
	}
	a.Type = data[0]
	a.Num = data[1]
	data = data[2:]
	if len(data) < int(a.Num*4) {
		return NewMessageError(eCode, eSubCode, nil, "AS4 param data length is short")
	}
	for i := 0; i < int(a.Num); i++ {
		a.AS = append(a.AS, binary.BigEndian.Uint32(data))
		data = data[4:]
	}
	return nil
}

func (a *As4PathParam) Len() int {
	return 2 + len(a.AS)*4
}

func (a *As4PathParam) ASLen() int {
	return len(a.AS)
}

func NewAs4PathParam(segType uint8, as []uint32) *As4PathParam {
	return &As4PathParam{
		Type: segType,
		Num:  uint8(len(as)),
		AS:   as,
	}
}

type DefaultAsPath struct {
}

func (p *DefaultAsPath) isValidAspath(data []byte) (bool, error) {
	eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
	eSubCode := uint8(BGP_ERROR_SUB_MALFORMED_AS_PATH)
	if len(data)%2 != 0 {
		return false, NewMessageError(eCode, eSubCode, nil, "AS PATH length is not odd")
	}

	tryParse := func(data []byte, use4byte bool) (bool, error) {
		for len(data) > 0 {
			if len(data) < 2 {
				return false, NewMessageError(eCode, eSubCode, nil, "AS PATH header is short")
			}
			segType := data[0]
			if segType == 0 || segType > 4 {
				return false, NewMessageError(eCode, eSubCode, nil, "unknown AS_PATH seg type")
			}
			asNum := data[1]
			data = data[2:]
			if asNum == 0 || int(asNum) > math.MaxUint8 {
				return false, NewMessageError(eCode, eSubCode, nil, "AS PATH the number of AS is incorrect")
			}
			segLength := asNum
			if use4byte == true {
				segLength *= 4
			} else {
				segLength *= 2
			}
			if int(segLength) > len(data) {
				return false, NewMessageError(eCode, eSubCode, nil, "seg length is short")
			}
			data = data[segLength:]
		}
		return true, nil
	}
	_, err := tryParse(data, true)
	if err == nil {
		return true, nil
	}

	_, err = tryParse(data, false)
	if err == nil {
		return false, nil
	}
	return false, NewMessageError(eCode, eSubCode, nil, "can't not parse")
}

type AsPathParamInterface interface {
	Serialize() ([]byte, error)
	DecodeFromBytes([]byte) error
	Len() int
	ASLen() int
}

type PathAttributeAsPath struct {
	DefaultAsPath
	PathAttribute
	Value []AsPathParamInterface
}

func (p *PathAttributeAsPath) DecodeFromBytes(data []byte) error {
	err := p.PathAttribute.DecodeFromBytes(data)
	if err != nil {
		return err
	}
	if p.PathAttribute.Length == 0 {
		// ibgp or something
		return nil
	}
	as4Bytes, err := p.DefaultAsPath.isValidAspath(p.PathAttribute.Value)
	if err != nil {
		return err
	}
	v := p.PathAttribute.Value
	for len(v) > 0 {
		var tuple AsPathParamInterface
		if as4Bytes == true {
			tuple = &As4PathParam{}
		} else {
			tuple = &AsPathParam{}
		}
		err := tuple.DecodeFromBytes(v)
		if err != nil {
			return err
		}
		p.Value = append(p.Value, tuple)
		if tuple.Len() > len(v) {

		}
		v = v[tuple.Len():]
	}
	return nil
}

func (p *PathAttributeAsPath) Serialize() ([]byte, error) {
	buf := make([]byte, 0)
	for _, v := range p.Value {
		vbuf, err := v.Serialize()
		if err != nil {
			return nil, err
		}
		buf = append(buf, vbuf...)
	}
	p.PathAttribute.Value = buf
	return p.PathAttribute.Serialize()
}

func (p *PathAttributeAsPath) MarshalJSON() ([]byte, error) {
	aslist := make([]uint32, 0)
	for _, a := range p.Value {
		path, y := a.(*As4PathParam)
		if y {
			aslist = append(aslist, path.AS...)
		} else {
			path := a.(*AsPathParam)
			for _, v := range path.AS {
				aslist = append(aslist, uint32(v))
			}
		}
	}
	return json.Marshal(struct {
		Type   string
		AsPath []uint32
	}{
		Type:   p.Type.String(),
		AsPath: aslist,
	})
}

func NewPathAttributeAsPath(value []AsPathParamInterface) *PathAttributeAsPath {
	t := BGP_ATTR_TYPE_AS_PATH
	return &PathAttributeAsPath{
		PathAttribute: PathAttribute{
			Flags: pathAttrFlags[t],
			Type:  t,
		},
		Value: value,
	}
}

type PathAttributeNextHop struct {
	PathAttribute
	Value net.IP
}

func (p *PathAttributeNextHop) DecodeFromBytes(data []byte) error {
	err := p.PathAttribute.DecodeFromBytes(data)
	if err != nil {
		return err
	}
	if len(p.PathAttribute.Value) != 4 && len(p.PathAttribute.Value) != 16 {
		eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
		eSubCode := uint8(BGP_ERROR_SUB_ATTRIBUTE_LENGTH_ERROR)
		return NewMessageError(eCode, eSubCode, nil, "nexthop length isn't correct")
	}
	p.Value = p.PathAttribute.Value
	return nil
}

func (p *PathAttributeNextHop) Serialize() ([]byte, error) {
	p.PathAttribute.Value = p.Value
	return p.PathAttribute.Serialize()
}

func (p *PathAttributeNextHop) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type    string
		Nexthop string
	}{
		Type:    p.Type.String(),
		Nexthop: p.Value.String(),
	})
}

func NewPathAttributeNextHop(value string) *PathAttributeNextHop {
	t := BGP_ATTR_TYPE_NEXT_HOP
	return &PathAttributeNextHop{
		PathAttribute: PathAttribute{
			Flags: pathAttrFlags[t],
			Type:  t,
		},
		Value: net.ParseIP(value).To4(),
	}
}

type PathAttributeMultiExitDisc struct {
	PathAttribute
	Value uint32
}

func (p *PathAttributeMultiExitDisc) DecodeFromBytes(data []byte) error {
	err := p.PathAttribute.DecodeFromBytes(data)
	if err != nil {
		return err
	}
	if len(p.PathAttribute.Value) != 4 {
		eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
		eSubCode := uint8(BGP_ERROR_SUB_ATTRIBUTE_LENGTH_ERROR)
		return NewMessageError(eCode, eSubCode, nil, "med length isn't correct")
	}
	p.Value = binary.BigEndian.Uint32(p.PathAttribute.Value)
	return nil
}

func (p *PathAttributeMultiExitDisc) Serialize() ([]byte, error) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, p.Value)
	p.PathAttribute.Value = buf
	return p.PathAttribute.Serialize()
}

func (p *PathAttributeMultiExitDisc) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type   string
		Metric uint32
	}{
		Type:   p.Type.String(),
		Metric: p.Value,
	})
}

func NewPathAttributeMultiExitDisc(value uint32) *PathAttributeMultiExitDisc {
	t := BGP_ATTR_TYPE_MULTI_EXIT_DISC
	return &PathAttributeMultiExitDisc{
		PathAttribute: PathAttribute{
			Flags: pathAttrFlags[t],
			Type:  t,
		},
		Value: value,
	}
}

type PathAttributeLocalPref struct {
	PathAttribute
	Value uint32
}

func (p *PathAttributeLocalPref) DecodeFromBytes(data []byte) error {
	err := p.PathAttribute.DecodeFromBytes(data)
	if err != nil {
		return err
	}
	if len(p.PathAttribute.Value) != 4 {
		eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
		eSubCode := uint8(BGP_ERROR_SUB_ATTRIBUTE_LENGTH_ERROR)
		return NewMessageError(eCode, eSubCode, nil, "local pref length isn't correct")
	}
	p.Value = binary.BigEndian.Uint32(p.PathAttribute.Value)
	return nil
}

func (p *PathAttributeLocalPref) Serialize() ([]byte, error) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, p.Value)
	p.PathAttribute.Value = buf
	return p.PathAttribute.Serialize()
}

func (p *PathAttributeLocalPref) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string
		Pref uint32
	}{
		Type: p.Type.String(),
		Pref: p.Value,
	})
}

func NewPathAttributeLocalPref(value uint32) *PathAttributeLocalPref {
	t := BGP_ATTR_TYPE_LOCAL_PREF
	return &PathAttributeLocalPref{
		PathAttribute: PathAttribute{
			Flags: pathAttrFlags[t],
			Type:  t,
		},
		Value: value,
	}
}

type PathAttributeAtomicAggregate struct {
	PathAttribute
}

func (p *PathAttributeAtomicAggregate) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string
	}{
		Type: p.Type.String(),
	})
}

func NewPathAttributeAtomicAggregate() *PathAttributeAtomicAggregate {
	t := BGP_ATTR_TYPE_ATOMIC_AGGREGATE
	return &PathAttributeAtomicAggregate{
		PathAttribute: PathAttribute{
			Flags: pathAttrFlags[t],
			Type:  t,
		},
	}
}

type PathAttributeAggregatorParam struct {
	AS      uint32
	askind  reflect.Kind
	Address net.IP
}

type PathAttributeAggregator struct {
	PathAttribute
	Value PathAttributeAggregatorParam
}

func (p *PathAttributeAggregator) DecodeFromBytes(data []byte) error {
	err := p.PathAttribute.DecodeFromBytes(data)
	if err != nil {
		return err
	}
	if len(p.PathAttribute.Value) != 6 && len(p.PathAttribute.Value) != 8 {
		eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
		eSubCode := uint8(BGP_ERROR_SUB_ATTRIBUTE_LENGTH_ERROR)
		return NewMessageError(eCode, eSubCode, nil, "aggregator length isn't correct")
	}
	if len(p.PathAttribute.Value) == 6 {
		p.Value.AS = uint32(binary.BigEndian.Uint16(p.PathAttribute.Value[0:2]))
		p.Value.Address = p.PathAttribute.Value[2:]
		p.Value.askind = reflect.Uint16
	} else {
		p.Value.AS = binary.BigEndian.Uint32(p.PathAttribute.Value[0:4])
		p.Value.Address = p.PathAttribute.Value[4:]
		p.Value.askind = reflect.Uint32
	}
	return nil
}

func (p *PathAttributeAggregator) Serialize() ([]byte, error) {
	var buf []byte
	switch p.Value.askind {
	case reflect.Uint16:
		buf = make([]byte, 6)
		binary.BigEndian.PutUint16(buf, uint16(p.Value.AS))
		copy(buf[2:], p.Value.Address)
	case reflect.Uint32:
		buf = make([]byte, 8)
		binary.BigEndian.PutUint32(buf, p.Value.AS)
		copy(buf[4:], p.Value.Address)
	}

	p.PathAttribute.Value = buf
	return p.PathAttribute.Serialize()
}

func (p *PathAttributeAggregator) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type    string
		AS      uint32
		Address net.IP
	}{
		Type:    p.Type.String(),
		AS:      p.Value.AS,
		Address: p.Value.Address,
	})
}

func NewPathAttributeAggregator(as interface{}, address string) *PathAttributeAggregator {
	v := reflect.ValueOf(as)
	t := BGP_ATTR_TYPE_AGGREGATOR
	return &PathAttributeAggregator{
		PathAttribute: PathAttribute{
			Flags: pathAttrFlags[t],
			Type:  t,
		},
		Value: PathAttributeAggregatorParam{
			AS:      uint32(v.Uint()),
			askind:  v.Kind(),
			Address: net.ParseIP(address).To4(),
		},
	}
}

type PathAttributeCommunities struct {
	PathAttribute
	Value []uint32
}

func (p *PathAttributeCommunities) DecodeFromBytes(data []byte) error {
	err := p.PathAttribute.DecodeFromBytes(data)
	if err != nil {
		return err
	}
	if len(p.PathAttribute.Value)%4 != 0 {
		eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
		eSubCode := uint8(BGP_ERROR_SUB_ATTRIBUTE_LENGTH_ERROR)
		return NewMessageError(eCode, eSubCode, nil, "communities length isn't correct")
	}
	value := p.PathAttribute.Value
	for len(value) >= 4 {
		p.Value = append(p.Value, binary.BigEndian.Uint32(value))
		value = value[4:]
	}
	return nil
}

func (p *PathAttributeCommunities) Serialize() ([]byte, error) {
	buf := make([]byte, len(p.Value)*4)
	for i, v := range p.Value {
		binary.BigEndian.PutUint32(buf[i*4:], v)
	}
	p.PathAttribute.Value = buf
	return p.PathAttribute.Serialize()
}

func (p *PathAttributeCommunities) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type  string
		Value []uint32
	}{
		Type:  p.Type.String(),
		Value: p.Value,
	})
}

func NewPathAttributeCommunities(value []uint32) *PathAttributeCommunities {
	t := BGP_ATTR_TYPE_COMMUNITIES
	return &PathAttributeCommunities{
		PathAttribute{
			Flags:  pathAttrFlags[t],
			Type:   t,
			Length: 0,
			Value:  nil},
		value,
	}
}

type PathAttributeOriginatorId struct {
	PathAttribute
	Value net.IP
}

func (p *PathAttributeOriginatorId) DecodeFromBytes(data []byte) error {
	err := p.PathAttribute.DecodeFromBytes(data)
	if err != nil {
		return err
	}
	if len(p.PathAttribute.Value) != 4 && len(p.PathAttribute.Value) != 16 {
		eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
		eSubCode := uint8(BGP_ERROR_SUB_ATTRIBUTE_LENGTH_ERROR)
		return NewMessageError(eCode, eSubCode, nil, "originatorid length isn't correct")
	}
	p.Value = p.PathAttribute.Value
	return nil
}

func (p *PathAttributeOriginatorId) Serialize() ([]byte, error) {
	buf := make([]byte, 4)
	copy(buf, p.Value)
	p.PathAttribute.Value = buf
	return p.PathAttribute.Serialize()
}

func (p *PathAttributeOriginatorId) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type    string
		Address string
	}{
		Type:    p.Type.String(),
		Address: p.Value.String(),
	})
}

func NewPathAttributeOriginatorId(value string) *PathAttributeOriginatorId {
	t := BGP_ATTR_TYPE_ORIGINATOR_ID
	return &PathAttributeOriginatorId{
		PathAttribute{
			Flags:  pathAttrFlags[t],
			Type:   t,
			Length: 0,
			Value:  nil},
		net.ParseIP(value).To4(),
	}
}

type PathAttributeClusterList struct {
	PathAttribute
	Value []net.IP
}

func (p *PathAttributeClusterList) DecodeFromBytes(data []byte) error {
	err := p.PathAttribute.DecodeFromBytes(data)
	if err != nil {
		return err
	}
	value := p.PathAttribute.Value
	if len(p.PathAttribute.Value)%4 != 0 {
		eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
		eSubCode := uint8(BGP_ERROR_SUB_ATTRIBUTE_LENGTH_ERROR)
		return NewMessageError(eCode, eSubCode, nil, "clusterlist length isn't correct")
	}
	for len(value) >= 4 {
		p.Value = append(p.Value, value[:4])
		value = value[4:]
	}
	return nil
}

func (p *PathAttributeClusterList) Serialize() ([]byte, error) {
	buf := make([]byte, len(p.Value)*4)
	for i, v := range p.Value {
		copy(buf[i*4:], v)
	}
	p.PathAttribute.Value = buf
	return p.PathAttribute.Serialize()
}

func (p *PathAttributeClusterList) MarshalJSON() ([]byte, error) {
	l := make([]string, 0)
	for _, addr := range p.Value {
		l = append(l, addr.String())
	}

	return json.Marshal(struct {
		Type    string
		Address []string
	}{
		Type:    p.Type.String(),
		Address: l,
	})
}

func NewPathAttributeClusterList(value []string) *PathAttributeClusterList {
	l := make([]net.IP, len(value))
	for i, v := range value {
		l[i] = net.ParseIP(v).To4()
	}
	t := BGP_ATTR_TYPE_CLUSTER_LIST
	return &PathAttributeClusterList{
		PathAttribute{
			Flags:  pathAttrFlags[t],
			Type:   t,
			Length: 0,
			Value:  nil},
		l,
	}
}

type PathAttributeMpReachNLRI struct {
	PathAttribute
	Nexthop net.IP
	Value   []AddrPrefixInterface
}

func (p *PathAttributeMpReachNLRI) DecodeFromBytes(data []byte) error {
	err := p.PathAttribute.DecodeFromBytes(data)
	if err != nil {
		return err
	}
	eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
	eSubCode := uint8(BGP_ERROR_SUB_ATTRIBUTE_LENGTH_ERROR)

	value := p.PathAttribute.Value
	if len(value) < 3 {
		return NewMessageError(eCode, eSubCode, value, "mpreach header length is short")
	}
	afi := binary.BigEndian.Uint16(value[0:2])
	safi := value[2]
	nexthopLen := value[3]
	if len(value) < int(4+nexthopLen) {
		return NewMessageError(eCode, eSubCode, value, "mpreach nexthop length is short")
	}
	nexthopbin := value[4 : 4+nexthopLen]
	value = value[4+nexthopLen:]
	if nexthopLen > 0 {
		offset := 0
		if safi == SAFI_MPLS_VPN {
			offset = 8
		}
		addrlen := 4
		if afi == AFI_IP6 {
			addrlen = 16
		}
		if len(nexthopbin) != offset+addrlen {
			return NewMessageError(eCode, eSubCode, value, "mpreach nexthop length is incorrect")
		}
		p.Nexthop = nexthopbin[offset : +offset+addrlen]
	}
	// skip reserved
	if len(value) == 0 {
		return NewMessageError(eCode, eSubCode, value, "no skip byte")
	}
	value = value[1:]
	for len(value) > 0 {
		prefix, err := routeFamilyPrefix(afi, safi)
		if err != nil {
			return NewMessageError(eCode, BGP_ERROR_SUB_ATTRIBUTE_FLAGS_ERROR, data[:p.PathAttribute.Len()], err.Error())
		}
		err = prefix.DecodeFromBytes(value)
		if err != nil {
			return err
		}
		if prefix.Len() > len(value) {
			return NewMessageError(eCode, eSubCode, value, "prefix length is incorrect")
		}
		value = value[prefix.Len():]
		p.Value = append(p.Value, prefix)
	}
	return nil
}

func (p *PathAttributeMpReachNLRI) Serialize() ([]byte, error) {
	afi := p.Value[0].AFI()
	safi := p.Value[0].SAFI()
	nexthoplen := 4
	if afi == AFI_IP6 {
		nexthoplen = 16
	}
	offset := 0
	if safi == SAFI_MPLS_VPN {
		offset = 8
		nexthoplen += 8
	}
	buf := make([]byte, 4+nexthoplen)
	binary.BigEndian.PutUint16(buf[0:], afi)
	buf[2] = safi
	buf[3] = uint8(nexthoplen)
	copy(buf[4+offset:], p.Nexthop)
	buf = append(buf, make([]byte, 1)...)
	for _, prefix := range p.Value {
		pbuf, err := prefix.Serialize()
		if err != nil {
			return nil, err
		}
		buf = append(buf, pbuf...)
	}
	p.PathAttribute.Value = buf
	return p.PathAttribute.Serialize()
}

func (p *PathAttributeMpReachNLRI) MarshalJSON() ([]byte, error) {
	// TODO: fix address printing
	return json.Marshal(struct {
		Type    string
		Nexthop string
		Address []string
	}{
		Type:    p.Type.String(),
		Nexthop: p.Nexthop.String(),
	})
}

func NewPathAttributeMpReachNLRI(nexthop string, nlri []AddrPrefixInterface) *PathAttributeMpReachNLRI {
	t := BGP_ATTR_TYPE_MP_REACH_NLRI
	return &PathAttributeMpReachNLRI{
		PathAttribute: PathAttribute{
			Flags: pathAttrFlags[t],
			Type:  t,
		},
		Nexthop: net.ParseIP(nexthop),
		Value:   nlri,
	}
}

type PathAttributeMpUnreachNLRI struct {
	PathAttribute
	Value []AddrPrefixInterface
}

func (p *PathAttributeMpUnreachNLRI) DecodeFromBytes(data []byte) error {
	err := p.PathAttribute.DecodeFromBytes(data)
	if err != nil {
		return nil
	}
	eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
	eSubCode := uint8(BGP_ERROR_SUB_ATTRIBUTE_LENGTH_ERROR)

	value := p.PathAttribute.Value
	if len(value) < 3 {
		return NewMessageError(eCode, eSubCode, value, "unreach header length is incorrect")
	}
	afi := binary.BigEndian.Uint16(value[0:2])
	safi := value[2]
	value = value[3:]
	for len(value) > 0 {
		prefix, err := routeFamilyPrefix(afi, safi)
		if err != nil {
			return NewMessageError(eCode, BGP_ERROR_SUB_ATTRIBUTE_FLAGS_ERROR, data[:p.PathAttribute.Len()], err.Error())
		}
		err = prefix.DecodeFromBytes(value)
		if err != nil {
			return err
		}
		if prefix.Len() > len(value) {
			return NewMessageError(eCode, eSubCode, data[:p.PathAttribute.Len()], "prefix length is incorrect")
		}
		value = value[prefix.Len():]
		p.Value = append(p.Value, prefix)
	}
	return nil
}

func (p *PathAttributeMpUnreachNLRI) Serialize() ([]byte, error) {
	buf := make([]byte, 3)
	afi := p.Value[0].AFI()
	safi := p.Value[0].SAFI()
	binary.BigEndian.PutUint16(buf, afi)
	buf[2] = safi
	for _, prefix := range p.Value {
		pbuf, err := prefix.Serialize()
		if err != nil {
			return nil, err
		}
		buf = append(buf, pbuf...)
	}
	p.PathAttribute.Value = buf
	return p.PathAttribute.Serialize()
}

func NewPathAttributeMpUnreachNLRI(nlri []AddrPrefixInterface) *PathAttributeMpUnreachNLRI {
	t := BGP_ATTR_TYPE_MP_UNREACH_NLRI
	return &PathAttributeMpUnreachNLRI{
		PathAttribute{
			Flags:  pathAttrFlags[t],
			Type:   t,
			Length: 0,
			Value:  nil},
		nlri,
	}
}

type ExtendedCommunityInterface interface {
	Serialize() ([]byte, error)
}

type TwoOctetAsSpecificExtended struct {
	SubType    uint8
	AS         uint16
	LocalAdmin uint32
}

func (e *TwoOctetAsSpecificExtended) Serialize() ([]byte, error) {
	buf := make([]byte, 8)
	buf[0] = 0x00
	buf[1] = e.SubType
	binary.BigEndian.PutUint16(buf[2:], e.AS)
	binary.BigEndian.PutUint32(buf[4:], e.LocalAdmin)
	return buf, nil
}

type IPv4AddressSpecificExtended struct {
	SubType    uint8
	IPv4       net.IP
	LocalAdmin uint16
}

func (e *IPv4AddressSpecificExtended) Serialize() ([]byte, error) {
	buf := make([]byte, 8)
	buf[0] = 0x01
	buf[1] = e.SubType
	copy(buf[2:6], e.IPv4)
	binary.BigEndian.PutUint16(buf[6:], e.LocalAdmin)
	return buf, nil
}

type FourOctetAsSpecificExtended struct {
	SubType    uint8
	AS         uint32
	LocalAdmin uint16
}

func (e *FourOctetAsSpecificExtended) Serialize() ([]byte, error) {
	buf := make([]byte, 8)
	buf[0] = 0x02
	buf[1] = e.SubType
	binary.BigEndian.PutUint32(buf[2:], e.AS)
	binary.BigEndian.PutUint16(buf[6:], e.LocalAdmin)
	return buf, nil
}

type OpaqueExtended struct {
	Value []byte
}

func (e *OpaqueExtended) Serialize() ([]byte, error) {
	buf := make([]byte, 8)
	buf[0] = 0x03
	copy(buf[1:], e.Value)
	return buf, nil
}

type UnknownExtended struct {
	Type  BGPAttrType
	Value []byte
}

func (e *UnknownExtended) Serialize() ([]byte, error) {
	buf := make([]byte, 8)
	buf[0] = uint8(e.Type)
	copy(buf[1:], e.Value)
	return buf, nil
}

type PathAttributeExtendedCommunities struct {
	PathAttribute
	Value []ExtendedCommunityInterface
}

func parseExtended(data []byte) ExtendedCommunityInterface {
	typehigh := data[0] & ^uint8(0x40)
	switch typehigh {
	case 0:
		e := &TwoOctetAsSpecificExtended{}
		e.SubType = data[1]
		e.AS = binary.BigEndian.Uint16(data[2:4])
		e.LocalAdmin = binary.BigEndian.Uint32(data[4:8])
		return e
	case 1:
		e := &IPv4AddressSpecificExtended{}
		e.SubType = data[1]
		e.IPv4 = data[2:6]
		e.LocalAdmin = binary.BigEndian.Uint16(data[6:8])
		return e
	case 2:
		e := &FourOctetAsSpecificExtended{}
		e.SubType = data[1]
		e.AS = binary.BigEndian.Uint32(data[2:6])
		e.LocalAdmin = binary.BigEndian.Uint16(data[6:8])
		return e
	case 3:
		e := &OpaqueExtended{}
		e.Value = data[1:8]
		return e
	}
	e := &UnknownExtended{}
	e.Type = BGPAttrType(data[0])
	e.Value = data[1:8]
	return e
}

func (p *PathAttributeExtendedCommunities) DecodeFromBytes(data []byte) error {
	err := p.PathAttribute.DecodeFromBytes(data)
	if err != nil {
		return err
	}
	if len(p.PathAttribute.Value)%8 != 0 {
		eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
		eSubCode := uint8(BGP_ERROR_SUB_ATTRIBUTE_LENGTH_ERROR)
		return NewMessageError(eCode, eSubCode, nil, "extendedcommunities length isn't correct")
	}
	value := p.PathAttribute.Value
	for len(value) >= 8 {
		e := parseExtended(value)
		p.Value = append(p.Value, e)
		value = value[8:]
	}
	return nil
}

func (p *PathAttributeExtendedCommunities) Serialize() ([]byte, error) {
	buf := make([]byte, 0)
	for _, p := range p.Value {
		ebuf, err := p.Serialize()
		if err != nil {
			return nil, err
		}
		buf = append(buf, ebuf...)
	}
	p.PathAttribute.Value = buf
	return p.PathAttribute.Serialize()
}

func NewPathAttributeExtendedCommunities(value []ExtendedCommunityInterface) *PathAttributeExtendedCommunities {
	t := BGP_ATTR_TYPE_EXTENDED_COMMUNITIES
	return &PathAttributeExtendedCommunities{
		PathAttribute: PathAttribute{
			Flags: pathAttrFlags[t],
			Type:  t,
		},
		Value: value,
	}
}

type PathAttributeAs4Path struct {
	PathAttribute
	Value []*As4PathParam
	DefaultAsPath
}

func (p *PathAttributeAs4Path) DecodeFromBytes(data []byte) error {
	err := p.PathAttribute.DecodeFromBytes(data)
	if err != nil {
		return err
	}
	eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
	eSubCode := uint8(BGP_ERROR_SUB_MALFORMED_ATTRIBUTE_LIST)
	v := p.PathAttribute.Value
	as4Bytes, err := p.DefaultAsPath.isValidAspath(p.PathAttribute.Value)
	if err != nil {
		return err
	}
	if as4Bytes == false {
		return NewMessageError(eCode, eSubCode, nil, "AS4 PATH param is malformed")
	}
	for len(v) > 0 {
		tuple := &As4PathParam{}
		tuple.DecodeFromBytes(v)
		p.Value = append(p.Value, tuple)
		if len(v) < tuple.Len() {
			return NewMessageError(eCode, eSubCode, nil, "AS4 PATH param is malformed")
		}
		v = v[tuple.Len():]
	}
	return nil
}

func (p *PathAttributeAs4Path) Serialize() ([]byte, error) {
	buf := make([]byte, 0)
	for _, v := range p.Value {
		vbuf, err := v.Serialize()
		if err != nil {
			return nil, err
		}
		buf = append(buf, vbuf...)
	}
	p.PathAttribute.Value = buf
	return p.PathAttribute.Serialize()
}

func (p *PathAttributeAs4Path) MarshalJSON() ([]byte, error) {
	aslist := make([]uint32, 0)
	for _, a := range p.Value {
		aslist = append(aslist, a.AS...)
	}
	return json.Marshal(struct {
		Type   string
		AsPath []uint32
	}{
		Type:   p.Type.String(),
		AsPath: aslist,
	})
}

func NewPathAttributeAs4Path(value []*As4PathParam) *PathAttributeAs4Path {
	t := BGP_ATTR_TYPE_AS4_PATH
	return &PathAttributeAs4Path{
		PathAttribute: PathAttribute{
			Flags: pathAttrFlags[t],
			Type:  t,
		},
		Value: value,
	}
}

type PathAttributeAs4Aggregator struct {
	PathAttribute
	Value PathAttributeAggregatorParam
}

func (p *PathAttributeAs4Aggregator) DecodeFromBytes(data []byte) error {
	err := p.PathAttribute.DecodeFromBytes(data)
	if err != nil {
		return err
	}
	if len(p.PathAttribute.Value) != 8 {
		eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
		eSubCode := uint8(BGP_ERROR_SUB_MALFORMED_ATTRIBUTE_LIST)
		return NewMessageError(eCode, eSubCode, nil, "AS4 Aggregator length is incorrect")
	}
	p.Value.AS = binary.BigEndian.Uint32(p.PathAttribute.Value[0:4])
	p.Value.Address = p.PathAttribute.Value[4:]
	return nil
}

func (p *PathAttributeAs4Aggregator) Serialize() ([]byte, error) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf[0:], p.Value.AS)
	copy(buf[4:], p.Value.Address)
	p.PathAttribute.Value = buf
	return p.PathAttribute.Serialize()
}

func NewPathAttributeAs4Aggregator(as uint32, address string) *PathAttributeAs4Aggregator {
	t := BGP_ATTR_TYPE_AS4_AGGREGATOR
	return &PathAttributeAs4Aggregator{
		PathAttribute: PathAttribute{
			Flags: pathAttrFlags[t],
			Type:  t,
		},
		Value: PathAttributeAggregatorParam{
			AS:      as,
			Address: net.ParseIP(address),
		},
	}
}

type PathAttributeUnknown struct {
	PathAttribute
}

func getPathAttribute(data []byte) (PathAttributeInterface, error) {
	if len(data) < 1 {
		eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
		eSubCode := uint8(BGP_ERROR_SUB_MALFORMED_ATTRIBUTE_LIST)
		msg := "attribute type length is short"
		return nil, NewMessageError(eCode, eSubCode, nil, msg)
	}
	switch BGPAttrType(data[1]) {
	case BGP_ATTR_TYPE_ORIGIN:
		return &PathAttributeOrigin{}, nil
	case BGP_ATTR_TYPE_AS_PATH:
		return &PathAttributeAsPath{}, nil
	case BGP_ATTR_TYPE_NEXT_HOP:
		return &PathAttributeNextHop{}, nil
	case BGP_ATTR_TYPE_MULTI_EXIT_DISC:
		return &PathAttributeMultiExitDisc{}, nil
	case BGP_ATTR_TYPE_LOCAL_PREF:
		return &PathAttributeLocalPref{}, nil
	case BGP_ATTR_TYPE_ATOMIC_AGGREGATE:
		return &PathAttributeAtomicAggregate{}, nil
	case BGP_ATTR_TYPE_AGGREGATOR:
		return &PathAttributeAggregator{}, nil
	case BGP_ATTR_TYPE_COMMUNITIES:
		return &PathAttributeCommunities{}, nil
	case BGP_ATTR_TYPE_ORIGINATOR_ID:
		return &PathAttributeOriginatorId{}, nil
	case BGP_ATTR_TYPE_CLUSTER_LIST:
		return &PathAttributeClusterList{}, nil
	case BGP_ATTR_TYPE_MP_REACH_NLRI:
		return &PathAttributeMpReachNLRI{}, nil
	case BGP_ATTR_TYPE_MP_UNREACH_NLRI:
		return &PathAttributeMpUnreachNLRI{}, nil
	case BGP_ATTR_TYPE_EXTENDED_COMMUNITIES:
		return &PathAttributeExtendedCommunities{}, nil
	case BGP_ATTR_TYPE_AS4_PATH:
		return &PathAttributeAs4Path{}, nil
	case BGP_ATTR_TYPE_AS4_AGGREGATOR:
		return &PathAttributeAs4Aggregator{}, nil
	}
	return &PathAttributeUnknown{}, nil
}

type NLRInfo struct {
	IPAddrPrefix
}

func NewNLRInfo(length uint8, prefix string) *NLRInfo {
	return &NLRInfo{
		IPAddrPrefix: *NewIPAddrPrefix(length, prefix),
	}
}

type BGPUpdate struct {
	WithdrawnRoutesLen    uint16
	WithdrawnRoutes       []WithdrawnRoute
	TotalPathAttributeLen uint16
	PathAttributes        []PathAttributeInterface
	NLRI                  []NLRInfo
}

func (msg *BGPUpdate) DecodeFromBytes(data []byte) error {

	// cache error codes
	eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
	eSubCode := uint8(BGP_ERROR_SUB_MALFORMED_ATTRIBUTE_LIST)

	// check withdrawn route length
	if len(data) < 2 {
		msg := "message length isn't enough for withdrawn route length"
		e := NewMessageError(eCode, eSubCode, nil, msg)
		return e
	}

	msg.WithdrawnRoutesLen = binary.BigEndian.Uint16(data[0:2])
	data = data[2:]

	// check withdrawn route
	if len(data) < int(msg.WithdrawnRoutesLen) {
		msg := "withdrawn route length exceeds message length"
		e := NewMessageError(eCode, eSubCode, nil, msg)
		return e
	}

	for routelen := msg.WithdrawnRoutesLen; routelen > 0; {
		w := WithdrawnRoute{}
		err := w.DecodeFromBytes(data)
		if err != nil {
			return err
		}
		routelen -= uint16(w.Len())
		if len(data) < w.Len() {
			return NewMessageError(eCode, eSubCode, nil, "Withdrawn route length is short")
		}
		data = data[w.Len():]
		msg.WithdrawnRoutes = append(msg.WithdrawnRoutes, w)
	}

	// check path total attribute length
	if len(data) < 2 {
		msg := "message length isn't enough for path total attribute length"
		e := NewMessageError(eCode, eSubCode, nil, msg)
		return e
	}

	msg.TotalPathAttributeLen = binary.BigEndian.Uint16(data[0:2])
	data = data[2:]

	// check path attribute
	if len(data) < int(msg.TotalPathAttributeLen) {
		msg := "path total attribute length exceeds message length"
		e := NewMessageError(eCode, eSubCode, nil, msg)
		return e
	}

	for pathlen := msg.TotalPathAttributeLen; pathlen > 0; {
		p, err := getPathAttribute(data)
		if err != nil {
			return err
		}
		err = p.DecodeFromBytes(data)
		if err != nil {
			return err
		}
		pathlen -= uint16(p.Len())
		if len(data) < p.Len() {
			return NewMessageError(eCode, BGP_ERROR_SUB_ATTRIBUTE_LENGTH_ERROR, data, "attribute length is short")
		}
		data = data[p.Len():]
		msg.PathAttributes = append(msg.PathAttributes, p)
	}

	for restlen := len(data); restlen > 0; {
		n := NLRInfo{}
		err := n.DecodeFromBytes(data)
		if err != nil {
			return err
		}
		restlen -= n.Len()
		if len(data) < n.Len() {
			return NewMessageError(eCode, BGP_ERROR_SUB_INVALID_NETWORK_FIELD, nil, "NLRI length is short")
		}
		data = data[n.Len():]
		msg.NLRI = append(msg.NLRI, n)
	}

	return nil
}

func (msg *BGPUpdate) Serialize() ([]byte, error) {
	wbuf := make([]byte, 2)
	for _, w := range msg.WithdrawnRoutes {
		onewbuf, err := w.Serialize()
		if err != nil {
			return nil, err
		}
		wbuf = append(wbuf, onewbuf...)
	}
	binary.BigEndian.PutUint16(wbuf, uint16(len(wbuf)-2))

	pbuf := make([]byte, 2)
	for _, p := range msg.PathAttributes {
		onepbuf, err := p.Serialize()
		if err != nil {
			return nil, err
		}
		pbuf = append(pbuf, onepbuf...)
	}
	binary.BigEndian.PutUint16(pbuf, uint16(len(pbuf)-2))

	buf := append(wbuf, pbuf...)
	for _, n := range msg.NLRI {
		nbuf, err := n.Serialize()
		if err != nil {
			return nil, err
		}
		buf = append(buf, nbuf...)
	}
	return buf, nil
}

func NewBGPUpdateMessage(withdrawnRoutes []WithdrawnRoute, pathattrs []PathAttributeInterface, nlri []NLRInfo) *BGPMessage {
	return &BGPMessage{
		Header: BGPHeader{Type: BGP_MSG_UPDATE},
		Body:   &BGPUpdate{0, withdrawnRoutes, 0, pathattrs, nlri},
	}
}

type BGPNotification struct {
	ErrorCode    uint8
	ErrorSubcode uint8
	Data         []byte
}

func (msg *BGPNotification) DecodeFromBytes(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("Not all Notificaiton bytes available")
	}
	msg.ErrorCode = data[0]
	msg.ErrorSubcode = data[1]
	if len(data) > 2 {
		msg.Data = data[2:]
	}
	return nil
}

func (msg *BGPNotification) Serialize() ([]byte, error) {
	buf := make([]byte, 2)
	buf[0] = msg.ErrorCode
	buf[1] = msg.ErrorSubcode
	buf = append(buf, msg.Data...)
	return buf, nil
}

func NewBGPNotificationMessage(errcode uint8, errsubcode uint8, data []byte) *BGPMessage {
	return &BGPMessage{
		Header: BGPHeader{Type: BGP_MSG_NOTIFICATION},
		Body:   &BGPNotification{errcode, errsubcode, data},
	}
}

type BGPKeepAlive struct {
}

func (msg *BGPKeepAlive) DecodeFromBytes(data []byte) error {
	return nil
}

func (msg *BGPKeepAlive) Serialize() ([]byte, error) {
	return nil, nil
}

func NewBGPKeepAliveMessage() *BGPMessage {
	return &BGPMessage{
		Header: BGPHeader{Len: 19, Type: BGP_MSG_KEEPALIVE},
		Body:   &BGPKeepAlive{},
	}
}

type BGPRouteRefresh struct {
	AFI         uint16
	Demarcation uint8
	SAFI        uint8
}

func (msg *BGPRouteRefresh) DecodeFromBytes(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("Not all RouteRefresh bytes available")
	}
	msg.AFI = binary.BigEndian.Uint16(data[0:2])
	msg.Demarcation = data[2]
	msg.SAFI = data[3]
	return nil
}

func (msg *BGPRouteRefresh) Serialize() ([]byte, error) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint16(buf[0:2], msg.AFI)
	buf[2] = msg.Demarcation
	buf[3] = msg.SAFI
	return buf, nil
}

func NewBGPRouteRefreshMessage(afi uint16, demarcation uint8, safi uint8) *BGPMessage {
	return &BGPMessage{
		Header: BGPHeader{Type: BGP_MSG_ROUTE_REFRESH},
		Body:   &BGPRouteRefresh{afi, demarcation, safi},
	}
}

type BGPBody interface {
	DecodeFromBytes([]byte) error
	Serialize() ([]byte, error)
}

const (
	BGP_HEADER_LENGTH      = 19
	BGP_MAX_MESSAGE_LENGTH = 4096
)

type BGPHeader struct {
	Marker []byte
	Len    uint16
	Type   uint8
}

func (msg *BGPHeader) DecodeFromBytes(data []byte) error {
	// minimum BGP message length
	if uint16(len(data)) < 19 {
		return fmt.Errorf("Not all BGP message header")
	}
	msg.Len = binary.BigEndian.Uint16(data[16:18])
	msg.Type = data[18]
	return nil
}

func (msg *BGPHeader) Serialize() ([]byte, error) {
	buf := make([]byte, 19)
	for i, _ := range buf[:16] {
		buf[i] = 0xff
	}
	binary.BigEndian.PutUint16(buf[16:18], msg.Len)
	buf[18] = msg.Type
	return buf, nil
}

type BGPMessage struct {
	Header BGPHeader
	Body   BGPBody
}

func parseBody(h *BGPHeader, data []byte) (*BGPMessage, error) {
	if len(data) < int(h.Len)-BGP_HEADER_LENGTH {
		return nil, fmt.Errorf("Not all BGP message bytes available")
	}
	msg := &BGPMessage{Header: *h}

	switch msg.Header.Type {
	case BGP_MSG_OPEN:
		msg.Body = &BGPOpen{}
	case BGP_MSG_UPDATE:
		msg.Body = &BGPUpdate{}
	case BGP_MSG_NOTIFICATION:
		msg.Body = &BGPNotification{}
	case BGP_MSG_KEEPALIVE:
		msg.Body = &BGPKeepAlive{}
	case BGP_MSG_ROUTE_REFRESH:
		msg.Body = &BGPRouteRefresh{}
	default:
		return nil, NewMessageError(BGP_ERROR_MESSAGE_HEADER_ERROR, BGP_ERROR_SUB_BAD_MESSAGE_TYPE, nil, "unknown message type")
	}
	err := msg.Body.DecodeFromBytes(data)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func ParseBGPMessage(data []byte) (*BGPMessage, error) {
	h := &BGPHeader{}
	err := h.DecodeFromBytes(data)
	if err != nil {
		return nil, err
	}
	return parseBody(h, data[19:h.Len])
}

func ParseBGPBody(h *BGPHeader, data []byte) (*BGPMessage, error) {
	return parseBody(h, data)
}

func (msg *BGPMessage) Serialize() ([]byte, error) {
	b, err := msg.Body.Serialize()
	if err != nil {
		return nil, err
	}
	if msg.Header.Len == 0 {
		msg.Header.Len = 19 + uint16(len(b))
	}
	h, err := msg.Header.Serialize()
	if err != nil {
		return nil, err
	}
	return append(h, b...), nil
}

type MessageError struct {
	TypeCode    uint8
	SubTypeCode uint8
	Data        []byte
	Message     string
}

func NewMessageError(typeCode, subTypeCode uint8, data []byte, msg string) error {
	return &MessageError{
		TypeCode:    typeCode,
		SubTypeCode: subTypeCode,
		Data:        data,
		Message:     msg,
	}
}

func (e *MessageError) Error() string {
	return e.Message
}

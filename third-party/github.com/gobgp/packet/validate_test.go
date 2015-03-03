package bgp

import (
	"encoding/binary"
	"github.com/gopher-net/gopher-net/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func bgpupdate() *BGPMessage {
	aspath := []AsPathParamInterface{
		NewAsPathParam(2, []uint16{65001}),
	}

	p := []PathAttributeInterface{
		NewPathAttributeOrigin(1),
		NewPathAttributeAsPath(aspath),
		NewPathAttributeNextHop("192.168.1.1"),
	}

	n := []NLRInfo{*NewNLRInfo(24, "10.10.10.0")}
	return NewBGPUpdateMessage(nil, p, n)
}

func Test_Validate_OK(t *testing.T) {
	assert := assert.New(t)
	message := bgpupdate().Body.(*BGPUpdate)
	res, err := ValidateUpdateMsg(message)
	assert.Equal(true, res)
	assert.NoError(err)

}

func Test_Validate_wellknown_but_nontransitive(t *testing.T) {
	assert := assert.New(t)
	message := bgpupdate().Body.(*BGPUpdate)

	originBytes := []byte{0, 1, 1, 1} // 0 means Flags
	origin := &PathAttributeOrigin{}
	origin.DecodeFromBytes(originBytes)
	message.PathAttributes[0] = origin

	res, err := ValidateUpdateMsg(message)
	assert.Equal(false, res)
	assert.Error(err)
	e := err.(*MessageError)
	assert.Equal(BGP_ERROR_UPDATE_MESSAGE_ERROR, e.TypeCode)
	assert.Equal(BGP_ERROR_SUB_ATTRIBUTE_FLAGS_ERROR, e.SubTypeCode)
	assert.Equal(originBytes, e.Data)
}

func Test_Validate_wellknown_but_partial(t *testing.T) {
	assert := assert.New(t)
	message := bgpupdate().Body.(*BGPUpdate)

	originBytes := []byte{BGP_ATTR_FLAG_PARTIAL, 1, 1, 1}
	origin := &PathAttributeOrigin{}
	origin.DecodeFromBytes(originBytes)
	message.PathAttributes[0] = origin

	res, err := ValidateUpdateMsg(message)
	assert.Equal(false, res)
	assert.Error(err)
	e := err.(*MessageError)
	assert.Equal(BGP_ERROR_UPDATE_MESSAGE_ERROR, e.TypeCode)
	assert.Equal(BGP_ERROR_SUB_ATTRIBUTE_FLAGS_ERROR, e.SubTypeCode)
	assert.Equal(originBytes, e.Data)
}

func Test_Validate_optional_nontransitive_but_partial(t *testing.T) {
	assert := assert.New(t)
	message := bgpupdate().Body.(*BGPUpdate)
	f := BGP_ATTR_FLAG_OPTIONAL | BGP_ATTR_FLAG_PARTIAL
	originBytes := []byte{byte(f), 1, 1, 1}
	origin := &PathAttributeOrigin{}
	origin.DecodeFromBytes(originBytes)
	message.PathAttributes[0] = origin

	res, err := ValidateUpdateMsg(message)
	assert.Equal(false, res)
	assert.Error(err)
	e := err.(*MessageError)
	assert.Equal(BGP_ERROR_UPDATE_MESSAGE_ERROR, e.TypeCode)
	assert.Equal(BGP_ERROR_SUB_ATTRIBUTE_FLAGS_ERROR, e.SubTypeCode)
	assert.Equal(originBytes, e.Data)
}

func Test_Validate_flag_mismatch(t *testing.T) {
	assert := assert.New(t)
	message := bgpupdate().Body.(*BGPUpdate)
	f := BGP_ATTR_FLAG_OPTIONAL
	// origin needs to be well-known
	originBytes := []byte{byte(f), 1, 1, 1}
	origin := &PathAttributeOrigin{}
	origin.DecodeFromBytes(originBytes)
	message.PathAttributes[0] = origin

	res, err := ValidateUpdateMsg(message)
	assert.Equal(false, res)
	assert.Error(err)
	e := err.(*MessageError)
	assert.Equal(BGP_ERROR_UPDATE_MESSAGE_ERROR, e.TypeCode)
	assert.Equal(BGP_ERROR_SUB_ATTRIBUTE_FLAGS_ERROR, e.SubTypeCode)
	assert.Equal(originBytes, e.Data)
}

func Test_Validate_duplicate_attribute(t *testing.T) {
	assert := assert.New(t)
	message := bgpupdate().Body.(*BGPUpdate)
	// duplicate origin path attribute
	originBytes := []byte{pathAttrFlags[BGP_ATTR_TYPE_ORIGIN], 1, 1, 1}
	origin := &PathAttributeOrigin{}
	origin.DecodeFromBytes(originBytes)
	message.PathAttributes = append(message.PathAttributes, origin)

	res, err := ValidateUpdateMsg(message)
	assert.Equal(false, res)
	assert.Error(err)
	e := err.(*MessageError)
	assert.Equal(BGP_ERROR_UPDATE_MESSAGE_ERROR, e.TypeCode)
	assert.Equal(BGP_ERROR_SUB_MALFORMED_ATTRIBUTE_LIST, e.SubTypeCode)
	assert.Nil(e.Data)
}

func Test_Validate_mandatory_missing(t *testing.T) {
	assert := assert.New(t)
	message := bgpupdate().Body.(*BGPUpdate)
	message.PathAttributes = message.PathAttributes[1:]
	res, err := ValidateUpdateMsg(message)
	assert.Equal(false, res)
	assert.Error(err)
	e := err.(*MessageError)
	assert.Equal(BGP_ERROR_UPDATE_MESSAGE_ERROR, e.TypeCode)
	assert.Equal(BGP_ERROR_SUB_MISSING_WELL_KNOWN_ATTRIBUTE, e.SubTypeCode)
	missing, _ := binary.Uvarint(e.Data)
	assert.Equal(1, missing)
}

func Test_Validate_mandatory_missing_nocheck(t *testing.T) {
	assert := assert.New(t)
	message := bgpupdate().Body.(*BGPUpdate)
	message.PathAttributes = message.PathAttributes[1:]
	message.NLRI = nil

	res, err := ValidateUpdateMsg(message)
	assert.Equal(true, res)
	assert.NoError(err)
}

func Test_Validate_invalid_origin(t *testing.T) {
	assert := assert.New(t)
	message := bgpupdate().Body.(*BGPUpdate)
	// origin needs to be well-known
	originBytes := []byte{pathAttrFlags[BGP_ATTR_TYPE_ORIGIN], 1, 1, 5}
	origin := &PathAttributeOrigin{}
	origin.DecodeFromBytes(originBytes)
	message.PathAttributes[0] = origin

	res, err := ValidateUpdateMsg(message)
	assert.Equal(false, res)
	assert.Error(err)
	e := err.(*MessageError)
	assert.Equal(BGP_ERROR_UPDATE_MESSAGE_ERROR, e.TypeCode)
	assert.Equal(BGP_ERROR_SUB_INVALID_ORIGIN_ATTRIBUTE, e.SubTypeCode)
	assert.Equal(originBytes, e.Data)
}

func Test_Validate_invalid_nexthop_zero(t *testing.T) {
	assert := assert.New(t)
	message := bgpupdate().Body.(*BGPUpdate)

	// invalid nexthop
	addr := net.ParseIP("0.0.0.1").To4()
	nexthopBytes := []byte{pathAttrFlags[BGP_ATTR_TYPE_NEXT_HOP], 3, 4}
	nexthopBytes = append(nexthopBytes, addr...)
	nexthop := &PathAttributeNextHop{}
	nexthop.DecodeFromBytes(nexthopBytes)
	message.PathAttributes[2] = nexthop

	res, err := ValidateUpdateMsg(message)
	assert.Equal(false, res)
	assert.Error(err)
	e := err.(*MessageError)
	assert.Equal(BGP_ERROR_UPDATE_MESSAGE_ERROR, e.TypeCode)
	assert.Equal(BGP_ERROR_SUB_INVALID_NEXT_HOP_ATTRIBUTE, e.SubTypeCode)
	assert.Equal(nexthopBytes, e.Data)
}

func Test_Validate_invalid_nexthop_lo(t *testing.T) {
	assert := assert.New(t)
	message := bgpupdate().Body.(*BGPUpdate)

	// invalid nexthop
	addr := net.ParseIP("127.0.0.1").To4()
	nexthopBytes := []byte{pathAttrFlags[BGP_ATTR_TYPE_NEXT_HOP], 3, 4}
	nexthopBytes = append(nexthopBytes, addr...)
	nexthop := &PathAttributeNextHop{}
	nexthop.DecodeFromBytes(nexthopBytes)
	message.PathAttributes[2] = nexthop

	res, err := ValidateUpdateMsg(message)
	assert.Equal(false, res)
	assert.Error(err)
	e := err.(*MessageError)
	assert.Equal(BGP_ERROR_UPDATE_MESSAGE_ERROR, e.TypeCode)
	assert.Equal(BGP_ERROR_SUB_INVALID_NEXT_HOP_ATTRIBUTE, e.SubTypeCode)
	assert.Equal(nexthopBytes, e.Data)
}

func Test_Validate_invalid_nexthop_de(t *testing.T) {
	assert := assert.New(t)
	message := bgpupdate().Body.(*BGPUpdate)

	// invalid nexthop
	addr := net.ParseIP("224.0.0.1").To4()
	nexthopBytes := []byte{pathAttrFlags[BGP_ATTR_TYPE_NEXT_HOP], 3, 4}
	nexthopBytes = append(nexthopBytes, addr...)
	nexthop := &PathAttributeNextHop{}
	nexthop.DecodeFromBytes(nexthopBytes)
	message.PathAttributes[2] = nexthop

	res, err := ValidateUpdateMsg(message)
	assert.Equal(false, res)
	assert.Error(err)
	e := err.(*MessageError)
	assert.Equal(BGP_ERROR_UPDATE_MESSAGE_ERROR, e.TypeCode)
	assert.Equal(BGP_ERROR_SUB_INVALID_NEXT_HOP_ATTRIBUTE, e.SubTypeCode)
	assert.Equal(nexthopBytes, e.Data)

}

func Test_Validate_unrecognized_well_known(t *testing.T) {

	assert := assert.New(t)
	message := bgpupdate().Body.(*BGPUpdate)
	f := BGP_ATTR_FLAG_TRANSITIVE
	unknownBytes := []byte{byte(f), 30, 1, 1}
	unknown := &PathAttributeUnknown{}
	unknown.DecodeFromBytes(unknownBytes)
	message.PathAttributes = append(message.PathAttributes, unknown)

	res, err := ValidateUpdateMsg(message)
	assert.Equal(false, res)
	assert.Error(err)
	e := err.(*MessageError)
	assert.Equal(BGP_ERROR_UPDATE_MESSAGE_ERROR, e.TypeCode)
	assert.Equal(BGP_ERROR_SUB_UNRECOGNIZED_WELL_KNOWN_ATTRIBUTE, e.SubTypeCode)
	assert.Equal(unknownBytes, e.Data)
}

package bgp

import (
	"encoding/binary"
	"github.com/gopher-net/gopher-net/configuration"
	"net"
	"strconv"
)

// Validator for BGPUpdate
func ValidateUpdateMsg(m *BGPUpdate) (bool, error) {
	eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
	eSubCodeAttrList := uint8(BGP_ERROR_SUB_MALFORMED_ATTRIBUTE_LIST)
	eSubCodeFlagsError := uint8(BGP_ERROR_SUB_ATTRIBUTE_FLAGS_ERROR)
	eSubCodeMissing := uint8(BGP_ERROR_SUB_MISSING_WELL_KNOWN_ATTRIBUTE)

	seen := make(map[BGPAttrType]PathAttributeInterface)
	// check path attribute
	for _, a := range m.PathAttributes {

		// check attribute flags
		ok, eMsg := ValidateFlags(a.getType(), a.getFlags())
		if !ok {
			data, _ := a.Serialize()
			return false, NewMessageError(eCode, eSubCodeFlagsError, data, eMsg)
		}

		// check duplication
		if _, ok := seen[a.getType()]; !ok {
			seen[a.getType()] = a
		} else {
			eMsg := "the path attribute apears twice. Type : " + strconv.Itoa(int(a.getType()))
			return false, NewMessageError(eCode, eSubCodeAttrList, nil, eMsg)
		}

		// check specific path attribute
		ok, e := ValidateAttribute(a)
		if !ok {
			return false, e
		}
	}

	if len(m.NLRI) > 0 {
		// check the existence of well-known mandatory attributes
		exist := func(attrs []BGPAttrType) (bool, BGPAttrType) {
			for _, attr := range attrs {
				_, ok := seen[attr]
				if !ok {
					return false, attr
				}
			}
			return true, 0
		}
		mandatory := []BGPAttrType{BGP_ATTR_TYPE_ORIGIN, BGP_ATTR_TYPE_AS_PATH, BGP_ATTR_TYPE_NEXT_HOP}
		if ok, t := exist(mandatory); !ok {
			eMsg := "well-known mandatory attributes are not present. type : " + strconv.Itoa(int(t))
			data := []byte{byte(t)}
			return false, NewMessageError(eCode, eSubCodeMissing, data, eMsg)
		}
	}
	return true, nil
}

func ValidateAttribute(a PathAttributeInterface) (bool, error) {

	eCode := uint8(BGP_ERROR_UPDATE_MESSAGE_ERROR)
	eSubCodeBadOrigin := uint8(BGP_ERROR_SUB_INVALID_ORIGIN_ATTRIBUTE)
	eSubCodeBadNextHop := uint8(BGP_ERROR_SUB_INVALID_NEXT_HOP_ATTRIBUTE)
	eSubCodeUnknown := uint8(BGP_ERROR_SUB_UNRECOGNIZED_WELL_KNOWN_ATTRIBUTE)

	switch p := a.(type) {
	case *PathAttributeOrigin:
		v := uint8(p.Value[0])
		if v != configuration.BGP_ORIGIN_ATTR_TYPE_IGP &&
			v != configuration.BGP_ORIGIN_ATTR_TYPE_EGP &&
			v != configuration.BGP_ORIGIN_ATTR_TYPE_INCOMPLETE {
			data, _ := a.Serialize()
			eMsg := "invalid origin attribute. value : " + strconv.Itoa(int(v))
			return false, NewMessageError(eCode, eSubCodeBadOrigin, data, eMsg)
		}
	case *PathAttributeNextHop:

		isZero := func(ip net.IP) bool {
			res := ip[0] & 0xff
			return res == 0x00
		}

		isClassDorE := func(ip net.IP) bool {
			res := ip[0] & 0xe0
			return res == 0xe0
		}

		//check IP address represents host address
		if p.Value.IsLoopback() || isZero(p.Value) || isClassDorE(p.Value) {
			eMsg := "invalid nexthop address"
			data, _ := a.Serialize()
			return false, NewMessageError(eCode, eSubCodeBadNextHop, data, eMsg)
		}
	case *PathAttributeUnknown:
		if p.getFlags()&BGP_ATTR_FLAG_OPTIONAL == 0 {
			eMsg := "unrecognized well-known attribute"
			data, _ := a.Serialize()
			return false, NewMessageError(eCode, eSubCodeUnknown, data, eMsg)
		}
	}

	return true, nil

}

// validator for PathAttribute
func ValidateFlags(t BGPAttrType, flags uint8) (bool, string) {

	/*
	 * RFC 4271 P.17 For well-known attributes, the Transitive bit MUST be set to 1.
	 */
	if flags&BGP_ATTR_FLAG_OPTIONAL == 0 && flags&BGP_ATTR_FLAG_TRANSITIVE == 0 {
		eMsg := "well-known attribute must have transitive flag 1"
		return false, eMsg
	}
	/*
	 * RFC 4271 P.17 For well-known attributes and for optional non-transitive attributes,
	 * the Partial bit MUST be set to 0.
	 */
	if flags&BGP_ATTR_FLAG_OPTIONAL == 0 && flags&BGP_ATTR_FLAG_PARTIAL != 0 {
		eMsg := "well-known attribute must have partial bit 0"
		return false, eMsg
	}
	if flags&BGP_ATTR_FLAG_OPTIONAL != 0 && flags&BGP_ATTR_FLAG_TRANSITIVE == 0 && flags&BGP_ATTR_FLAG_PARTIAL != 0 {
		eMsg := "optional non-transitive attribute must have partial bit 0"
		return false, eMsg
	}

	// check flags are correct
	if f, ok := pathAttrFlags[t]; ok {
		if f != flags {
			eMsg := "flags are invalid. attribtue type : " + strconv.Itoa(int(t))
			return false, eMsg
		}
	}
	return true, ""
}

func ValidateBGPMessage(m *BGPMessage) error {
	if m.Header.Len > BGP_MAX_MESSAGE_LENGTH {
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, m.Header.Len)
		return NewMessageError(BGP_ERROR_MESSAGE_HEADER_ERROR, BGP_ERROR_SUB_BAD_MESSAGE_LENGTH, buf, "too long length")
	}
	return nil
}

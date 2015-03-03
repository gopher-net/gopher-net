package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"regexp"

	log "github.com/gopher-net/gopher-net/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/gopher-net/gopher-net/Godeps/_workspace/src/gopkg.in/v1/yaml"
)

var ValidV4RegEx, _ = regexp.Compile(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`)

func CreateUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := rand.Read(uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	uuid[8] = 0x80
	uuid[4] = 0x40
	return hex.EncodeToString(uuid), nil
}

func GetLocalHostIPs() ([]net.IP, error) {
	ifaces, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	var ips []net.IP
	for _, iface := range ifaces {
		if ipnet, ok := iface.(*net.IPNet); ok {
			ips = append(ips, ipnet.IP)
		}
	}
	return ips, nil
}

func PrettyPrint(data interface{}, format string) {
	var p []byte
	var err error
	switch format {
	case "json":
		p, err = json.MarshalIndent(data, "", "\t")
	case "yaml":
		p, err = yaml.Marshal(data)
	default:
		log.Printf("unsupported format: %s", format)
		return
	}
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("%s", p)
}

// Return the first IP address that is not a loopback
func GetNodeIP() (addresses []string, err error) {
	addrs, _ := net.InterfaceAddrs()
	for _, i := range addrs {
		ipnet, err := i.(*net.IPNet)

		if !err {
			log.Fatal("no ip address found: ", i)
		}
		ip4 := ipnet.IP.To4()

		if !ip4.IsLoopback() && ip4 != nil {
			addr := ip4.String()
			addresses = append(addresses, addr)
			break
		}
	}
	return
}

// verify a new bgp neighbor is not a local ip
func (daemon *Daemon) checkBgpPeerAddr(neighborIp string) bool {
	p, _ := GetLocalHostIPs()
	for i := 0; i < len(p); i++ {
		localIP := p[i].String()
		if localIP == neighborIp {
			log.Errorf("bgp peer %s is equal to a local ip address %s , ignoring. \n", neighborIp, p[i])
			return false
		}
	}
	return true
}

// verify a new bgp neighbor is not already defined
func (daemon *Daemon) checkIfPeerExists(neighborIp string) bool {
	_, exists := daemon.neighborMap[neighborIp]
	if exists {
		log.Debugf("Failed to add new BGP neighbor, a node with a peer address [ %s ] already exists.", neighborIp)
		return false
	}
	return true
}

func isValidIp(s string) bool {
	return ValidV4RegEx.MatchString(s)
}

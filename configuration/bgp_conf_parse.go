package configuration

import (
	"fmt"
	"strings"

	"github.com/gopher-net/gopher-net/Godeps/_workspace/src/code.google.com/p/gcfg"
	"github.com/gopher-net/gopher-net/Godeps/_workspace/src/github.com/BurntSushi/toml"
)

const (
	DEFAULT_HOLDTIME                  = 90
	DEFAULT_IDLE_HOLDTIME_AFTER_RESET = 30
)

func ReadConfigfileServe(path string, configCh chan BgpType, reloadCh chan bool) {
	for {
		<-reloadCh

		b := BgpType{}
		_, err := toml.DecodeFile(path, &b)
		if err == nil {
			// TODO: validate configuration
			for i, _ := range b.NeighborList {
				SetNeighborTypeDefault(&b.NeighborList[i])
			}
		}
		configCh <- b
	}
}

func inSlice(n NeighborType, b []NeighborType) bool {
	for _, nb := range b {
		if nb.NeighborAddress.String() == n.NeighborAddress.String() {
			return true
		}
	}
	return false
}

func setTimersTypeDefault(timersT *TimersType) {
	if timersT.HoldTime == 0 {
		timersT.HoldTime = float64(DEFAULT_HOLDTIME)
	}
	if timersT.KeepaliveInterval == 0 {
		timersT.KeepaliveInterval = timersT.HoldTime / 3
	}
	if timersT.IdleHoldTImeAfterReset == 0 {
		timersT.IdleHoldTImeAfterReset = float64(DEFAULT_IDLE_HOLDTIME_AFTER_RESET)
	}
}

func SetNeighborTypeDefault(neighborT *NeighborType) {
	setTimersTypeDefault(&neighborT.Timers)
}

// Below is old
type BgpConfig struct {
	BGP_Local_Address    string
	BGP_Peers            []string
	BGP_Route_Reflectors []string
	BGP_As_Number        int
	BGP_Version          int
	BGP_HoldTime         int
	BGP_Router_Id        int
}

type OvsConfig struct {
	Ovs_Endpoints []string
}

type DockerBridgeConfig struct {
	Docker_Bridge string
}

type NetworkConfig struct {
	// TODO Change to slices type net.IPAddr
	Local_Subnets []string
}

type KvClusterConfig struct {
	Leader_Node    string
	Candidate_Node string
}

type Config struct {
	BGP               BgpConfig
	OpenvSwitch       OvsConfig
	Docker_Bridge     DockerBridgeConfig
	Container_Subnets NetworkConfig
	Kv_Cluster        KvClusterConfig
}

func ParseConfig() (Config, error) {
	var config Config
	// TODO: use absolute path
	configFilePath := "../sample_config.conf"
	err := gcfg.ReadFileInto(&config, configFilePath)
	if err != nil {
		fmt.Println("error parsing config: ", err)
	}
	return config, err
}

func ReadConfig(file string, config *Config) error {
	return gcfg.ReadFileInto(config, file)
}

func ParseBgpPeersConfig() ([]string, error) {
	var config Config

	configFilePath := "./sample_config.conf"
	err := gcfg.ReadFileInto(&config, configFilePath)
	if err != nil {
		fmt.Println("error parsing config: ", err)
	}
	// If there are more then one peer, split to a slice on commas
	peers := strings.Split((config.BGP.BGP_Peers[0]), ",")
	if len(peers) > 1 {
		return peers, err
	}
	// If only one peer then return the single val
	return config.BGP.BGP_Peers, err
}

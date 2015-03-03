package main

import (
	"github.com/gopher-net/gopher-net/api"
	"github.com/gopher-net/gopher-net/configuration"
	"github.com/gopher-net/gopher-net/daemon"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	log "github.com/gopher-net/gopher-net/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	flags "github.com/gopher-net/gopher-net/Godeps/_workspace/src/github.com/jessevdk/go-flags"
	bgp "github.com/gopher-net/gopher-net/third-party/github.com/gobgp/packet"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP)

	var opts struct {
		ConfigFile string `short:"f" long:"config-file" description:"specifying a config file"`
		LogLevel   string `short:"l" long:"log-level" description:"specifying log level"`
		LogJson    bool   `shot:"j" long:"log-json" description:"use json format for logging"`
	}
	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}
	// set logging levels, debug is the default if not specified
	switch opts.LogLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
	log.Info("Logging level is: ", log.GetLevel())
	log.SetOutput(os.Stderr)
	if opts.LogJson {
		log.SetFormatter(&log.JSONFormatter{})
	}
	if opts.ConfigFile == "" {
		opts.ConfigFile = "./bgpd.conf"
	}
	configCh := make(chan configuration.BgpType)
	reloadCh := make(chan bool)
	// read in config files
	go configuration.ReadConfigfileServe(opts.ConfigFile, configCh, reloadCh)
	reloadCh <- true
	// start the BGP daemon
	bgpDaemon := daemon.NewBgpDaemon(bgp.BGP_PORT)
	go bgpDaemon.Serve()
	// start REST server
	restServer := api.NewRestServer(api.REST_PORT, bgpDaemon.RestReqCh)
	go restServer.Serve()
	// listen for config changes
	var bgpConfig *configuration.BgpType = nil
	for {
		select {
		case newConfig := <-configCh:
			var added []configuration.NeighborType
			var deleted []configuration.NeighborType
			if bgpConfig == nil {
				bgpDaemon.SetGlobalType(newConfig.Global)
				bgpConfig = &newConfig
				added = newConfig.NeighborList
				deleted = []configuration.NeighborType{}
			}
			for _, p := range added {
				log.Infof("Peer %v is added", p.NeighborAddress)
				bgpDaemon.NeighborAdd(p)
			}
			for _, p := range deleted {
				log.Infof("Peer %v is deleted", p.NeighborAddress)
				bgpDaemon.NeighborDelete(p)
			}
		case sig := <-sigCh:
			switch sig {
			case syscall.SIGHUP:
				log.Info("relaod the config file")
				reloadCh <- true
			}
		}
	}
}

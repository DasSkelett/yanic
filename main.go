package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/monitormap/micro-daemon/data"
	"github.com/monitormap/micro-daemon/models"
	"github.com/monitormap/micro-daemon/respond"
	"github.com/monitormap/micro-daemon/websocketserver"
)

var (
	configFile       string
	config           *models.Config
	wsserverForNodes *websocketserver.Server
	multiCollector   *respond.MultiCollector
	statsDb          *StatsDb
	nodes            = models.NewNodes()
	//aliases          = models.NewNodes()
)

func main() {
	flag.StringVar(&configFile, "config", "config.yml", "path of configuration file (default:config.yaml)")
	flag.Parse()
	config = models.ConfigReadFile(configFile)

	if config.Nodes.Enable {
		go nodes.Saver(config)
	}

	if config.Webserver.Enable {
		if config.Webserver.WebsocketNode {
			wsserverForNodes = websocketserver.NewServer("/nodes")
			go wsserverForNodes.Listen()
		}
		http.Handle("/", http.FileServer(http.Dir(config.Webserver.Webroot)))
	}

	if config.Influxdb.Enable {
		statsDb = NewStatsDb()
	}

	if config.Respondd.Enable {
		collectInterval := time.Second * time.Duration(config.Respondd.CollectInterval)
		multiCollector = respond.NewMultiCollector(collectInterval, onReceive)
	}

	// TODO bad
	if config.Webserver.Enable {
		log.Fatal(http.ListenAndServe(net.JoinHostPort(config.Webserver.Address, config.Webserver.Port), nil))
	}

	// Wait for INT/TERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	log.Println("received", sig)

	// Close everything at the end
	if wsserverForNodes != nil {
		wsserverForNodes.Close()
	}
	if multiCollector != nil {
		multiCollector.Close()
	}
	if statsDb != nil {
		statsDb.Close()
	}
}

// called for every parsed announced-message
func onReceive(addr net.UDPAddr, msg interface{}) {
	switch msg := msg.(type) {

	case *data.NodeInfo:
		nodes.Get(msg.NodeId).Nodeinfo = msg

	case *data.Neighbours:
		nodes.Get(msg.NodeId).Neighbours = msg

	case *data.Statistics:
		nodes.Get(msg.NodeId).Statistics = msg

		// store data?
		if statsDb != nil {
			statsDb.Add(msg)
		}

	default:
		log.Println("unknown message:", msg)
	}
}

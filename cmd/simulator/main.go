package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"lab01/pkg/core"
	"lab01/pkg/util"
)

type Config struct {
	NumNodes         int   `json:"NumNodes"`
	BlocksToFinalize int   `json:"BlocksToFinalize"`
	Seed             int64 `json:"Seed"`
	MaxTicks         int   `json:"MaxTicks"`
	LatencyMin       int   `json:"LatencyMin"`
	LatencyMax       int   `json:"LatencyMax"`
}

func main() {
	cfgFile := flag.String("config", "config/scenario1.json", "config json")
	outFile := flag.String("out", "logs/run1.log", "log output file")
	flag.Parse()

	b, err := os.ReadFile(*cfgFile)
	if err != nil {
		log.Fatal(err)
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		log.Fatal(err)
	}

	_ = os.MkdirAll("logs", 0755)
	f, err := os.Create(*outFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	w := io.MultiWriter(os.Stdout, f)
	logger := util.NewLogger(w)

	rand.Seed(cfg.Seed)

	net := core.NewNetwork(cfg.Seed, cfg.LatencyMin, cfg.LatencyMax)

	// create nodes
	nodes := make([]*core.Node, 0, cfg.NumNodes)
	for i := 0; i < cfg.NumNodes; i++ {
		id := core.NodeID(fmt.Sprintf("node%02d", i))
		n := core.NewNode(id, cfg.Seed+int64(i), cfg.NumNodes, logger)
		core.RegisterNode(net, n)
		nodes = append(nodes, n)
	}

	// connect nodes into network
	net.SetNodes(nodes)

	// start nodes
	for _, n := range nodes {
		n.Start()
	}

	// run ticks until enough blocks finalized or max ticks
	ticks := 0
	for ticks < cfg.MaxTicks {
		net.Tick()
		for _, n := range nodes {
			n.OnTick()
		}
		ticks++
		// check if majority finalized BlocksToFinalize
		count := 0
		for _, n := range nodes {
			if n.FinalizedHeight() >= uint64(cfg.BlocksToFinalize) {
				count++
			}
		}
		if count >= (cfg.NumNodes/2 + 1) {
			logger.Printf("SIM|TICK=%d|DONE|finalized_by_majority=%d", ticks, count)
			break
		}
		// avoid busy loop
		time.Sleep(5 * time.Millisecond)
	}

	logger.Printf("SIM|TICKS=%d|END", ticks)
	// dump final state of node0
	s := nodes[0].SnapshotState()
	sb, _ := json.MarshalIndent(s, "", "  ")
	logger.Printf("STATE|node=node00|%s", string(sb))
}

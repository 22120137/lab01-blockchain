package core

import (
	"fmt"
	"io"
	"testing"
	"time"

	"lab01/pkg/util"
)

func TestConsensus_SimpleFinalization(t *testing.T) {
	// deterministic small network
	numNodes := 4
	seed := int64(12345)
	net := NewNetwork(seed, 1, 1)

	// discard logs to keep test output clean
	logger := util.NewLogger(io.Discard)

	// create nodes
	nodes := make([]*Node, 0, numNodes)
	for i := 0; i < numNodes; i++ {
		id := NodeID(fmt.Sprintf("node%02d", i))
		n := NewNode(id, seed+int64(i), numNodes, logger)
		RegisterNode(net, n)
		nodes = append(nodes, n)
	}

	net.SetNodes(nodes)
	for _, n := range nodes {
		n.Start()
	}

	// run ticks for a short time and check if majority finalized height >= 1
	maxTicks := 100
	for tick := 0; tick < maxTicks; tick++ {
		net.Tick()
		for _, n := range nodes {
			n.OnTick()
		}
		// small sleep to give goroutines a moment (keeps test stable)
		time.Sleep(2 * time.Millisecond)

		count := 0
		for _, n := range nodes {
			if n.FinalizedHeight() >= 1 {
				count++
			}
		}
		if count >= (numNodes/2 + 1) {
			// success
			return
		}
	}
	t.Fatalf("expected majority to finalize height 1 within %d ticks", maxTicks)
}

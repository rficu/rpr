package main

import (
	"github.com/rficu/rpr/pkg/connectivity"
	"github.com/rficu/rpr/pkg/rpr"
)

func main() {

	// spawn two nodes that have more than enough upload bandwidth
	// and a node that has only 1 mbps of upload bandwidth available
	node1 := connectivity.CreateNode(22000, 22002, 5, 100, "COMPAT")
	node2 := connectivity.CreateNode(23000, 23002, 10, 100, "COMPAT")
	node3 := connectivity.CreateNode(24000, 24002, 1, 20, "COMPAT")

	// TODO
	connectivity.Call(node1, node2.Tcp)

	// TODO
	rpr.RprFinalize(node1)
	rpr.RprFinalize(node2)

	// TODO
	connectivity.Call(node3, node2.Tcp)
	connectivity.Call(node3, node1.Tcp)

	// TODO
	rpr.RprFinalize(node3)

	// finally start the rtp loops, i.e., start exchanging rtp packets
	connectivity.StartRtpLoop(node1)
	connectivity.StartRtpLoop(node2)
	connectivity.StartRtpLoop(node3)

	for {
	}
}

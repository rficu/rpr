package main

import (
	"github.com/rficu/rpr/pkg/connectivity"
)

func main() {

	// spawn two nodes that have more than enough upload bandwidth
	// and a node that has only 1 mbps of upload bandwidth available
	node1 := connectivity.CreateNode(22000, 22002, 5, 100, "COMPAT")
	node2 := connectivity.CreateNode(23000, 23002, 10, 100, "COMPAT")
	node3 := connectivity.CreateNode(24000, 24002, 5, 20, "COMPAT")

	// this call initiation exchanges rtp ports without any rpr package relay agreements
	connectivity.Call(node1, node2.Tcp)

	// these call initiations try to establish a RPR packet relay agreement
	// as the node3 does not have enough upload bandwidth
	connectivity.Call(node3, node2.Tcp)
	connectivity.Call(node3, node1.Tcp)

	// finally start the rtp loops, i.e., start exchanging rtp packets
	connectivity.StartRtpLoop(node1)
	connectivity.StartRtpLoop(node2)
	connectivity.StartRtpLoop(node3)

	for {
	}
}

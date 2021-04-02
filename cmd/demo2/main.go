package main

import (
	"github.com/rficu/rpr/pkg/connectivity"
	"time"
)

func main() {

	// spawn two nodes that have more than enough upload bandwidth
	// and a node that has only 1 mbps of upload bandwidth available
	node1 := connectivity.CreateNode(22000, 22002, 5, 100, "COMPAT")
	node2 := connectivity.CreateNode(23000, 23002, 10, 100, "COMPAT")
	node3 := connectivity.CreateNode(24000, 24002, 3, 20, "COMPAT")

	// initialize context between the participants by calling each other
	connectivity.Call(node1, node2.Tcp)
	connectivity.Call(node3, node2.Tcp)
	connectivity.Call(node3, node1.Tcp)

	// finally start the rtp loops, i.e., start exchanging rtp packets
	connectivity.StartRtpLoop(node1)
	connectivity.StartRtpLoop(node2)
	connectivity.StartRtpLoop(node3)

	time.Sleep(2 * time.Second)

	node4 := connectivity.CreateNode(25000, 24002, 10, 20, "COMPAT")

	connectivity.Call(node4, node1.Tcp)
	connectivity.Call(node4, node2.Tcp)
	connectivity.Call(node4, node3.Tcp)

	connectivity.StartRtpLoop(node1)
	connectivity.StartRtpLoop(node2)
	connectivity.StartRtpLoop(node3)
	connectivity.StartRtpLoop(node4)

	for {
	}
}

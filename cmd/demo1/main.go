package main

import (
	"github.com/rficu/rpr/pkg/connectivity"
)

func main() {

	// spawn two nodes that have more than enough upload bandwidth
	// and a node that has only 1 mbps of upload bandwidth available
	node1 := connectivity.CreateNode(22000, 22002, 5, 100, "COMPAT")
	node2 := connectivity.CreateNode(23000, 23002, 10, 100, "COMPAT")
	node3 := connectivity.CreateNode(24000, 24002, 2, 20, "COMPAT")

	// initialize context between the participants by calling each other
	connectivity.Call(node1, node2.Tcp)
	connectivity.Call(node3, node1.Tcp)
	connectivity.Call(node3, node2.Tcp)

	for {
	}
}

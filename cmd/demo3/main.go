package main

import (
	"github.com/rficu/rpr/pkg"
	"math/rand"
	"time"
)

func main() {

	rand.Seed(time.Now().UnixNano())

	// spawn bootstrap and wait for it to initialize itself
	go rpr.InitBootstrap("127.0.0.1:2222")
	time.Sleep(500 * time.Millisecond)

	go rpr.InitNode("127.0.0.1:2222", 8100, 5, "COMPAT")
	time.Sleep(2000 * time.Millisecond)

	go rpr.InitNode("127.0.0.1:2222", 8200, 10, "COMPAT")
	time.Sleep(5 * 1000 * time.Millisecond)

	go rpr.InitNode("127.0.0.1:2222", 8300, 5, "COMPAT")
	time.Sleep(5 * 1000 * time.Millisecond)

	// this node requires packet relay services from one of the nodes
	go rpr.InitNode("127.0.0.1:2222", 8400, 1, "COMPAT")
	time.Sleep(5 * 1000 * time.Millisecond)

	// this node represents a video conferencing solution
	// that does not support RPR.
	//
	// It does, however, support RFC 3550 so it is able
	// receive the relayed packets
	go rpr.InitNode("127.0.0.1:2222", 8300, 5, "INCOMPAT")

	for {
		time.Sleep(5 * 1000 * time.Millisecond)
	}
}

package main

// This demo showcases the adaptivity of RPR. First three are three nodes
// and none of them require packet relaying but the bandwidth usage of one of the
// nodes is at its maximum and then a new node joins the session.
//
// This forces the node that had its bandwidth usage limited contact some node
// of the session and request packet relay services from that.
//
// Finally, the first node of the session leaves and thus packet relaying is no longer necessary
// which the client node notices and terminates the ongoing RRP agreement with the relay node
// and starts sending packets normally

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

	// this node is capable of acting as a relay node
	go rpr.InitNode("127.0.0.1:2222", 8100, 5)
	time.Sleep(1000 * time.Millisecond)

	// this node is capable of acting as a relay node
	go rpr.InitNode("127.0.0.1:2222", 8200, 10)
	time.Sleep(3 * 1000 * time.Millisecond)

	// this node can only acts as a client node because all of its
	// bandwidth is used by sending video to the two nodes above
	go rpr.InitNode("127.0.0.1:2222", 8300, 2)
	time.Sleep(3 * 1000 * time.Millisecond)

	// new node joins, it does not require packet relaying
	// but the node above cannot no longer send packets to
	// everybody which is why it start looking for
	// a relay node for its packets
	go rpr.InitNode("127.0.0.1:2222", 8400, 5)

	// let the session continue for 10 seconds
	time.Sleep(10 * 1000 * time.Millisecond)

	// TODO terminate first node

	for {
		time.Sleep(5 * 1000 * time.Millisecond)
	}
}

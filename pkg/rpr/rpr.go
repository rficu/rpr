package rpr

import (
	"encoding/gob"
	"fmt"
	"github.com/rficu/rpr/pkg/rtp"
	"sort"
	"time"
)

const (
	RELAY_DISCOVER = 0 // TODO
	RELAY_OFFER    = 1 // offer relay service if there's capacity available
	RELAY_REQUEST  = 2 // request packet relay
	RELAY_REJECT   = 3 // reject relay reserve/request
	RELAY_ACCEPT   = 4 // accept relay offer
)

type RprInit struct {
	Identifier uint32
	Capacity   int
	RelayType  int
}

type RprResponse struct {
	Identifier uint32
	Capacity   int
	RelayType  int
}

func rprMessageLoop(node *Node, Identifier uint32) {

	var msg RprMessage

	for {
		node.Rpr.Nodes[Identifier].Dec.Decode(&msg)

		node.Rpr.MsgReceived <- msg
		_ = <-node.Rpr.Nodes[Identifier].MsgHandled
	}
}

func buildRelayList(node *Node) {

	// TODO use other metrics on top of capacity

	for i, _ := range node.Rpr.Nodes {
		node.Rpr.Candidates = append(node.Rpr.Candidates, node.Rpr.Nodes[i])
	}

	sort.Slice(node.Rpr.Candidates, func(i, j int) bool {
		return node.Rpr.Candidates[i].Capacity > node.Rpr.Candidates[j].Capacity
	})
}

func contactRelayNode(node *Node) {
	var candidate RprNode

	// TODO make sure there's enough elements

	candidate, node.Rpr.Candidates = node.Rpr.Candidates[0], node.Rpr.Candidates[1:]

	candidate.Enc.Encode(RprMessage{
		node.Identifier,
		RELAY_REQUEST,
	})
}

func RprMainLoop(node *Node) {

	for {
		select {
		case <-node.Rpr.NodeJoined:

			if node.Rpr.Capacity <= len(node.Sessions) {
				if len(node.Rpr.Nodes) == 0 {
					fmt.Println("[rpr] warning: our capacity is full but there are no relay nodes available!")
					return
				}

				if node.Rpr.Role == NODE_CLIENT {
					fmt.Printf("[rpr] our capacity is maxed but we're already utilizing packet relaying!\n")
					return
				}

				buildRelayList(node)
				contactRelayNode(node)
			}

		case msg := <-node.Rpr.MsgReceived:

			// TODO explain
			if msg.RelayType == RELAY_REQUEST {
				if node.Rpr.Capacity <= len(node.Sessions) || node.Rpr.Role == NODE_RELAY {
					node.Rpr.Nodes[msg.Identifier].Enc.Encode(RprMessage{
						node.Identifier,
						RELAY_REJECT,
					})
					node.Rpr.Nodes[msg.Identifier].MsgHandled <- true
				}

				node.Rpr.Nodes[msg.Identifier].Enc.Encode(RprMessage{
					node.Identifier,
					RELAY_OFFER,
				})
				node.Rpr.Nodes[msg.Identifier].MsgHandled <- true

				// TODO explain
			} else if msg.RelayType == RELAY_OFFER {
				fmt.Printf("[rpr] %x: start using %x as relay node\n",
					uint32(node.Identifier), uint32(msg.Identifier))

				node.Rpr.Nodes[msg.Identifier].Enc.Encode(RprMessage{
					node.Identifier,
					RELAY_ACCEPT,
				})

				relay, _ := node.Rpr.Nodes[msg.Identifier]

				node.Rpr.Role = NODE_CLIENT
				node.Rpr.Node = relay
				node.Rpr.Nodes[msg.Identifier].MsgHandled <- true

				// TODO explain
			} else if msg.RelayType == RELAY_ACCEPT {
				fmt.Printf("[rpr] %x: start relaying packets for %x\n",
					uint32(node.Identifier), uint32(msg.Identifier))

				client, _ := node.Rpr.Nodes[msg.Identifier]

				node.Rpr.Node = client
				node.Rpr.Role = NODE_RELAY
				node.Rpr.Nodes[msg.Identifier].MsgHandled <- true

			} else if msg.RelayType == RELAY_REJECT {
				contactRelayNode(node)
				node.Rpr.Nodes[msg.Identifier].MsgHandled <- true
			} else {
				fmt.Printf("unknown relay message received: %d\n", msg.RelayType)
			}
		}
	}
}

func HandshakeResponder(local *Node, remote *ConnectivityInfo, enc *gob.Encoder, dec *gob.Decoder) {

	if remote.Compat != "COMPAT" {
		return
	}

	var msg RprInit

	dec.Decode(&msg)
	enc.Encode(RprInit{
		local.Identifier,
		local.Upload,
		RELAY_DISCOVER,
	})

	local.Rpr.Nodes[msg.Identifier] = RprNode{
		enc, dec, remote.Identifier, msg.Capacity, make(chan bool),
	}
	local.Rpr.Capacity = local.Upload

	// spawn a thread for this connection to listen for incoming packet relay requests
	go rprMessageLoop(local, msg.Identifier)
}

func HandshakeInitiator(local *Node, remote *ConnectivityInfo, enc *gob.Encoder, dec *gob.Decoder) {

	if remote.Compat != "COMPAT" {
		return
	}

	var resp RprResponse

	enc.Encode(RprInit{
		local.Identifier,
		local.Upload,
		RELAY_DISCOVER,
	})
	dec.Decode(&resp)

	local.Rpr.Nodes[resp.Identifier] = RprNode{
		enc, dec, remote.Identifier, resp.Capacity, make(chan bool),
	}
	local.Rpr.Capacity = local.Upload

	go rprMessageLoop(local, resp.Identifier)
}

func SendData(node *Node, session *rtp.Rtp, RemoteIdentifier uint32) {

	stamp := uint32(0)
	var payload [10]byte

	for {
		if node.Rpr.Role == NODE_CLIENT {
			if node.Rpr.Node.Identifier == RemoteIdentifier {
				session.SendPacket(node.Identifier, stamp, []uint32{}, payload)
			}
		} else {
			session.SendPacket(node.Identifier, stamp, []uint32{}, payload)
		}

		stamp += 160
		time.Sleep(time.Second)
	}
}

func RecvData(node *Node, session *rtp.Rtp) {
	for {
		select {
		case packet := <-session.PacketReceived:
			if node.Rpr.Role == NODE_RELAY && node.Rpr.Node.Identifier == packet.Ssrc {
				for _, remoteNode := range node.Sessions {
					if remoteNode.Remote.Identifier == packet.Ssrc {
						continue
					}

					remoteNode.Rtp.Session.SendPacket(
						node.Rpr.Node.Identifier,
						packet.Timestamp,
						[]uint32{node.Identifier},
						packet.Payload,
					)
				}
			}

			if len(packet.Csrc) > 0 {
				fmt.Printf("[rtp] %x: got relayed package from %x (%x)\n",
					uint32(node.Identifier), packet.Ssrc, packet.Csrc[0])
			} else {
				fmt.Printf("[rtp] %x: got package from %x\n", uint32(node.Identifier), packet.Ssrc)
			}
		}
	}
}

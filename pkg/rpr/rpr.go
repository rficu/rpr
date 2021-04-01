package rpr

import (
	"encoding/gob"
	"fmt"
	"github.com/wernerd/GoRTP/src/net/rtp"
	"time"
)

const (
	RELAY_DISCOVER = 0 // todo
	RELAY_OFFER    = 1 // offer relay service if there's capacity available
	RELAY_RESERVE  = 2 // reserve packet relay services from a relay node
	RELAY_REQUEST  = 3 // request packet relay
	RELAY_REJECT   = 4 // reject relay reserve/request
	RELAY_ACCEPT   = 5 // accept relay offer
)

type RprMessage struct {
	Identifier uint32
	RelayType  int
}

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

func rprRequestRelay(node *Node) bool {

	var msg RprMessage

	for _, relayNode := range node.Rpr.Nodes {
		relayNode.Enc.Encode(RprMessage{
			node.Identifier,
			RELAY_REQUEST,
		})
		relayNode.Dec.Decode(&msg)

		if msg.RelayType == RELAY_OFFER {
			fmt.Printf("[rpr] %x: start using %x as relay node\n",
				uint32(node.Identifier), uint32(msg.Identifier))

			relayNode.Enc.Encode(RprMessage{
				node.Identifier,
				RELAY_ACCEPT,
			})

			node.Rpr.Role = NODE_CLIENT
			node.Rpr.Node = relayNode
			delete(node.Rpr.Nodes, msg.Identifier)
			return true
		}
	}

	return false
}

func rprMessageLoop(local *Node, remote *ConnectivityInfo, enc *gob.Encoder, dec *gob.Decoder) {

	var msg RprMessage

	for {
		dec.Decode(&msg)

		if msg.RelayType == RELAY_REQUEST {

			// we don't have enough capacity or are already acting as a relay for someone else
			// TODO calculate our capacity correctly
			if local.Rpr.Capacity <= len(local.Sessions) || local.Rpr.Role == NODE_RELAY {
				enc.Encode(RprMessage{
					local.Identifier,
					RELAY_REJECT,
				})
			}

			enc.Encode(RprMessage{
				local.Identifier,
				RELAY_OFFER,
			})
			dec.Decode(&msg)

			if msg.RelayType == RELAY_ACCEPT {
				fmt.Printf("[rpr] %x: start relaying packets for %x\n",
					uint32(local.Identifier), uint32(msg.Identifier))

				client, _ := local.Rpr.Nodes[msg.Identifier]
				delete(local.Rpr.Nodes, msg.Identifier)

				local.Rpr.Node = client
				local.Rpr.Role = NODE_RELAY
			}
		}
	}
}

func RprFinalize(local *Node) {

	if local.Rpr.Capacity <= len(local.Sessions) {
		if len(local.Rpr.Nodes) == 0 {
			fmt.Println("[rpr] warning: our capacity is full but there are no relay nodes available!")
			return
		}

		if rprRequestRelay(local) == false {
			fmt.Printf("[rpr] warning: failed to find suitable relay node for us!\n")
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
		enc, dec, remote.Identifier, msg.Capacity,
	}
	local.Rpr.Capacity = local.Upload

	// spawn a thread for this connection to listen for incoming packet relay requests
	go rprMessageLoop(local, remote, enc, dec)
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
		enc, dec, remote.Identifier, resp.Capacity,
	}
	local.Rpr.Capacity = local.Upload
}

func sendRtpPacket(session *rtp.Session, ts uint32, payload []byte, csrc []uint32) {
	rp := session.NewDataPacket(ts)
	rp.SetPayload(payload[0:10])
	rp.SetCsrcList(csrc)
	session.WriteData(rp)
	rp.FreePacket()
}

func SendData(node *Node, sess *rtp.Session, RemoteIdentifier uint32) {

	stamp := uint32(0)
	localPay := make([]byte, 160)

	for {
		if node.Rpr.Role == NODE_CLIENT {
			if node.Rpr.Node.Identifier == RemoteIdentifier {
				sendRtpPacket(sess, stamp, localPay, []uint32{})
			}
		} else {
			sendRtpPacket(sess, stamp, localPay, []uint32{})
		}

		stamp += 160
		time.Sleep(time.Second)
	}
}

func RecvData(node *Node, sess *rtp.Session) {

	dataReceiver := sess.CreateDataReceiveChan()
	var cnt int

	for {
		select {
		case rp := <-dataReceiver:
			if node.Rpr.Role == NODE_RELAY {
				if node.Rpr.Node.Identifier == rp.Ssrc() {
					for _, remoteNode := range node.Sessions {
						if remoteNode.Remote.Identifier != rp.Ssrc() {
							sendRtpPacket(
								remoteNode.Rtp.Session,
								rp.Timestamp(),
								rp.Payload(),
								[]uint32{rp.Ssrc()},
							)
						}
					}
				}
			}

			if rp.CsrcCount() > 0 {
				fmt.Printf("[rtp] %x: got relayed package from %x\n", uint32(node.Identifier), rp.Ssrc())
			} else {
				fmt.Printf("[rtp] %x: got package from %x\n", uint32(node.Identifier), rp.Ssrc())
			}
			cnt++
			rp.FreePacket()
		}
	}
}

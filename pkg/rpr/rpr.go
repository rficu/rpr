package rpr

import (
	"encoding/gob"
	"fmt"
	"github.com/wernerd/GoRTP/src/net/rtp"
	"time"
)

const (
	RELAY_OFFER   = 0 // offer relay service if there's capacity available
	RELAY_RESERVE = 1 // reserve packet relay services from a relay node
	RELAY_REQUEST = 2 // request packet relay
	RELAY_REJECT  = 3 // reject relay reserve/request
	RELAY_ACCEPT  = 4 // accept relay offer
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

type RprMessage struct {
	Identifier uint32
	RelayType  int
}

func rprMessageLoop(local *Node, remote *ConnectivityInfo, enc *gob.Encoder, dec *gob.Decoder) {
	var msg RprMessage

	for {
		dec.Decode(&msg)

		if msg.RelayType == RELAY_REQUEST {
			if _, ok := local.Rpr.ReservedNodes[msg.Identifier]; ok {
				enc.Encode(RprMessage{
					local.Identifier,
					RELAY_OFFER,
				})
				dec.Decode(&msg)

				if msg.RelayType == RELAY_ACCEPT {
					fmt.Printf("[rpr] %x: start relaying packets for %x\n",
						uint32(local.Identifier), uint32(remote.Identifier))

					local.Rpr.Role = NODE_RELAY
					client, _ := local.Rpr.ReservedNodes[msg.Identifier]
					delete(local.Rpr.ReservedNodes, msg.Identifier)
					local.Rpr.ClientNodes[msg.Identifier] = client
				}
			}
		}
	}
}

func RprFinalize(local *Node) {

	var msg RprMessage
	_ = msg

	// TODO implement proper relay node selection, for now just select the first available
	if local.Rpr.Capacity <= 0 {
		if len(local.Rpr.RelayNodes) == 0 {
			fmt.Println("[rpr] warning: our capacity is full but there are no relay nodes available!")
			return
		}

		for _, relayNode := range local.Rpr.RelayNodes {
			relayNode.Enc.Encode(RprMessage{
				local.Identifier,
				RELAY_REQUEST,
			})
			relayNode.Dec.Decode(&msg)

			if msg.RelayType == RELAY_OFFER {
				fmt.Printf("[rpr] %x: start using %x as relay node\n",
					uint32(local.Identifier), uint32(msg.Identifier))

				relayNode.Enc.Encode(RprMessage{
					local.Identifier,
					RELAY_ACCEPT,
				})

				local.Rpr.Role = NODE_CLIENT
				local.Rpr.RelayNode = relayNode
				local.Rpr.Capacity = len(local.Sessions) - 1
				delete(local.Rpr.RelayNodes, msg.Identifier)
				break
			}
		}
	}
}

// TODO
func HandshakeResponder(local *Node, remote *ConnectivityInfo, enc *gob.Encoder, dec *gob.Decoder) {

	if remote.Compat != "COMPAT" {
		return
	}

	// read init message from initiator and based on RelayType, craft RprResponse
	var init RprInit
	var resp RprResponse

	dec.Decode(&init)

	if init.RelayType == RELAY_RESERVE {
		if local.Rpr.Capacity > 2 {
			local.Rpr.ReservedNodes[remote.Identifier] = RprNode{
				enc,
				dec,
				remote.Identifier,
			}
		}
	} else if init.RelayType == RELAY_OFFER {
		local.Rpr.RelayNodes[remote.Identifier] = RprNode{
			enc,
			dec,
			remote.Identifier,
		}
	}

	resp.Capacity = local.Rpr.Capacity
	resp.Identifier = local.Identifier

	enc.Encode(&resp)

	// spawn a thread for this connection to listen for incoming packet relay requests
	go rprMessageLoop(local, remote, enc, dec)
}

// TODO
func HandshakeInitiator(local *Node, remote *ConnectivityInfo, enc *gob.Encoder, dec *gob.Decoder) {

	if remote.Compat != "COMPAT" {
		return
	}

	var init RprInit
	var resp RprResponse

	// if our capacity is running low, ask if the node
	// could relay packets for us
	//
	// at this point we're only reserving a possible slot
	// as we don't know how many other relay nodes there
	// are available and with what characteristics (available
	// capacity, topology, latency etc.)
	//
	// if we have capacity, we offer it for the remote node
	init.Capacity = local.Rpr.Capacity
	init.Identifier = local.Identifier

	if local.Rpr.Capacity <= 2 {
		init.RelayType = RELAY_RESERVE
	} else {
		init.RelayType = RELAY_OFFER
	}

	enc.Encode(&init)
	dec.Decode(&resp)

	if resp.RelayType == RELAY_OFFER {
		local.Rpr.RelayNodes[remote.Identifier] = RprNode{
			enc,
			dec,
			remote.Identifier,
		}
	} else if resp.RelayType == RELAY_RESERVE {
		if init.RelayType == RELAY_OFFER {
			local.Rpr.ReservedNodes[remote.Identifier] = RprNode{
				enc,
				dec,
				remote.Identifier,
			}
		}
	}
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
			if node.Rpr.RelayNode.Identifier == RemoteIdentifier {
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
				if _, ok := node.Rpr.ClientNodes[rp.Ssrc()]; ok {
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

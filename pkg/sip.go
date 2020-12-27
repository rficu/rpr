package rpr

import (
	"encoding/gob"
	"fmt"
	"net"
)

const (
	SIP_INVITE = 0
	SIP_ACK    = 1
)

type NodeCapabilities struct {
	RPRCompat bool
	rtpPort   int
	UniqID    uint32
}

type SipMessage struct {
	UserAgent   string
	MessageType int
	RtpPort     int
	UniqID      uint32
}

// initialize session with remote node by exchanging user agents and rtp ports
//
// this function returns node capabilities which contain the remote rtp and port
// whether remote supports RPR
//
// this function is called by a new node that wishes to join the call
//
// this function blocks until ACK is received
func sipInitSessionInvite(conn net.Conn, id uint32, rtpPort int, userAgent string) NodeCapabilities {

	fmt.Printf("[sip] sending invite message: %d '%s'\n", rtpPort, userAgent)

	var msg SipMessage

	dec := gob.NewDecoder(conn)
	enc := gob.NewEncoder(conn)

	enc.Encode(&SipMessage{userAgent, SIP_INVITE, rtpPort, id})
	dec.Decode(&msg)

	return NodeCapabilities{userAgent == msg.UserAgent, msg.RtpPort, msg.UniqID}
}

// initialize session with remote node by exchanging user agents and rtp ports
//
// this function returns node capabilities which contain the remote rtp and port
// whether remote supports RPR
//
// this function is called by a node that is already part of a call and
// is negotiating session parameters with a joining participant
//
// this function blocks until INVITE has been received and ACK has been sent
func sipInitSessionAck(conn net.Conn, id uint32, rtpPort int, userAgent string) NodeCapabilities {

	var msg SipMessage
	var capabilities NodeCapabilities

	dec := gob.NewDecoder(conn)
	enc := gob.NewEncoder(conn)

	dec.Decode(&msg)

	capabilities.RPRCompat = msg.UserAgent == userAgent
	capabilities.rtpPort = msg.RtpPort
	capabilities.UniqID = msg.UniqID

	fmt.Printf("[sip] sending ack message: %d '%s'\n", rtpPort, userAgent)

	msg.MessageType = SIP_ACK
	msg.RtpPort = rtpPort
	msg.UserAgent = userAgent
	msg.UniqID = id

	enc.Encode(&msg)

	return capabilities
}

package rtp

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
)

type RtpPacket struct {
	Timestamp uint32
	Ssrc      uint32
	Csrc      []uint32
	Payload   [10]byte
}

type Rtp struct {
	PacketReceived chan RtpPacket
	recvConn       *net.UDPConn
	sendConn       net.Conn
}

func udpRunner(rtpInstance *Rtp) {

	inputBytes := make([]byte, 4096)

	for {
		var packet RtpPacket
		length, _, _ := rtpInstance.recvConn.ReadFromUDP(inputBytes)
		buffer := bytes.NewBuffer(inputBytes[:length])
		decoder := gob.NewDecoder(buffer)
		decoder.Decode(&packet)
		rtpInstance.PacketReceived <- packet
	}
}

func (r Rtp) SendPacket(ssrc, ts uint32, csrc []uint32, payload [10]byte) {

	packet := RtpPacket{
		ts, ssrc, csrc, payload,
	}

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	encoder.Encode(packet)
	r.sendConn.Write(buffer.Bytes())
	buffer.Reset()
}

func CreateSession(addr string, localPort int, remotePort int) *Rtp {

	var rtpInstance Rtp

	fmt.Printf("[rtp] %s:%d <-> %s:%d\n", addr, localPort, addr, remotePort)

	localAddress, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", localPort))
	conn, _ := net.ListenUDP("udp", localAddress)

	rtpInstance.recvConn = conn

	destinationAddress, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", remotePort))
	conn, _ = net.DialUDP("udp", nil, destinationAddress)

	rtpInstance.sendConn = conn
	rtpInstance.PacketReceived = make(chan RtpPacket)

	go udpRunner(&rtpInstance)
	return &rtpInstance
}

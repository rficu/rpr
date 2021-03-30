package connectivity

import (
	"fmt"
	"github.com/rficu/rpr/pkg/rpr"
	"github.com/wernerd/GoRTP/src/net/rtp"
	"net"
	"time"
)

var localPay [160]byte

func sendData(sess *rtp.Session) {

	var cnt int
	stamp := uint32(0)
	for {
		rp := sess.NewDataPacket(stamp)
		rp.SetPayload(localPay[:])
		sess.WriteData(rp)
		rp.FreePacket()
		if (cnt % 50) == 0 {
			// fmt.Printf("Local sent %d packets\n", cnt)
		}
		cnt++
		stamp += 160
		time.Sleep(20e6)
	}
}

func recvData(id uint32, sess *rtp.Session) {

	dataReceiver := sess.CreateDataReceiveChan()
	var cnt int

	for {
		select {
		case rp := <-dataReceiver: // just get a packet - maybe we add some tests later
			if (cnt % 100) == 0 {
				fmt.Printf("%x got package from %x\n", id, rp.Ssrc())
			}
			cnt++
			rp.FreePacket()
			// case <-stopLocalRecv:
			// 	return
		}
	}
}

func StartRtpLoop(node *rpr.Node) {

	var addr, _ = net.ResolveIPAddr("ip", "127.0.0.1")
	var rsRemote *rtp.Session

	for i, remoteNode := range node.Sessions {
		remotePort := remoteNode.Remote.Rtp
		tpRemote, _ := rtp.NewTransportUDP(addr, remotePort, "")
		rsRemote = rtp.NewSession(tpRemote, tpRemote)
		rsRemote.AddRemote(&rtp.Address{addr.IP, node.Rtp + i*2, node.Rtp + 1 + i*2, ""})

		strRemoteIdx, _ := rsRemote.NewSsrcStreamOut(&rtp.Address{
			addr.IP,
			remotePort,
			remotePort,
			"",
		}, uint32(node.Identifier), 0)
		rsRemote.SsrcStreamOutForIndex(strRemoteIdx).SetPayloadType(0)

		go recvData(uint32(node.Identifier), rsRemote)

		// simple RTP: just listen on the RTP and RTCP receive transports. Do not start Session.
		rsRemote.ListenOnTransports()

		go sendData(rsRemote)
	}
}

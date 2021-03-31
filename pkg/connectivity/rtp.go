package connectivity

import (
	"github.com/rficu/rpr/pkg/rpr"
	"github.com/wernerd/GoRTP/src/net/rtp"
	"net"
)

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

		go rpr.RecvData(uint32(node.Identifier), rsRemote)

		// simple RTP: just listen on the RTP and RTCP receive transports. Do not start Session.
		rsRemote.ListenOnTransports()

		go rpr.SendData(rsRemote)
	}
}
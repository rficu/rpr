package rpr

import (
	"fmt"
	"github.com/wernerd/GoRTP/src/net/rtp"
	"net"
	"time"
)

type RTP_Context struct {
	stop          bool
	stopLocalRecv chan bool
	rtpSession    *rtp.Session
}

var stopLocalRecv chan bool
var localPay [160]byte

func sendData(stop *bool, session *Session) {

	stamp := uint32(0)

	for !(*stop) {
		if session.rprInfo.role == CLIENT_NODE {
			if uint32(session.them.uniqID) == uint32(session.rprInfo.uniqID) {
				rp := session.rtpCtx.rtpSession.NewDataPacket(stamp)
				rp.SetPayload(localPay[:])
				session.rtpCtx.rtpSession.WriteData(rp)
				rp.FreePacket()
			}
		} else {
			rp := session.rtpCtx.rtpSession.NewDataPacket(stamp)
			rp.SetPayload(localPay[:])
			session.rtpCtx.rtpSession.WriteData(rp)
			rp.FreePacket()
		}

		stamp += 160
		time.Sleep(5 * 1000 * time.Millisecond)
	}
}

func recvData(session *Session, ctx *Context) {
	dataReceiver := session.rtpCtx.rtpSession.CreateDataReceiveChan()

	for {
		select {
		case rp := <-dataReceiver:
			// we are providing packet relaying services to some node and have
			// received a packet from that node and we must now send this packet
			// to other nodes of the session
			if session.rprInfo.role == RELAY_NODE {
				for i, sess := range ctx.sessions {
					if &ctx.sessions[i] != session {
						rp2 := sess.rtpCtx.rtpSession.NewDataPacket(rp.Timestamp())
						rp2.SetPayload(rp.Payload())
						rp2.SetCsrcList([]uint32{rp.Ssrc()})
						sess.rtpCtx.rtpSession.WriteData(rp2)
						rp2.FreePacket()
					}
				}
			}
			if rp.CsrcCount() > 0 {
				fmt.Printf("[rtp] %x received relayed packet from %x\n", session.us.uniqID, rp.CsrcList()[0])
			} else {
				fmt.Printf("[rtp] %x received packet from %x\n", session.us.uniqID, rp.Ssrc())
			}
			rp.FreePacket()
		case <-stopLocalRecv:
			return
		}
	}
}

func makeRtpSession(session *Session, ctx *Context) {
	addr, _ := net.ResolveIPAddr("ip", "127.0.0.1")
	transport, _ := rtp.NewTransportUDP(addr, session.us.rtpPort, "")
	session.rtpCtx.rtpSession = rtp.NewSession(transport, transport)

	addr_ := "127.0.0.1"
	remoteAddr := rtp.Address{addr.IP, session.them.rtpPort, session.them.rtpPort + 1, ""}
	localAddr := rtp.Address{addr.IP, session.us.rtpPort, session.us.rtpPort + 1, ""}

	session.rtpCtx.rtpSession.AddRemote(&remoteAddr)

	fmt.Printf("[rtp] create session: %s:%d <-> %s:%d\n", addr_, session.us.rtpPort,
		addr_, session.them.rtpPort)

	strLocalIdx, _ := session.rtpCtx.rtpSession.NewSsrcStreamOut(&localAddr, uint32(session.us.uniqID), 0)
	session.rtpCtx.rtpSession.SsrcStreamOutForIndex(strLocalIdx).SetPayloadType(0)

	session.rtpCtx.stopLocalRecv = make(chan bool, 1)
	session.rtpCtx.stop = false

	go recvData(session, ctx)

	session.rtpCtx.rtpSession.StartSession()

	go sendData(&session.rtpCtx.stop, session)
}

func rtpLoop(ctx *Context) {

	for i, _ := range ctx.sessions {
		makeRtpSession(&ctx.sessions[i], ctx)
		time.Sleep(5 * 1000 * time.Millisecond)
	}

	for {
		prev := len(ctx.sessions)

		time.Sleep(2000 * time.Millisecond)

		if len(ctx.sessions) > prev {
			fmt.Printf("initialize new rtp session for %x\n", ctx.sessions[len(ctx.sessions)-1].them.uniqID)
			makeRtpSession(&ctx.sessions[len(ctx.sessions)-1], ctx)
		}
	}
}

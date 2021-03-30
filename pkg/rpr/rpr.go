package rpr

import (
	"fmt"
	"github.com/wernerd/GoRTP/src/net/rtp"
	"time"
)

func Handshake(us *Node, theirInfo *ConnectivityInfo) {
}

func SendData(sess *rtp.Session) {

	var cnt int
	var localPay [160]byte
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

func RecvData(id uint32, sess *rtp.Session) {

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

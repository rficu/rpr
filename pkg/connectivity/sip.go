package connectivity

import (
	"encoding/gob"
	"fmt"
	"github.com/rficu/rpr/pkg/rpr"
	"github.com/rficu/rpr/pkg/rtp"
	"math/rand"
	"net"
	"sync"
)

func Call(us *rpr.Node, tcp int) {

	var theirInfo rpr.ConnectivityInfo
	var c net.Conn
	var err error

	for {
		c, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", tcp))
		if err == nil {
			break
		}
	}

	enc := gob.NewEncoder(c)
	dec := gob.NewDecoder(c)

	// send our ConnectivityInfo to remote, read their response (i.e., exchange rtp ports
	// and bandwidth/compatibility info), perform rpr handshake and save the new node's
	// info to our node object
	us.Mtx.Lock()

	enc.Encode(&rpr.ConnectivityInfo{
		us.Identifier,
		us.Rtp + len(us.Sessions)*2,
		us.Upload,
		us.Download,
		us.Compat,
	})
	dec.Decode(&theirInfo)

	if us.Compat == "COMPAT" && theirInfo.Compat == "COMPAT" {
		rpr.HandshakeInitiator(us, &theirInfo, enc, dec)
	}

	sess := rpr.Session{
		theirInfo,
		rpr.RtpContext{
			rtp.CreateSession("127.0.0.1", us.Rtp+len(us.Sessions)*2, theirInfo.Rtp),
			false,
		},
	}

	us.Rpr.Role = rpr.NODE_NORMAL
	us.Sessions = append(us.Sessions, sess)

	if us.Compat == "COMPAT" && theirInfo.Compat == "COMPAT" {
		us.Rpr.NodeJoined <- true
	}

	go rpr.SendData(us, sess.Rtp.Session, theirInfo.Identifier)
	go rpr.RecvData(us, sess.Rtp.Session)

	us.Mtx.Unlock()
}

func sipListener(us *rpr.Node) {

	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", us.Tcp))
	if err != nil {
		fmt.Println(err)
		return
	}

	for us.Running {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}

		var theirInfo rpr.ConnectivityInfo
		enc := gob.NewEncoder(c)
		dec := gob.NewDecoder(c)

		us.Mtx.Lock()

		dec.Decode(&theirInfo)

		enc.Encode(&rpr.ConnectivityInfo{
			us.Identifier,
			us.Rtp + len(us.Sessions)*2,
			us.Upload,
			us.Download,
			us.Compat,
		})

		if us.Compat == "COMPAT" && theirInfo.Compat == "COMPAT" {
			rpr.HandshakeResponder(us, &theirInfo, enc, dec)
		}

		sess := rpr.Session{
			theirInfo,
			rpr.RtpContext{
				rtp.CreateSession("127.0.0.1", us.Rtp+len(us.Sessions)*2, theirInfo.Rtp),
				false,
			},
		}

		us.Sessions = append(us.Sessions, sess)
		if us.Compat == "COMPAT" && theirInfo.Compat == "COMPAT" {
			us.Rpr.NodeJoined <- true
		}

		go rpr.SendData(us, sess.Rtp.Session, theirInfo.Identifier)
		go rpr.RecvData(us, sess.Rtp.Session)

		us.Mtx.Unlock()
	}
}

func CreateNode(tcp int, rtp int, upload int, download int, compat string) *rpr.Node {
	ret := rpr.Node{
		tcp,
		rtp,
		upload,
		download,
		compat,
		rand.Uint32(),
		rpr.RprContext{
			upload,
			rpr.NODE_NORMAL,
			rpr.RprNode{},
			map[uint32]rpr.RprNode{},
			[]rpr.RprNode{},
			make(chan bool),
			make(chan rpr.RprMessage),
		},
		[]rpr.Session{},
		sync.Mutex{},
		true,
		make(chan bool),
	}

	go sipListener(&ret)

	if compat == "COMPAT" {
		go rpr.RprMainLoop(&ret)
	}

	return &ret
}

func EndCall(node *rpr.Node) {
	fmt.Printf("[sip] %x exiting...\n", node.Identifier)
	node.Exiting <- true
	node.Running = false
	// TODO stop sip loop
}

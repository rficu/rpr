package connectivity

import (
	"encoding/gob"
	"fmt"
	"github.com/rficu/rpr/pkg/rpr"
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
	rpr.Handshake(us, &theirInfo)

	sess := rpr.Session{
		theirInfo,
		rpr.RtpContext{},
	}

	us.Sessions = append(us.Sessions, sess)
	us.Mtx.Unlock()
}

func sipListener(us *rpr.Node) {

	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", us.Tcp))
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
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

		rpr.Handshake(us, &theirInfo)

		sess := rpr.Session{
			theirInfo,
			rpr.RtpContext{},
		}

		us.Sessions = append(us.Sessions, sess)
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
		rand.Int(),
		[]rpr.Session{},
		sync.Mutex{},
	}
	go sipListener(&ret)
	return &ret
}

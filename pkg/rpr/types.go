package rpr

import (
	"encoding/gob"
	"github.com/wernerd/GoRTP/src/net/rtp"
	"sync"
)

const (
	NODE_NORMAL = 0
	NODE_RELAY  = 1
	NODE_CLIENT = 2
)

type RprNode struct {
	Enc        *gob.Encoder
	Dec        *gob.Decoder
	Identifier uint32
	Capacity   int
}

type RprContext struct {
	Role     int                // are we client/relay/normal node
	Capacity int                // TODO
	Node     RprNode            // selected relay/client node
	Nodes    map[uint32]RprNode // all nodes of the session that support rpr
}

type ConnectivityInfo struct {
	Identifier uint32
	Rtp        int
	Upload     int
	Download   int
	Compat     string
}

type RtpContext struct {
	Session       *rtp.Session
	StopLocalRecv chan bool
	Stop          bool
}

type Session struct {
	Remote ConnectivityInfo
	Rtp    RtpContext
}

type Node struct {
	Tcp        int
	Rtp        int
	Upload     int
	Download   int
	Compat     string
	Identifier uint32
	Rpr        RprContext
	Sessions   []Session
	Mtx        sync.Mutex
}

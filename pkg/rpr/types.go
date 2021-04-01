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
}

// TODO convert these slices to maps
type RprContext struct {
	Capacity      int                // how much capacity the node has left
	Role          int                // are we client/relay/normal node
	RelayNode     RprNode            // selected relay node
	ClientNodes   map[uint32]RprNode // list of client nodes we're serving
	RelayNodes    map[uint32]RprNode // list of relay nodes we can use
	ReservedNodes map[uint32]RprNode // list of nodes that have reserved space
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

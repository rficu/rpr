package rpr

import (
	"github.com/wernerd/GoRTP/src/net/rtp"
	"sync"
)

type RPRContext struct {
}

type ConnectivityInfo struct {
	Identifier int
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
	Remote     ConnectivityInfo
	RtpContext RtpContext
}

type Node struct {
	Tcp        int
	Rtp        int
	Upload     int
	Download   int
	Compat     string
	Identifier int
	Sessions   []Session
	// Nodes       []ConnectivityInfo
	// RtpContexts []RtpContext
	Mtx sync.Mutex
}

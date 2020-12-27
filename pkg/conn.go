package rpr

import (
	"encoding/gob"
	"fmt"
	"math/rand"
	"net"
)

type Context struct {
	id             uint32       // id of the owner of this context
	port           int          // tcp port of the node for call initiation
	sessionChannel chan Session // channel for communication between acceptCalls and rtpLoop
	sessions       []Session    // list of sessions
	relayCtx       RPR_Context  // RPR context (global to context)
}

type Node struct {
	tcpPort      int    // remote's port for sip-related tcp traffic
	rtpPort      int    // remote's port for rtp traffic
	rprSupported bool   // does remote support RPR
	uniqID       uint32 // unique id for the node
}

type Session struct {
	us      Node     // our info
	them    Node     // remote info
	conn    net.Conn // ongoing tcp connection with remote
	rprInfo PacketRelayAgreement
	rtpCtx  RTP_Context // RTP-related context
}

type Participant struct {
	Address string
	Port    int
	UniqID  uint32
}

// @param connn: address (ip:port) of bootstrap node
// @param port: caller's tcp port
func getParticipants(conn string, port int) ([]Participant, Participant) {
	c, err := net.Dial("tcp", conn)
	if err != nil {
		fmt.Println(err)
		return []Participant{}, Participant{}
	}

	enc := gob.NewEncoder(c)
	dec := gob.NewDecoder(c)
	p := Participant{"127.0.0.1", port, rand.Uint32()}

	var participants []Participant

	dec.Decode(&participants)
	enc.Encode(&p)

	return participants, p
}

func finalizeInit(ctx Context) {

	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", ctx.port))
	if err != nil {
		fmt.Println(err)
		return
	}

	node := Node{ctx.port, 10000 + ctx.port + len(ctx.sessions)*2, true, ctx.id}

	// once the RPR context has been initialized with every node,
	// separate RTP runner can be created for each
	go rtpLoop(&ctx)

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}

		capab := sipInitSessionAck(c, ctx.id, node.rtpPort, "COMPAT")
		them := Node{0, capab.rtpPort, capab.RPRCompat, capab.UniqID}

		session := Session{node, them, c, PacketRelayAgreement{}, RTP_Context{}}

		if session.them.rprSupported {
			rprPerformHandshake(&ctx.relayCtx, &session)
		}

		fmt.Printf("initialize new generic session for %x\n", session.them.uniqID)
		ctx.sessions = append(ctx.sessions, session)
		node.rtpPort += 2 // rtp + rtcp
	}
}

// @param conn: address of the bootstrap node
func InitBootstrap(conn string) {
	l, err := net.Listen("tcp", conn)
	if err != nil {
		fmt.Println(err)
		return
	}

	var participants []Participant
	var participant Participant

	fmt.Printf("[bootstrap] listening incoming connections: %s...\n", conn)

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}

		enc := gob.NewEncoder(c)
		dec := gob.NewDecoder(c)

		enc.Encode(&participants)
		dec.Decode(&participant)

		fmt.Printf("[bootstrap] node %x from %s:%d joined\n",
			participant.UniqID, participant.Address, participant.Port)

		participants = append(participants, participant)
	}
}

func InitNode(conn string, port int, up int) {
	// connect to bootstrap node and fetch all participants of an ongoing call
	participants, p := getParticipants(conn, port)

	// when we have received the participants, call each node by sending them
	// a SIP INVITE message that contains our rtp for this node and
	// our user agent ("COMPAT"). Remote node responds to our INVITE
	// message with SIP ACK message that contains remote's rtp port
	// and their user agent.
	// Create new Session object for each node
	var ctx Context

	for i, participant := range participants {
		c_n, err := net.Dial("tcp", fmt.Sprintf("%s:%d", participant.Address, participant.Port))
		if err != nil {
			fmt.Printf("unable to connect to remote at %d: %s\n", participant.Port, err)
			return
		}

		rtpPort := 10000 + port + (i * 2)
		capab := sipInitSessionInvite(c_n, p.UniqID, rtpPort, "COMPAT")

		var remote Node = Node{participant.Port, capab.rtpPort, capab.RPRCompat, participant.UniqID}
		var local Node = Node{rtpPort, rtpPort, true, p.UniqID}
		ctx.sessions = append(ctx.sessions, Session{local, remote, c_n, PacketRelayAgreement{}, RTP_Context{}})
	}

	// When a session has been established with a remote node,
	// we perform RPR handshake with a remote to either
	//
	// a) query if remote is capable of relaying packets for us
	// b) query if remote needs us to relay packets for them
	//
	// RPR handshake is performed only if both we and remote support RPR
	ctx.relayCtx = initRPR(&ctx.sessions, up)
	ctx.id = p.UniqID
	ctx.port = port

	// finalize the session by transferring control to finalizeInit()
	// which spawns a separate thread for rtp loop and starts listening
	// to incoming calls
	finalizeInit(ctx)
}

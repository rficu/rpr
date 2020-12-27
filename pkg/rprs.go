package rpr

import (
	"encoding/gob"
	"fmt"
	"math"
)

const (
	RELAY_REJECT  = 0 // node cannot provide/does not require packet relaying
	RELAY_REQUEST = 1 // node requires packet relaying
	RELAY_OFFER   = 2 // node offers packet relay service
	RELAY_RELEASE = 3 // resign from the negotiated packet relay agreement
)

const (
	CONF_END     = 0 // end rpr session with this node
	CONF_ACCEPT  = 1 // accept packet relay offer
	CONF_CONFIRM = 2 // confirm the acceptance of packet relaying
)

const (
	NORMAL_NODE = 0
	CLIENT_NODE = 1
	RELAY_NODE  = 2
)

type RPR_Context struct {
	capacity int // how much bandwidth is in use
	upload   int // maximum upload bandwidth available
	nodes    int // number of active nodes in the session
	serving  int // how many nodes this node is serving
}

type RPR_Session struct {
	remote Node
}

type PacketRelayAgreement struct {
	role   int    // are we the client or relay
	uniqID uint32 // unique id of relay node
}

type InitMessage struct {
	UniqID      uint32 // our unique id
	Upload      int    // our upload bandwidth (mbps)
	RelayType   int    // TODO
	RelayNeeded bool   // do we need packet relaying
}

// when InitMessage is received, it is assumed that the joining node might require
// packet relaying but it is also possible that an old node also requires packet
// relaying now that the session is about to have one more node
type InitResponseMessage struct {
	UniqID         uint32 // our unique id
	Upload         int    // our upload bandwidth
	Capacity       int    // capacity of the node
	RelayNeeded    bool   // do we need packet relaying
	RelayAvailable bool   // can we provide packet relaying
	RelayType      int    // TODO
}

type RPRMessage struct {
	UniqID    uint32 // our unique id
	RelayType int    // relay type
}

type SessionConfigurationMessage struct {
	UniqID      uint32 // our unique id
	MessageType int    // indicates how the RPR handshake continues
}

func initRPR(sessions *[]Session, upload int) RPR_Context {

	if len((*sessions)) == 0 {
		return RPR_Context{1, upload, 0, 0}
	}

	fmt.Printf("[rpr] starting session initialization with %d nodes...\n", len((*sessions)))

	// the simplified assumption current version of RPR makes is that
	// node streams the same video feed to others node and that
	// required bandwidth per node is 1 Mbps
	var initMsg InitMessage
	var initRespMsg InitResponseMessage
	var initResponses []InitResponseMessage
	var sessionConfig SessionConfigurationMessage

	initMsg.UniqID = (*sessions)[0].us.uniqID
	initMsg.Upload = upload

	// We need packet relay services if our upload bandwidth is less than
	// # of nodes * 1 mbps. InitMessage does not yet initiate any packet
	// relay agreements but only notifies other participants that such
	// a service is needed by this node
	if upload < len((*sessions)) {
		initMsg.RelayType = RELAY_REQUEST
	} else {
		initMsg.RelayType = RELAY_REJECT
	}

	for _, session := range *sessions {
		if !session.them.rprSupported {
			continue
		}

		enc := gob.NewEncoder(session.conn)
		dec := gob.NewDecoder(session.conn)

		enc.Encode(&initMsg)
		dec.Decode(&initRespMsg)

		initResponses = append(initResponses, initRespMsg)
	}

	// when all responses have been received, we need to parse them
	// to find the most suitable node
	// the most suitable is considered to be the one that is capable
	// of providing packet relaying while also having the amount
	// of nodes being serviced
	selected := -1
	bandwidth := math.MinInt32

	for i, response := range initResponses {
		if response.RelayType == RELAY_OFFER {
			if response.Upload-response.Capacity > bandwidth {
				fmt.Printf("selected node has %d and %d: %d, %d\n",
					response.Upload, response.Capacity, response.Upload-response.Capacity, bandwidth)
				bandwidth = response.Upload - response.Capacity
				selected = i
			}
		}
	}

	// none of the nodes could provide packet relaying while we requested it
	// packet loss can be expected but the call is started nonetheless
	if selected == -1 && initMsg.RelayType == RELAY_REQUEST {
		fmt.Printf("[rpr] could not find suitable node for packet relaying!\n")

		for _, session := range *sessions {
			enc := gob.NewEncoder(session.conn)
			enc.Encode(&SessionConfigurationMessage{(*sessions)[0].us.uniqID, CONF_END})
			session.rprInfo.role = CLIENT_NODE
		}

		return RPR_Context{1, upload, len((*sessions)), 0}
	}

	// packet relaying was not requested from other nodes
	// so we can directly proceed to the media exchange
	if initMsg.RelayType == RELAY_REJECT {
		fmt.Println("[rpr] packet relaying not requested")

		for _, session := range *sessions {
			enc := gob.NewEncoder(session.conn)
			enc.Encode(&SessionConfigurationMessage{(*sessions)[0].us.uniqID, CONF_END})
			session.rprInfo.role = CLIENT_NODE
		}

		return RPR_Context{1, upload, len((*sessions)), 0}
	}

	// we have requested packet relaying from one of the nodes and our request
	// was accepted. Send confirmation message to that node, wait for final
	// confirmation and update our sessions object to only send packets to
	// the selected relay node and send CONF_END message to all but the selected node
	fmt.Printf("[rpr] selected relay node: %d\n", initResponses[selected].UniqID)

	for i, session := range *sessions {
		if i != selected {
			enc := gob.NewEncoder(session.conn)
			enc.Encode(&SessionConfigurationMessage{(*sessions)[0].us.uniqID, CONF_END})
			session.rprInfo.role = CLIENT_NODE
		}
	}

	fmt.Printf("[rpr] selected id: %x\n", (*sessions)[selected].them.uniqID)

	enc := gob.NewEncoder((*sessions)[selected].conn)
	dec := gob.NewDecoder((*sessions)[selected].conn)

	enc.Encode(&SessionConfigurationMessage{(*sessions)[0].us.uniqID, CONF_ACCEPT})
	dec.Decode(&sessionConfig)

	if sessionConfig.MessageType != CONF_CONFIRM {
		fmt.Printf("[rpr] could not confirm packet relay with remote: %d!\n", sessionConfig.MessageType)
		return RPR_Context{1, upload, len((*sessions)), 0}
	}

	fmt.Printf("[rpr] packet relay confirmation received from %x\n", (*sessions)[selected].them.uniqID)

	for i, _ := range *sessions {
		(*sessions)[i].rprInfo.role = CLIENT_NODE
		(*sessions)[i].rprInfo.uniqID = (*sessions)[selected].them.uniqID
	}

	return RPR_Context{1, upload, len((*sessions)), 0}
}

func rprPerformHandshake(ctx *RPR_Context, session *Session) {
	fmt.Printf("[rpr] performing rpr handshake between %d and %d...\n",
		session.us.uniqID, session.them.uniqID)

	// we're the receiving node in this case so we wait for
	// the new node to send us an InitMessage that contains
	// their upload bandwidth and their need for packet relaying
	var initMsg InitMessage
	var initRespMsg InitResponseMessage
	var confMsg SessionConfigurationMessage

	enc := gob.NewEncoder(session.conn)
	dec := gob.NewDecoder(session.conn)

	initRespMsg.UniqID = session.us.uniqID
	initRespMsg.Upload = ctx.upload
	initRespMsg.Capacity = ctx.capacity

	// in this version of RPR there are two modes we can provide:
	//	- relay offer
	//  - relay reject
	//
	// relay offer means that if we have enough bandwidth we can
	// provide packet relaying for the remote. If our bandwidth
	// is at its maximum, we preemptively reject any packet relay requests
	if (ctx.upload - ctx.capacity) <= 0 {
		initRespMsg.RelayType = RELAY_REJECT
	} else if int(math.Exp2(float64(ctx.serving+1))) < ctx.upload {
		fmt.Printf("[rpr] node is capable of providing packet relay\n")
		initRespMsg.RelayType = RELAY_OFFER
	} else {
		fmt.Printf("[rpr] capacity %d serving %d upload %d\n",
			ctx.capacity, ctx.serving, ctx.upload)
	}

	// receive init message from remote and respond to it with init response
	dec.Decode(&initMsg)
	enc.Encode(&initRespMsg)

	// when remote has received our init response, it sends us a session configuration
	// message which contains information about how to RPR session proceeds
	// i.e. does it accept our relay offer, reserve a slot or reject our offer
	dec.Decode(&confMsg)

	// if remote does not request packet relaying from us, we can exit early
	// CONF_END message need not to be acknowledged
	//
	// if remote accepts our offer (and we provided an offer), we confirm
	// the request and update our RPR context and session so the RTP
	// runner knows how to deal with packets coming from remote
	if confMsg.MessageType == CONF_END {
		fmt.Printf("[rpr] end rpr handshake: %x\n", session.us.uniqID)
		session.rprInfo.role = NORMAL_NODE
	} else if confMsg.MessageType == CONF_ACCEPT {
		if initRespMsg.RelayType != RELAY_OFFER {
			fmt.Println("[rpr] offer accepted but none was provided!")
			session.rprInfo.role = NORMAL_NODE
			ctx.nodes++
			return
		}

		enc.Encode(&SessionConfigurationMessage{session.us.uniqID, CONF_CONFIRM})

		session.rprInfo.role = RELAY_NODE
		session.rprInfo.uniqID = session.them.uniqID

		fmt.Printf("[rpr] packet relay confirmation sent: %x\n", session.us.uniqID)
	}

	ctx.nodes++
}

func rprRequestRelay(ctx Context) bool {

	if !rprNeedRelay(ctx.relayCtx) {
		return true
	}

	var selected int
	var reqMsg RPRMessage
	var respMsg RPRMessage
	var sessConf SessionConfigurationMessage

	reqMsg.UniqID = (ctx.sessions)[0].us.uniqID
	reqMsg.RelayType = RELAY_REQUEST

	// select the first offer received
	for i, session := range ctx.sessions {
		enc := gob.NewEncoder(session.conn)
		dec := gob.NewDecoder(session.conn)

		enc.Encode(&reqMsg)
		dec.Decode(&respMsg)

		if respMsg.RelayType == RELAY_OFFER {
			selected = i
			goto accept
		} else {
			enc := gob.NewEncoder(session.conn)
			enc.Encode(&SessionConfigurationMessage{(ctx.sessions)[0].us.uniqID, CONF_END})
		}
	}

	return false

accept:
	fmt.Printf("[rpr] selected id: %x\n", (ctx.sessions)[selected].them.uniqID)

	enc := gob.NewEncoder((ctx.sessions)[selected].conn)
	dec := gob.NewDecoder((ctx.sessions)[selected].conn)

	enc.Encode(&SessionConfigurationMessage{(ctx.sessions)[0].us.uniqID, CONF_ACCEPT})
	dec.Decode(&sessConf)

	if sessConf.MessageType != CONF_CONFIRM {
		fmt.Printf("[rpr] could not confirm packet relay with remote: %d!\n", sessConf.MessageType)
		// TODO retry with other candidate relay nodes
		return false
	}

	fmt.Printf("[rpr] packet relay confirmation received from %x\n", (ctx.sessions)[selected].them.uniqID)

	for i, _ := range ctx.sessions {
		(ctx.sessions)[i].rprInfo.role = CLIENT_NODE
		(ctx.sessions)[i].rprInfo.uniqID = (ctx.sessions)[selected].them.uniqID
	}

	return true
}

func rprReleaseAgreement(ctx *Context, session *Session) {

	var sessConf SessionConfigurationMessage

	enc := gob.NewEncoder(session.conn)
	dec := gob.NewDecoder(session.conn)

	enc.Encode(&RPRMessage{(ctx.sessions)[0].us.uniqID, RELAY_RELEASE})
	dec.Decode(&sessConf)

	if sessConf.MessageType != CONF_CONFIRM {
		fmt.Printf("received an invalid response from relay node: %d\n", sessConf.MessageType)
	}
	enc.Encode(&SessionConfigurationMessage{(ctx.sessions)[0].us.uniqID, CONF_END})

	for i, _ := range ctx.sessions {
		(ctx.sessions)[i].rprInfo.role = NORMAL_NODE
		(ctx.sessions)[i].rprInfo.uniqID = 0
	}
}

func rprPacketHandler(ctx *Context, session *Session) {

	enc := gob.NewEncoder(session.conn)
	dec := gob.NewDecoder(session.conn)

	var rprMsg RPRMessage
	var sessMsg SessionConfigurationMessage

	for {
		dec.Decode(&rprMsg)

		switch rprMsg.RelayType {
		case RELAY_REQUEST:
			if ctx.relayCtx.upload-ctx.relayCtx.nodes > 1 {

				rprMsg.UniqID = session.us.uniqID
				rprMsg.RelayType = RELAY_OFFER

				enc.Encode(&rprMsg)
				dec.Decode(&sessMsg)

				if sessMsg.MessageType == CONF_ACCEPT {
					enc.Encode(&SessionConfigurationMessage{(ctx.sessions)[0].us.uniqID, CONF_CONFIRM})
					session.rprInfo.role = RELAY_NODE
				} else if sessMsg.MessageType == CONF_END {
					fmt.Printf("client abruptly ended the handshake with us!\n")
				} else {
					fmt.Printf("received an invalid message: %d!\n", sessMsg.MessageType)
				}
			} else {
				rprMsg.UniqID = session.us.uniqID
				rprMsg.RelayType = RELAY_REJECT

				enc.Encode(&rprMsg)
				dec.Decode(&sessMsg)

				if sessMsg.MessageType != CONF_END {
					fmt.Printf("received an invalid message: %d!\n", sessMsg.MessageType)
				}
			}
		case RELAY_RELEASE:
			for _, sess := range ctx.sessions {
				if sess.them.uniqID == rprMsg.UniqID {
					enc.Encode(&SessionConfigurationMessage{(ctx.sessions)[0].us.uniqID, CONF_CONFIRM})
					dec.Decode(&sessMsg)

					if sessMsg.MessageType != CONF_END {
						fmt.Printf("received an invalid message: %d!\n", sessMsg.MessageType)
					}

					session.rprInfo.role = RELAY_NODE
				}
			}
		}
	}
}

func rprNeedRelay(ctx RPR_Context) bool {

	if ctx.upload-ctx.nodes <= 0 {
		return true
	}

	return false
}

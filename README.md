# Resource-aware Packet Routing for Large-scale P2P Video Conferences (RPR)

RPR is a scheme for allocating upload bandwidth from other participants of a video call for peers that have a very limited upload bandwidth to enable everyone to send video at acceptable quality.

If you're interested in the idea behind this project, please read [this](https://vizardy.net/blog/scaling_to_infinity.html) blogpost about it.

This repository provides an implementation of RPR and a few demo applications that showcase the functionality of the protocol. Because the main point is to showcase the protocol, the implementation is very verbose and cuts some corners. For example, the demo applications assume that a constant 1 Mbps of video is sent and fake values for upload bandwidth are used instead of finding the actual values.

`pkg/rpr` contains the actual protocol implementation and RTP send and receiver functions that are modified to support packet relaying. `pkg/connectivity` contains code related to call initiation and is not part of the actual protocol. `pkg/rtp` contains a very tiny RTP implementation that only implements the necessary stuff from RFC 3550 to make this work.

## Interoperability

So what is the value of this if it requires reinventing the wheel for video conferences and breaks compatibility with other software? [RFC 3350](https://tools.ietf.org/html/rfc3550) provides two wonderful concepts: synchronization sources (SSRC) and mixers/translators. SSRCs can be used to distinguish different participants of a session because SSRCs are unique. Translators, on the other hand, receive packets from one or more sources and send these packets to some destination without modifiying the packet. In other words, RFC 3550 supports the packet relaying concept of RPR.

The "only" thing that requires outside the specification signaling is the actual packet relaying agreement between participants. The support for RPR can be signaled e.g. using a SIP INVITE messages which tells the software name and version. Application can assume that if it is conversing with itself (e.g. two Skypes are discussing) that both applications support RPR and packet relaying agreement can be reached. If the SIP INVITE message indicates that the application is conversing with an incompatible implementation (e.g. Skype and Jitsi), RPR packet relay agreement does not take place and the call is initiated the normal way. Demo 3 provides an example of this functionality and it requires **no extra functionality** from an incompatible video conference application whilst providing packet relaying capabilities for RPR-compatible applications. The reception of RPR-relayed packets should be possible by any RFC 3550 compatible implementation so what is meant here by uncompatible is RPR-uncompatibility, i.e., the RPR handshake does not take place during call initiation.

## Demos

### Demo 1 - Routing decision done during call initiation

In this demo, a new caller joins to an ongoing call and because the amount of participants
is large enough to saturate the caller's upload bandwidth, it requests packet relaying from
one of the participants.

```
go run cmd/demo1/main.go
```

### Demo 2 - Adaptive routing - New node joins the call, packet relaying must take place

In this demo, the new caller initiates a session with every node and as it notices that
it does not have enough bandwidth, it requests packet relay service from one of the nodes.

On top of this, one of the older participants of a call also notices that it is no longer
able to send the video to everyone so it also requests packet relay service from one of
the nodes.

```
go run cmd/demo2/main.go
```

### Demo 3 - Packet routing with an uncompatible video call application

In this demo, all but one node are RPR-compatible. During SIP message exchange, nodes notice
that one node is of type "INCOMPAT" while RPR-compatible are of type "COMPAT". "COMPAT" nodes
initiate the session with "INCOMPAT" right away and start RTP media transportation but before
starting RTP media transport between "COMPAT" nodes, they perform RPR initiation to agree
on possible packet relaying procedures.

```
go run cmd/demo3/main.go
```

### Demo 4 - Relay-initiated handover

In this demo, a node that is performing packet relaying for another node must leave. Thus it contacts
the client node to inform about this. The client node then contacts other suitable nodes and asks if
they could provide packet relaying for it. Once a suitable relay node is found, the old relay node
is informed about this, it leaves the session and the client nodes starts sending packets to new
relay node.

```
go run cmd/demo4/main.go
```

# Resource-aware Packet Routing Scheme for Large-scale P2P Video Conferences (RPR)

RPR is a scheme for allocating upload bandwidth from other participants of a video call for peers that have a very limited upload bandwidth to enable everyone to send video at acceptable quality.

If you're interested in the idea behind this project, please read [this blogpost about it](https://vizardy.net/blog/scaling_to_infinity.html)

This repository provides an implementation of RPR and a few demo applications that showcase the functionality of the protocol. Because the main point is to showcase the protocol, the implementation is very verbose and cuts some corners. For example, the demo applications assume that a constant 1 Mbps of video is sent and fake values for upload bandwidth are used instead of finding the actual values.
[GoRTP](https://github.com/wernerd/GoRTP) is used as the RTP implementation. Due to my inexperience with this library, I cannot get it to send an RTP packet with an SSRC other than the one that is bound to the created session and that is why CSRC is used to signal the original source instead.

## Interoperability

So what is the value of this if it requires reinventing the wheel for video conferences and breaks compatibility with other software? [RFC 3350](https://tools.ietf.org/html/rfc3550) provides two wonderful concepts: synchronization sources (SSRC) and mixers/translators. SSRCs can be used to distinguish different participants of a session because SSRCs are unique. Translators, on the other hand, receive packets from one or more sources and send these packets to some destination without modifiying the packet. In other words, RFC 3550 supports the packet relaying concept of RPR.

The "only" thing that requires outside the specification signaling is the actual packet relaying agreement between participants. The support for RPR can be signaled e.g. using a SIP INVITE messages which tells the software name and version. Application can assume that if it is conversing with itself (e.g. two Skypes are discussing) that both applications support RPR and packet relaying agreement can be reached. If the SIP INVITE message indicates that the application is conversing with an incompatible implementation (e.g. Skype and Jitsi), RPR packet relay agreement does not take place and the call is initiated the normal way. Demo 3 provides an example of this functionality and it requires **no extra functionality** from an incompatible video conference application whilst providing packet relaying capabilities for RPR-compatible applications. The reception of RPR-relayed packets should be possible by any RFC 3550 compatible implementation so what is meant here by uncompatible is RPR-uncompatibility, i.e. the RPR handshake does not take place during call initiation.

## Demos

For simplicity's sake, the SIP part of the call initiation dialog is removed and only a dummy SIP INVITE is sent when the call is initiated. Once SIP INVITE and SIP 200 OK messages have been exchanged, both participants know each other RPR-related capabilities. If RPR is not supported (or assumed that it is not supported), the call starts immediately as RTP ports for media are transported in the SIP messages.

If, on the other hand, RPR is supported by both participants, the RTP media exchange is preceeded by RPR initiation procedure.

The RPR functionality has been extracted into these very simple application to make them easier to understand. Everything could be crammed into one large application but it's harder to pinpoint different aspects of RPR that way.

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

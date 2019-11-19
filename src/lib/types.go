package lib

import (
	"net"
)

// Message ID values
const (
	SeederPush byte = iota
	Announce
	MsgInterested
	MsgNotInterested
	MsgHave
	PieceRequest
	PieceResponse
	PeerManagerRequest //state change msg
	PeerManagerResponse
	Have
)

type SeederPushMsg struct {
	TrackerAddress net.UDPAddr
	MetaDataFile   []byte
}

type AnnounceMsg struct {
	HavePieceIndex []int
}

type HaveMsg struct {
	pieceIndex int
}

type ConnectionData struct {
	// data       []byte
	// remoteAddr string
	// localAddr  string
	Conn net.Conn
}

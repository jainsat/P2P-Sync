package lib

import (
	"fmt"
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
	Active
	Inactive
	Seeder
	PeerInfoManagerRequest
	PeerInfoManagerResponse
)

type PeerInfoManagerRequestMsg struct {
	State      int
	NumOfPeers int
	Peers      []string
}

type PeerInfoManagerResponseMsg struct {
	Err   string
	Peers []string
}

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

// Doubly circular linkedlist
type Node struct {
	Val        string
	Prev, Next *Node
}

type DCL struct {
	Head, Tail, Itr *Node
	ValToNodeMap    map[string]*Node
}

func NewDCL() *DCL {
	return &DCL{Head: nil, Tail: nil, Itr: nil, ValToNodeMap: make(map[string]*Node)}
}

func (dcl *DCL) Append(val string) bool {
	_, ok := dcl.ValToNodeMap[val]
	if ok {
		GetInstance().Debug("value already exist %v\n", val)
		return false
	}
	node := &Node{Val: val}
	dcl.ValToNodeMap[val] = node
	if dcl.Head == nil {
		// first node
		node.Next = node
		node.Prev = node
		dcl.Head = node
		dcl.Tail = node
		dcl.Itr = node
	} else {
		node.Next = dcl.Head
		node.Prev = dcl.Tail
		dcl.Head.Prev = node
		dcl.Tail.Next = node
		dcl.Tail = node
	}
	return true
}

func (dcl *DCL) Remove(val string) bool {
	node, ok := dcl.ValToNodeMap[val]
	if !ok {
		GetInstance().Debug("Value does not exist: %v\n", val)
		return false
	}
	delete(dcl.ValToNodeMap, val)
	if node == dcl.Head && node == dcl.Tail {
		dcl.Head = nil
		dcl.Tail = nil
	} else {
		node.Prev.Next = node.Next
		node.Next.Prev = node.Prev
		if node == dcl.Head {
			dcl.Head = node.Next
		}
		if node == dcl.Tail {
			dcl.Tail = node.Prev
		}
	}
	return true
}

func (dcl *DCL) Next() string {

	if dcl.Itr == nil {
		GetInstance().Debug("DCL is empty\n")
		return ""
	}
	v := dcl.Itr.Val
	dcl.Itr = dcl.Itr.Next
	return v
}

func (dcl *DCL) Print() {
	node := dcl.Head

	for node != nil {
		fmt.Printf("%v ", node.Val)
		node = node.Next
		if node == dcl.Head {
			break
		}
	}
	fmt.Println()
}

func (dcl *DCL) RPrint() {
	node := dcl.Tail
	for node != nil {
		fmt.Printf("%v ", node.Val)
		node = node.Prev
		if node == dcl.Tail {
			break
		}
	}
	fmt.Println()
}

type Set struct {
	set map[string]bool
}

func NewSet() *Set {
	s := Set{set: make(map[string]bool)}
	return &s
}

func (s *Set) Add(val string) {
	s.set[val] = true
}

func (s *Set) Remove(val string) {
	delete(s.set, val)
}

func (s *Set) getItems() map[string]bool {
	return s.set

}

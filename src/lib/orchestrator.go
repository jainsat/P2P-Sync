package lib

import (
	"bufio"
	"net"
	"strings"
	"time"
)

type Orchestrator struct {
	handler             *MsgHandler
	writeConnectionsMap map[string]chan []byte
	maxConnections      int
	liveConnections     map[string]*ConnectionData
	pieceManager        *PieceManager
}

func NewOrchestrator() *Orchestrator {
	orc := Orchestrator{}
	orc.pieceManager = NewPieceManager()
	orc.handler = NewMsgHandler(orc.pieceManager)
	orc.writeConnectionsMap = make(map[string]chan []byte)
	orc.maxConnections = 5
	orc.liveConnections = make(map[string]*ConnectionData)
	return &orc
}

var (
	delimiter byte = '\n'
)

func parseIp(addr string) string {
	return strings.Split(addr, ":")[0]
}

func (orc *Orchestrator) Listen(peerCh chan *ConnectionData) {
	for {
		recvdConn := <-peerCh
		remoteAddr := recvdConn.Conn.RemoteAddr().String()

		GetLogger().Debug("Received connection with %v on channel\n", remoteAddr)

		// Check if connection with this ip already exists.
		remoteIpOnly := parseIp(remoteAddr)
		GetLogger().Debug("Remote ip = %v\n", remoteIpOnly)
		c, ok := orc.liveConnections[remoteIpOnly]
		if ok {
			GetLogger().Debug("Connection with %v already exists, checking if need to close conn\n", c)
			recvdConn.Conn.Close()
			continue

			// Dont bother telling the remote that you are closing the connection because
			// remote will do the same if there were two connection at it's side.
		} else {
			orc.liveConnections[remoteIpOnly] = recvdConn
		}

		// Bandwidth checker
		if len(orc.writeConnectionsMap) >= orc.maxConnections {
			// Deny and send not interested

			// TBD - Sotya
			continue
		}
		if bufChan, ok := orc.writeConnectionsMap[remoteAddr]; ok {
			// Connection already exists, just pass the corresponding buffer channel
			//GetLogger().Debug("buffered channel", bufChan)
			// This case should not happen.. FATAL
			// TBD
			GetLogger().Debug("I should never be here. %v", bufChan)
		} else {
			// If this is under limit, spawn a new go routine
			// Update my state
			var newCh = make(chan []byte, 1000)
			// Spawn the go routine which justs pushes any incoming data on the
			// buffer to the respective connection
			orc.writeConnectionsMap[remoteAddr] = newCh
			go orc.writeDataOnConnection(newCh, recvdConn.Conn)
		}

		// Trigger manager now with the given connection
		//go
		go orc.readDataOnConnection(recvdConn.Conn, peerCh)
	}
}

func (orc *Orchestrator) writeDataOnConnection(bufChan chan []byte, conn net.Conn) {
	GetLogger().Debug("Write goroutine starting for [%v, %v]\n", conn.LocalAddr().String(), conn.RemoteAddr().String())
	for data := range bufChan {
		// Writing Data to the connection
		conn.Write(data)
	}
}

func (orc *Orchestrator) readDataOnConnection(conn net.Conn, peerCh chan *ConnectionData) {
	// Read the incoming connection into the buffer.
	GetLogger().Debug("Read goroutine starting for [%v, %v]\n", conn.LocalAddr().String(), conn.RemoteAddr().String())
	for {
		buf, err := bufio.NewReader(conn).ReadBytes(delimiter)
		if err != nil {
			GetLogger().Debug("EOF reached.\n")
			break
		}
		GetLogger().Debug("Received message from %v to %v\n", conn.RemoteAddr().String(), conn.LocalAddr().String())

		// Handle message
		orc.handler.HandleMessage(buf, orc.writeConnectionsMap[conn.RemoteAddr().String()], peerCh), conn.RemoteAddr().String())
	}
}

func (orc *Orchestrator) RequestPieces() {
	for {
		for remoteIp := range orc.liveConnections {
			if orc.pieceManager.getNumOfInProgressPieces(remoteIp) <= 3 {
				conn := orc.liveConnections[remoteIp].Conn
				pieceToRequest := orc.pieceManager.getPiece(conn.RemoteAddr().String())
				if pieceToRequest != NoPiece {
					// build a piece request
					// Send it over the channel.
					req := PieceRequestMsg{PieceIndex: pieceToRequest}
					wrCh := orc.writeConnectionsMap[conn.RemoteAddr().String()]
					wrCh <- SerializeMsg(PieceRequest, req)

				}
			}

		}
		time.Sleep(50 * time.Millisecond)
	}
}

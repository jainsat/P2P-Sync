package lib

import (
	"bufio"
	"net"
)

var (
	writeConnectionsMap map[string]chan []byte
	maxConnections      = 5
)

func Listen(ch chan *ConnectionData) {
	for {
		recvdConn := <-ch
		remoteAddr := recvdConn.Conn.RemoteAddr().String()

		// Bandwidth checker
		if len(writeConnectionsMap) >= maxConnections {
			// Deny and send not interested

			// TBD - Sotya
			continue
		}
		if bufChan, ok := writeConnectionsMap[remoteAddr]; ok {
			// Connection already exists, just pass the corresponding buffer channel
			GetInstance().Debug("buffered channel", bufChan)
			// This case should not happen.. FATAL
			// TBD
		} else {
			// If this is under limit, spawn a new go routine
			// Update my state
			var newCh = make(chan []byte, 1000)
			// Spawn the go routine which justs pushes any incoming data on the
			// buffer to the respective connection
			writeConnectionsMap[remoteAddr] = newCh
			go writeDataOnConnection(newCh, recvdConn.Conn)
		}

		// Trigger manager now with the given connection
		//go
		readDataOnConnection(recvdConn.Conn)
	}
}

func writeDataOnConnection(bufChan chan []byte, conn net.Conn) {
	for data := range bufChan {
		// Writing Data to the connection
		conn.Write(data)
	}
}

func readDataOnConnection(conn net.Conn) {
	// Read the incoming connection into the buffer.
	for {
		buf, err := bufio.NewReader(conn).ReadBytes('\n')
		if err != nil {
			GetInstance().Debug("EOF reached. Will no")
			break
		}
		GetInstance().Debug("Received message", string(buf))

		// Handle message
		HandleMessage(buf, writeConnectionsMap[conn.RemoteAddr().String()])
	}
}

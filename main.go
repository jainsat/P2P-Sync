package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

var (
	dataCh chan *ConnectionData
	port   = "2000"
)

type ConnectionData struct {
	// data       []byte
	// remoteAddr string
	// localAddr  string
	conn net.Conn
}

func listen(ch chan *ConnectionData) {

	for {
		recvdConn := <-ch

		remoteAddr := recvdConn.conn.RemoteAddr().String()
		fmt.Println("Recevied a data from", remoteAddr)

		//buf := make([]byte, 1024)
		// Read the incoming connection into the buffer.
		// _, err := recvdConn.conn.Read(buf)
		for {
			buf, err := bufio.NewReader(recvdConn.conn).ReadBytes('\n')
			if err != nil {
				fmt.Println("EOF reached")
				break
			}
			fmt.Println("Received message", string(buf))

			localAddr := recvdConn.conn.LocalAddr().String()
			fmt.Println("Local Address", localAddr)

			// Frame reply
			replyMsg := "Hello " + string(buf) + "\n"

			// Write the message in the connection channel.
			recvdConn.conn.Write([]byte(replyMsg))
			fmt.Println(replyMsg, "written to client")

			// Closing the connection
			// recvdConn.conn.Close()
			fmt.Println("")
		}

	}

}

func main() {
	dataCh = make(chan *ConnectionData)

	go listen(dataCh)

	// Listen on TCP port 2000 on all available unicast and
	// anycast IP addresses of the local system.
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	fmt.Println("P2P-sync daemon listening on port", port)
	for {
		// Wait for a connection.
		fmt.Println("P2P-sync daemon Waiting for message")
		conn, err := l.Accept()
		fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())

		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("")
		dataCh <- &ConnectionData{
			conn: conn,
		}
	}
}

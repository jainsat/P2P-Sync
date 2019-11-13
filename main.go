package main

import (
	"fmt"
	"log"
	"net"
)

var (
	dataCh chan *ConnectionData
	port   = "2000"
)

type ConnectionData struct {
	data       []byte
	remoteAddr string
	localAddr  string
	conn       net.Conn
}

func listen(ch chan *ConnectionData) {

	for {
		recvVal := <-ch
		fmt.Println("Yay. Got a data")
		fmt.Println(string(recvVal.data))

		// Frame reply
		replyMsg := "Hello " + string(recvVal.data)
		// Write the message in the connection channel.
		recvVal.conn.Write([]byte(replyMsg))

		// Closing the connection
		recvVal.conn.Close()
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

	fmt.Println("P2p daemon listening on port", port)
	for {
		// Wait for a connection.
		fmt.Println("Waiting for message")
		conn, err := l.Accept()
		fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())

		if err != nil {
			log.Fatal(err)
		}
		// Handle the connection in a new goroutine.
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently.
		buf := make([]byte, 1024)
		// Read the incoming connection into the buffer.
		_, err = conn.Read(buf)
		if err != nil {
			fmt.Printf("conn.Read failed!!")
			continue
		}
		dataCh <- &ConnectionData{
			remoteAddr: conn.RemoteAddr().String(),
			localAddr:  conn.LocalAddr().String(),
			data:       buf,
			conn:       conn,
		}
	}
}

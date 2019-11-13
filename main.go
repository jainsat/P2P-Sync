package main

import (
	"fmt"
	"log"
	"net"
)

type ConnectionData struct {
	data       []byte
	remoteAddr string
	localAddr  string
}

var (
	dataCh chan *ConnectionData
)

func listen(ch chan *ConnectionData) {

	for {
		data := <-ch
		fmt.Println("Yay. Got a data")
		fmt.Println(data)
	}

}

func main() {
	dataCh = make(chan *ConnectionData)

	//go listen(dataCh)

	// Listen on TCP port 2000 on all available unicast and
	// anycast IP addresses of the local system.
	l, err := net.Listen("tcp", ":2000")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	fmt.Println("Listening on port", 2000)
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
		go func(c net.Conn) {

			buf := make([]byte, 1024)
			// Read the incoming connection into the buffer.
			_, err := conn.Read(buf)
			if err != nil {
				fmt.Println("Error reading:", err.Error())
			}
			fmt.Println("Received message:", string(buf))

			// Frame reply
			replyMsg := "Hello " + string(buf)
			// Write the message in the connection channel.
			conn.Write([]byte(replyMsg))
			c.Close()
		}(conn)
	}
}

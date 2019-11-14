package main

import (
	"fmt"
	"log"
	"net"

	"lib"
)

var (
	dataCh chan *lib.ConnectionData
	port   = "2000"
)

func main() {
	dataCh = make(chan *lib.ConnectionData)

	go lib.Listen(dataCh)

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
		dataCh <- &lib.ConnectionData{
			Conn: conn,
		}
	}
}

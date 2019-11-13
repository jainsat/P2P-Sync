package main

import (
	"fmt"
	"io/ioutil"
	"net"
)

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
	}

}

func main() {

	// connect to this socket
	conn, _ := net.Dial("tcp", "127.0.0.1:2000")

	// Write to a connection
	fmt.Println("Sending message")
	_, err := conn.Write([]byte("ritesh"))
	checkError(err)

	// Read from a  connection
	fmt.Println("Waiting for server to send something")
	result, err := ioutil.ReadAll(conn)
	checkError(err)

	fmt.Println(string(result))
}

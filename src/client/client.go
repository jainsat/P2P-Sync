package main

import (
	"fmt"
	"io/ioutil"
	"lib"
	"net"
	"strconv"
	"sync"
	"time"
)

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
	}

}

var (
	serverAddrs = "127.0.0.1:2000"
)

// Spawnes 10 different connections to the destination server
func multiConnectionsToServer() {
	baseStr := "Hello"
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int, wg *sync.WaitGroup) {
			message := baseStr + strconv.Itoa(i)

			// connect to this socket
			conn, _ := net.Dial("tcp", serverAddrs)

			// Write to a connection
			fmt.Println("Sending message")
			_, err := conn.Write([]byte(message))
			checkError(err)

			// Read from a  connection
			fmt.Println("Waiting for server to send something")
			result, err := ioutil.ReadAll(conn)
			checkError(err)
			fmt.Println(string(result))
			wg.Done()
		}(i, &wg)

	}
	wg.Wait()
}

func readFromConnection(conn net.Conn) []byte {
	result, _ := ioutil.ReadAll(conn)
	if len(result) != 0 {
		return result
	}
	return readFromConnection(conn)
}

func multipleMessagesOnSameConnection() {
	//baseStr := "Hello"
	//var wg sync.WaitGroup

	// connect to this socket
	conn, _ := net.Dial("tcp", serverAddrs)
	defer conn.Close()
	//tcpConn := conn.(*net.TCPConn)
	//tcpConn.SetWriteBuffer(1024 * 10)

	for i := 100; i < 110; i++ {
		//wg.Add(1)
		//go func(i int, wg *sync.WaitGroup) {
		//message := strconv.Itoa(i) + "\n"
		message := getDummyAnnounce()

		// Write to a connection
		fmt.Println("Sending message", message)
		_, err := conn.Write([]byte(message))
		checkError(err)

		time.Sleep(2 * time.Second)

		// Read from a  connection
		// fmt.Println("Waiting for server to send something")

		// result, err := bufio.NewReader(conn).ReadString('\n')
		// checkError(err)
		// fmt.Println(string(result))
		// 	wg.Done()
		// }(i, &wg)

	}
	//wg.Wait()
}

func singleClient() {
	// connect to this socket
	conn, _ := net.Dial("tcp", serverAddrs)

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

func getDummyAnnounce() []byte {
	arr := []int{1, 2, 3}
	a := lib.AnnounceMsg{HavePieceIndex: arr}
	return lib.SerializeMsg(lib.Announce, a)

}

func main() {
	// Single Client trigger
	// singleClient()

	// Multi Client trigger
	// multiConnectionsToServer()
	multipleMessagesOnSameConnection()
}

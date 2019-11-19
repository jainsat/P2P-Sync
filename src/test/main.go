package main

import (
	"fmt"
	"io/ioutil"
	"lib"
	"net/http"
)

func testDCL() {
	dcl := lib.NewDCL()
	dcl.Append("1")
	dcl.Append("2")
	dcl.Append("3")
	dcl.Append("4")
	dcl.Append("5")

	dcl.Print()
	dcl.RPrint()
	fmt.Println(dcl.Next())
	fmt.Println(dcl.Next())
	fmt.Println(dcl.Next())
	fmt.Println(dcl.Next())
	fmt.Println(dcl.Next())
	fmt.Println(dcl.Next())

	fmt.Println(dcl.Next())
}

func runServer() {
	http.HandleFunc("/announce", AnnounceHandler)
	http.ListenAndServe(":9999", nil)
}

func AnnounceHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the request
	body, err := ioutil.ReadAll(r.Body)
	// Deserialize

	fmt.Println("IP ", r.RemoteAddr)
	fmt.Println("Header", r.Header)
}

func main() {

}

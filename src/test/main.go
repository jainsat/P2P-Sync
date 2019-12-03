package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"lib"
	"reflect"
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

// func runServer() {
// 	http.HandleFunc("/announce", AnnounceHandler)
// 	http.ListenAndServe(":9999", nil)
// }

// func AnnounceHandler(w http.ResponseWriter, r *http.Request) {
// 	// Parse the request
// 	body, err := ioutil.ReadAll(r.Body)
// 	// Deserialize

// 	fmt.Println("IP ", r.RemoteAddr)
// 	fmt.Println("Header", r.Header)
// }

func getSeederPush() interface{} {
	seederPush := lib.SeederPushMsg{
		TrackerURL: "http://172.0.1.1:10000/announce",
		// TBD - Read meta data info file
		MetaDataFile: []byte{},
		AmISeeder:    true,
	}
	return seederPush
}
func testSerialize() {

	seederPush := getSeederPush()
	w := new(bytes.Buffer)
	enc := gob.NewEncoder(w)
	enc.Encode(lib.SeederPush)
	enc.Encode(seederPush)
	//enc.Encode("\n")
	//fmt.Printf("%v\n\n", w.Bytes())

	//dict := make(map[string]interface{})
	//dict["msgType"] = lib.SeederPush
	//dict["data"] = seederPush
	f, _ := json.Marshal(seederPush)
	fmt.Printf("%v\n", f)
	var res lib.SeederPushMsg
	//var push lib.SeederPushMsg
	v, _ := json.Marshal(lib.SeederPush)
	//var n uint8
	//n = 10
	//jj, _ := json.Marshal(n)
	jj := []byte{10}
	json.Unmarshal(f, &res)
	kk := append(v, f...)
	kk = append(kk, jj...)
	//fmt.Printf("%v \n", res["data"].(lib.SeederPushMsg))
	fmt.Printf("%v %v %v\n", kk, reflect.TypeOf(res))

	w1 := new(bytes.Buffer)
	enc1 := gob.NewEncoder(w1)
	//enc1.Encode(lib.SeederPush)
	enc1.Encode(seederPush)
	//enc1.Encode("\n")
	//fmt.Printf("%v\n", w1.Bytes())

}

func testPieceManager() {
	pm := lib.NewPieceManager(0)
	// Empty test
	fmt.Println(pm.GetPiece(0))

	//  Add
	pm.HandleHaveMessage(1, 2)
	pm.HandleHaveMessage(1, 1)
	pm.HandleHaveMessage(2, 3)
	pm.HandleHaveMessage(3, 2)
	pm.HandleHaveMessage(3, 3)
	pm.HandleHaveMessage(3, 4)

	fmt.Println(pm.GetPiece(1))
	fmt.Println(pm.GetPiece(1))
	fmt.Println(pm.GetPiece(1))
	fmt.Println(pm.GetPiece(2))
	fmt.Println(pm.GetPiece(3))
	fmt.Println(pm.GetPiece(1))
	fmt.Println(pm.GetPiece(2))
	fmt.Println(pm.GetPiece(3))
	pm.HandleHaveMessage(3, 5)
	fmt.Println(pm.GetPiece(3))
	pm.HandleHaveMessage(1, 6)
	pm.HandleHaveMessage(2, 6)
	pm.HandleHaveMessage(3, 6)
	fmt.Println(pm.GetPiece(1))
	fmt.Println(pm.GetPiece(2))
	fmt.Println(pm.GetPiece(3))
	pm.Notify(false, 2, 3)
	fmt.Println(pm.GetPiece(3))
	pm.Notify(true, 3, 3)
	fmt.Println(pm.GetPiece(3))

}

func main() {
	testPieceManager()

}

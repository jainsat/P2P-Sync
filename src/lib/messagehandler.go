package lib

import (
	"bytes"
	"encoding/gob"
)

func HandleMessage(data []byte, bufChan chan []byte) {
	if len(data) == 0 {

	}
	data = data[:len(data)]
	msgType := data[:1][0]
	switch msgType {
	case SeederPush:
		handleSeederPush(data[1:])
	case Announce:
		handleAnnounce(data[1:])
	default:

	}
}

func SerializeMsg(msgType byte, msg interface{}) []byte {
	w := new(bytes.Buffer)
	enc := gob.NewEncoder(w)
	enc.Encode(msgType)
	enc.Encode(msg)
	return w.Bytes()
}

func handleSeederPush(data []byte) {
	//logger1.Print("Handle seeder push")

}

func handleAnnounce(data []byte) {
	logger1 := GetInstance()

	logger1.Debug("Handle announce")

}

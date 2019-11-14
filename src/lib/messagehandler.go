package lib

import (
	"bytes"
	"encoding/gob"
)

func HandleMessage(data []byte, bufChan chan []byte) {
	if len(data) == 0 {

	}
	msgType := data[:1][0]
	switch msgType {
	case SeederPush:
		handleSeederPush(data[1:len(data)])
	case Announce:
		//handleAnnounce(data[1:])

	}
}

func serializeMsg(msgType int, msg interface{}) []byte {
	w := new(bytes.Buffer)
	enc := gob.NewEncoder(w)
	enc.Encode(msgType)
	enc.Encode(msg)
	return w.Bytes()
}

func handleSeederPush(data []byte) {
	//logger1.Print("Handle seeder push")

}

func HandleAnnounce(data []byte) {
	logger1 := GetInstance()

	logger1.Debug("Handle announce")

}

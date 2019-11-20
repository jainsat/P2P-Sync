package lib

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

func HandleMessage(data []byte, writeChan chan []byte, peerCh chan *ConnectionData) {
	if len(data) == 0 {

	}
	data = data[:len(data)]
	msgType := data[:1][0]
	switch msgType {
	case SeederPush:
		handleSeederPush(data[1:], peerCh)
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

func DeserializeMsg(msgType byte, data []byte) interface{} {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	switch msgType {
	case SeederPush:
		var trackerAddress string
		var metadatafile []byte
		if d.Decode(&trackerAddress) != nil ||
			d.Decode(&metadatafile) != nil {
			GetLogger().Debug("Done parsing SeederPush\n")
		}
		return SeederPushMsg{TrackerAddress: trackerAddress, MetaDataFile: metadatafile}

	default:
		GetLogger().Debug("Unknown message type: %v\n", msgType)

	}

}

func handleSeederPush(data []byte, peerCh chan *ConnectionData) {
	GetLogger().Debug("Handling seeder push message\n")
	res := DeserializeMsg(SeederPush, data).(SeederPushMsg)
	// Contact tracker
	req := PeerInfoManagerResponseMsg{State: Active, NumOfPeers: 6}
	go findPeers(req, res.TrackerAddress, peerCh)
	go processMetaData(res.MetaDataFile)
}

func handleAnnounce(data []byte) {
	GetLogger().Debug("Announce message received\n")

}

func processMetaData(metaData []byte) {

}

func findPeers(req PeerInfoManagerRequestMsg, trackerUrl string, peerCh chan *ConnectionData) {
	bytesRepresentation, err := json.Marshal(req)
	if err != nil {
		GetLogger().Debug("json marshalling failed %v\n", err)
		os.Exit(1)
	}
	var result PeerInfoManagerResponseMsg

	for retry := 5; retry >= 0; retry-- {
		GetLogger().Debug("Contacting tracker at URL %v\n", trackerUrl)
		resp, err := http.Post(trackerUrl, "application/json", bytes.NewBuffer(bytesRepresentation))
		if err != nil {
			GetLogger().Debug("error while doing http post to tracker %v\n", err)
			time.Sleep(time.Second * 1)
			GetLogger().Debug("Retrying again\n")
		}
		json.NewDecoder(resp.Body).Decode(&result)

		if result.Err != "" {
			GetLogger().Debug("Error sent by tracker %v\n", result.Err)
			time.Sleep(time.Second * 1)
			GetLogger().Debug("Retrying again\n")
		} else {
			GetLogger().Debug("Got the peer list %v\n", result.Peers)
			break
		}
	}
	if retry < 0 {
		GetLogger().Debug("Tried connecting with tracker 6 times. Giving up now\n")
		os.Exit(1)
	}
	GetLogger().Debug("Got the list of peers %v\n", peers)
	announceMsg := AnnounceMsg{HavePieceIndex: []int{1, 2, 4}} // hardcoding for now, change it later

	for _, peer := range result.Peers {
		go connect(SerializeMsg(Announce, announceMsg), peer+":2000", peerCh)
	}
}

func connect(msg []byte, peerAddr string, peerCh chan *ConnectionData) {
	for retry := 5; retry >= 0; retry-- {
		conn, e := net.Dial("tcp", peerAddr)
		if e == nil {
			// Write to a connection
			fmt.Println("Connected with  %v\n", peerAddr)

			// TODO: Should we set a deadline here.
			_, err := conn.Write(msg)
			if err != nil {
				GetLogger().Debug("Error while writing to %v : %v\n", peerAddr, err)
			} else {
				GetLogger().Debug("Sent announce message to %v\n", peerAddr)
				// Pass new connection to orchestrator.
				peerCh <- ConnectionData{Conn: conn}
				break
			}
		} else {
			GetLogger().Debug("Error occurred while connecting with %v\n", peerAddr)
		}
		// Sleep for sometime and then retry
		time.Sleep(1 * time.Second)
	}
	if retry < 0 {
		GetLogger().Debug("Tried connecting with %v 6 times. Giving up now\n", peerAddr)
		os.Exit(1)
	}
}

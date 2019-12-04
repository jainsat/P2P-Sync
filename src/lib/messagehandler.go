package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

type MsgHandler struct {
	pieceManager *PieceManager
}

func NewMsgHandler(pieceManager *PieceManager) *MsgHandler {
	return &MsgHandler{pieceManager: pieceManager}
}

func (mh *MsgHandler) HandleMessage(data []byte, writeChan chan []byte, peerCh chan *ConnectionData, remoteIp string) {
	if len(data) == 0 {

	}
	var msgType byte
	GetLogger().Debug("Handle message data=%v\n", data)
	json.Unmarshal(data[0:1], &msgType)
	GetLogger().Debug("Message type = %v, seeder push=%v\n", msgType, SeederPush)
	data = data[1 : len(data)-1]

	switch msgType {
	case SeederPush:
		mh.handleSeederPush(data, peerCh)
	case Announce:
		mh.handleAnnounce(data)
	case Have:
		mh.handleHaveMessage(data, remoteIp)
	default:

	}
}

func SerializeMsg(msgType byte, msg interface{}) []byte {
	// w := new(bytes.Buffer)
	// enc := gob.NewEncoder(w)
	// enc.Encode(msgType)
	// enc.Encode(msg)
	// enc.Encode("\n")
	// return w.Bytes()
	t, err1 := json.Marshal(msgType)
	ms, err2 := json.Marshal(msg)
	//dm, err3 := json.Marshal('\n')

	if err1 != nil || err2 != nil {
		GetLogger().Debug("Json marshalling failing, %v, %v\n", err1, err2)
		os.Exit(1)
	}
	vv := []byte{10}
	final := append(t, ms...)
	//final = append(final, dm...)
	final = append(final, vv...)
	return final

}

func DeserializeMsg(msgType byte, data []byte) interface{} {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return ""
	}
	// r := bytes.NewBuffer(data)
	// d := gob.NewDecoder(r)
	// switch msgType {
	// case SeederPush:
	// 	var trackerAddress string
	// 	var metadatafile []byte
	// 	if d.Decode(&trackerAddress) != nil ||
	// 		d.Decode(&metadatafile) != nil {
	// 		GetLogger().Debug("Error parsing SeederPush\n")
	// 	}
	// 	return SeederPushMsg{TrackerURL: trackerAddress, MetaDataFile: metadatafile}

	// default:
	// 	GetLogger().Debug("Unknown message type: %v\n", msgType)

	// }
	// return ""
	var err error
	switch msgType {
	case SeederPush:
		var msg SeederPushMsg
		err = json.Unmarshal(data, &msg)
		if err == nil {
			return msg
		}
	case Have:
		var msg HaveMsg
		err = json.Unmarshal(data, &msg)
		if err == nil {
			return msg
		}

	}

	GetLogger().Debug("Error occurred while deserializing msgtype=%v err=%v\n", msgType, err)
	os.Exit(1)
	return -1
}

func (mh *MsgHandler) handleSeederPush(data []byte, peerCh chan *ConnectionData) {
	GetLogger().Debug("Handling seeder push message\n")
	res := DeserializeMsg(SeederPush, data).(SeederPushMsg)
	// Contact tracker
	var state byte
	if res.AmISeeder {
		state = Seeder
	} else {
		state = Active
	}
	req := PeerInfoManagerRequestMsg{State: state, NumOfPeers: 6}
	go findPeers(req, res.TrackerURL, peerCh)
	go processMetaData(res.MetaDataFile)
}

func (mh *MsgHandler) handleAnnounce(data []byte) {
	GetLogger().Debug("Announce message received\n")

}

func processMetaData(metaData []byte) {

}

func (mh *MsgHandler) handleHaveMessage(data []byte, peer string) {
	GetLogger().Debug("Have message received\n")
	res := DeserializeMsg(Have, data).(HaveMsg)
	GetLogger().Debug("Have message = %v\n", res)
	mh.pieceManager.updatePieceInfo(peer, res.PieceIndex)
}

func findPeers(req PeerInfoManagerRequestMsg, trackerUrl string, peerCh chan *ConnectionData) {
	bytesRepresentation, err := json.Marshal(req)
	if err != nil {
		GetLogger().Debug("json marshalling failed %v\n", err)
		os.Exit(1)
	}
	var result PeerInfoManagerResponseMsg
	var retry int
	for retry = 5; retry >= 0; retry-- {
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
	GetLogger().Debug("Got the list of peers %v\n", result.Peers)
	announceMsg := AnnounceMsg{HavePieceIndex: []int{1, 2, 4}} // hardcoding for now, change it later

	for _, peer := range result.Peers {
		go connect(SerializeMsg(Announce, announceMsg), peer+":2000", peerCh)
	}
}

func connect(msg []byte, peerAddr string, peerCh chan *ConnectionData) {
	var retry int
	for retry = 5; retry >= 0; retry-- {
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
				peerCh <- &ConnectionData{Conn: conn}
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

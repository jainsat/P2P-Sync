package lib

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type Peer struct {
	writeConnectionsMap map[string]chan []byte
	maxConnections      int
	liveConnections     map[string]*ConnectionData
	pieceManager        *PieceManager
	trackerURL          string
	metaData            FileMetaInfo
	fileIndexBytes      map[int][]byte
}

func NewPeer() *Peer {
	peer := Peer{}
	peer.pieceManager = NewPieceManager()
	peer.writeConnectionsMap = make(map[string]chan []byte)
	peer.maxConnections = 5
	peer.liveConnections = make(map[string]*ConnectionData)
	peer.fileIndexBytes = make(map[int][]byte)
	return &peer
}

var (
	delimiter byte = '\n'
	PieceSize      = 50 * 1024
)

func parseIp(addr string) string {
	return strings.Split(addr, ":")[0]
}

func (peer *Peer) Listen(peerCh chan *ConnectionData) {
	for {
		recvdConn := <-peerCh
		remoteAddr := recvdConn.Conn.RemoteAddr().String()

		GetLogger().Debug("Received connection with %v on channel\n", remoteAddr)

		// Check if connection with this ip already exists.
		remoteIpOnly := parseIp(remoteAddr)
		GetLogger().Debug("Remote ip = %v\n", remoteIpOnly)
		c, ok := peer.liveConnections[remoteIpOnly]
		if ok {
			GetLogger().Debug("Connection with %v already exists, checking if need to close conn\n", c)
			recvdConn.Conn.Close()
			continue

			// Dont bother telling the remote that you are closing the connection because
			// remote will do the same if there were two connection at it's side.
		} else {
			peer.liveConnections[remoteIpOnly] = recvdConn
		}

		// Bandwidth checker
		if len(peer.writeConnectionsMap) >= peer.maxConnections {
			// Deny and send not interested

			// TBD - Sotya
			continue
		}
		if bufChan, ok := peer.writeConnectionsMap[remoteAddr]; ok {
			// Connection already exists, just pass the corresponding buffer channel
			//GetLogger().Debug("buffered channel", bufChan)
			// This case should not happen.. FATAL
			// TBD
			GetLogger().Debug("I should never be here. %v", bufChan)
		} else {
			// If this is under limit, spawn a new go routine
			// Update my state
			var newCh = make(chan []byte, 1000)
			// Spawn the go routine which justs pushes any incoming data on the
			// buffer to the respective connection
			peer.writeConnectionsMap[remoteAddr] = newCh
			go peer.writeDataOnConnection(newCh, recvdConn.Conn)
		}

		// Trigger manager now with the given connection
		//go
		go peer.readDataOnConnection(recvdConn.Conn, peerCh)
	}
}

func (peer *Peer) writeDataOnConnection(bufChan chan []byte, conn net.Conn) {
	GetLogger().Debug("Write goroutine starting for [%v, %v]\n", conn.LocalAddr().String(), conn.RemoteAddr().String())
	for data := range bufChan {
		// Writing Data to the connection
		conn.Write(data)
	}
}

func intToBytes(a uint32) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, a)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}
	return buf.Bytes()
}

func bytesToInt(buf []byte) uint32 {
	val := binary.BigEndian.Uint32(buf)
	return val
}

func (peer *Peer) readDataOnConnection(conn net.Conn, peerCh chan *ConnectionData) {
	// Read the incoming connection into the buffer.
	GetLogger().Debug("Read goroutine starting for [%v, %v]\n", conn.LocalAddr().String(), conn.RemoteAddr().String())
	for {
		// First read msg size
		sbuf := make([]byte, 4)
		_, err := conn.Read(sbuf)
		if err != nil {
			GetLogger().Debug("EOF reached.\n")
			break
		}
		size := bytesToInt(sbuf)

		GetLogger().Debug("size of message = %v\n", size)

		buf := make([]byte, size)
		var n uint32
		n = uint32(0)
		for n < size {
			n1, err := conn.Read(buf[n:])
			n = n + uint32(n1)
			GetLogger().Debug("bytes read =%v\n", n1)
			if err != nil {
				GetLogger().Debug("EOF reached.\n")
				break
			}

		}
		//GetLogger().Debug("buffer = %v\n", buf)
		GetLogger().Debug("Received message from %v to %v\n", conn.RemoteAddr().String(), conn.LocalAddr().String())

		// Handle message
		peer.HandleMessage(buf, peer.writeConnectionsMap[conn.RemoteAddr().String()], peerCh, conn.RemoteAddr().String())
	}
}

func (peer *Peer) RequestPieces() {
	for {
		for remoteIp := range peer.liveConnections {
			if peer.pieceManager.GetNumOfInProgressPieces(remoteIp) <= 3 {
				conn := peer.liveConnections[remoteIp].Conn
				pieceToRequest := peer.pieceManager.GetPiece(conn.RemoteAddr().String())
				if pieceToRequest != NoPiece {
					// build a piece request
					// Send it over the channel.
					GetLogger().Debug("Requesting Piece: %v\n", pieceToRequest)
					req := PieceRequestMsg{PieceIndex: pieceToRequest}
					wrCh := peer.writeConnectionsMap[conn.RemoteAddr().String()]

					serializeMsg := SerializeMsg(PieceRequest, req)
					//GetLogger().Debug("Serialized Piece: %v\n", serializeMsg)
					wrCh <- serializeMsg
				}
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (p *Peer) HandleMessage(data []byte, writeChan chan []byte, peerCh chan *ConnectionData, remoteIp string) {
	if len(data) == 0 {

	}
	var msgType byte
	GetLogger().Debug("Handle message data: %v\n", data[0:10])
	json.Unmarshal(data[0:1], &msgType)
	GetLogger().Debug("Message type = %v, seeder push=%v\n", msgType, SeederPush)
	data = data[1:]

	switch msgType {
	case SeederPush:
		p.handleSeederPush(data, peerCh, remoteIp)
	case Announce:
		p.handleAnnounce(data, remoteIp)
	case Have:
		p.handleHaveMessage(data, remoteIp)
	case PieceRequest:
		p.handlePieceRequest(data, remoteIp)
	case PieceResponse:
		p.handlePieceResponse(data, remoteIp, peerCh)
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
	//GetLogger().Debug("JSON Msg: %v", ms)
	//GetLogger().Debug("JSON Msg String: %v", string(ms))
	if err1 != nil || err2 != nil {
		GetLogger().Debug("Json marshalling failing, %v, %v\n", err1, err2)
		os.Exit(1)
	}
	//vv := []byte{10}
	final := append(t, ms...)
	//GetLogger().Debug("JSON Final String: %v", string(final))
	//final = append(final, dm...)
	sizeBuf := intToBytes(uint32(len(final)))
	GetLogger().Debug("Serialize size=%v\n", len(final))
	final = append(sizeBuf, final...)
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
	//GetLogger().Debug("Deserialize bytes in json: %v", string(data))
	//GetLogger().Debug("Deserialize bytes: %v", data)
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
	case Announce:
		var msg AnnounceMsg
		err = json.Unmarshal(data, &msg)
		if err == nil {
			return msg
		}
	case PieceRequest:
		var msg PieceRequestMsg
		err = json.Unmarshal(data, &msg)
		if err == nil {
			return msg
		}
	case PieceResponse:
		var msg PieceResponseMsg
		err = json.Unmarshal(data, &msg)
		if err == nil {
			return msg
		}
	}

	GetLogger().Debug("Error occurred while deserializing msgtype=%v err=%v\n", msgType, err)
	os.Exit(1)
	return -1
}

// updateFileIndexBytes
func (p *Peer) updateFileIndexBytes(fname string) error {
	index := 0
	f, err := os.Open(fname)
	defer f.Close()
	if err != nil {
		fmt.Println("Error while opening file: ", err)
		return err
	}
	for {
		var buf []byte
		if index == int(p.metaData.TotalPieces)-1 {
			buf = make([]byte, int(p.metaData.LastPieceSize))
		} else {
			buf = make([]byte, PieceSize)
		}

		n, err := f.Read(buf)
		if n > 0 {
			p.fileIndexBytes[index] = buf
			// if index == 0 {
			// 	GetLogger().Debug("updateFileIndexBytes Piece 0 string val: %v\n", string(buf))
			// 	GetLogger().Debug("updateFileIndexBytes Buffer: %v\n", buf)
			// }
		}
		if err == io.EOF {
			GetLogger().Debug("Reached EOF while Creating pieces\n")
			break
		}
		if err != nil {
			GetLogger().Debug("Error while creating pieces: %v\n", err)
			return err
		}
		index++
	}
	GetLogger().Debug("updateFileIndexBytes successfully completed.\n")
	return nil
}

// frameSeederPush returns a SeederPush message
func (p *Peer) frameSeederPush(amIStarter bool) *SeederPushMsg {
	return &SeederPushMsg{
		TrackerURL: p.trackerURL,
		AmIStarter: amIStarter,
		MetaData:   p.metaData,
	}
}

func (p *Peer) handleSeederPush(data []byte, peerCh chan *ConnectionData, remoteIp string) {
	GetLogger().Debug("Handling seeder push message\n")
	res := DeserializeMsg(SeederPush, data).(SeederPushMsg)
	// Contact tracker
	var state byte
	// Set the trackerURL and metadata
	p.trackerURL = res.TrackerURL
	p.metaData = res.MetaData
	GetLogger().Debug("Meta Data: %+v\n", p.metaData)
	if res.AmIStarter {
		state = Seeder
		// Read the file and make a map of index to pieces
		// Do not perform this if my starting state is not a seeder
		p.updateFileIndexBytes(res.MetaData.Name)
		p.pieceManager.ReceivedAllPieces(int(p.metaData.TotalPieces))

	} else {
		state = Active
		// Setting all pieces to true for active
		var allPieces []int
		for i := 0; i < int(p.metaData.TotalPieces); i++ {
			allPieces = append(allPieces, i)
		}
		p.pieceManager.UpdatePieceInfos(remoteIp, allPieces)

	}

	myPeers := make(map[string]bool)
	for remoteIp := range p.liveConnections {
		myPeers[remoteIp] = true
	}
	req := PeerInfoManagerRequestMsg{State: state, NumOfPeers: 6, Peers: myPeers}
	go p.findPeers(req, res.TrackerURL, peerCh)
	//go processMetaData(res.MetaData)
}

// handleAnnounce updates the piece info for the given peer
func (p *Peer) handleAnnounce(data []byte, peer string) {
	GetLogger().Debug("Announce message received\n")
	res := DeserializeMsg(Announce, data).(AnnounceMsg)
	// Update piecemanager
	p.pieceManager.UpdatePieceInfos(peer, res.HavePieceIndex)
}

// frameAnnounce for a given peer checks myPieces from PieceManager and then
// creates a piecesIndex with the indexes of the piece that the given peer has
func (p *Peer) frameAnnounce(peer string) *AnnounceMsg {
	// Check for pieces here
	mpc := p.pieceManager.myPieces
	piecesIndex := []int{}
	for k, v := range mpc {
		if v {
			piecesIndex = append(piecesIndex, k)
		}
	}
	// Return Announce Message
	return &AnnounceMsg{piecesIndex}
}

func (peer *Peer) handlePieceResponse(data []byte, remoteIp string, peerCh chan *ConnectionData) {
	GetLogger().Debug("Piece response message received from %v\n", remoteIp)
	res := DeserializeMsg(PieceResponse, data).(PieceResponseMsg)
	// Notify pieceManager
	peer.fileIndexBytes[res.PieceIndex] = res.PieceData
	// Send have message to those who does not have this piece.
	peer.pieceManager.Notify(true, remoteIp, res.PieceIndex)
	if res.PieceIndex == 0 {
		GetLogger().Debug("Received Piece 0: %v\n.", res.PieceData)
	}
	GetLogger().Debug("Piece response message received from %v, Index:%v, DataSize:%v\n", remoteIp, res.PieceIndex, len(res.PieceData))
	go peer.sendHaveMessage(res.PieceIndex)
	if peer.pieceManager.GetTotalCurrentPieces() == peer.metaData.TotalPieces {
		GetLogger().Debug("Became a SEEDER. Calling aggregator\n.")
		peer.aggregatePieces()
		// We have become a seeder
		// Contact tracker with state as seed. Get Peers.
		req := PeerInfoManagerRequestMsg{State: Seeder, NumOfPeers: 6}
		// This will send a seeder push to the peers.
		go peer.findPeers(req, peer.trackerURL, peerCh)
	}
}

func (peer *Peer) handlePieceRequest(data []byte, remoteIp string) {
	GetLogger().Debug("Piece request message received from %v\n", remoteIp)
	res := DeserializeMsg(PieceRequest, data).(PieceRequestMsg)
	GetLogger().Debug("Piece request for : %v", res.PieceIndex)
	// Check if you have this piece
	// Most probably, you should have
	// Log a warning if not
	if peer.pieceManager.HavePiece(res.PieceIndex) {
		pieceResponse := PieceResponseMsg{}
		pieceResponse.PieceIndex = res.PieceIndex
		// Get the data from mapper and populate.
		pieceResponse.PieceData = peer.fileIndexBytes[res.PieceIndex]
		if res.PieceIndex == 0 {
			GetLogger().Debug("Request for Piece 0: %v\n.", pieceResponse.PieceData)
		}
		GetLogger().Debug("Piece request for Index:%v, DataSize:%v\n", res.PieceIndex, len(pieceResponse.PieceData))
		data := SerializeMsg(PieceResponse, pieceResponse)
		GetLogger().Debug("Handle piece req=%v\n", data[:10])
		peer.writeConnectionsMap[remoteIp] <- data
	} else {
		GetLogger().Debug("Dont have piece %v, request from %v\n", res.PieceIndex, remoteIp)
	}

}

func (peer *Peer) sendHaveMessage(piece int) {
	// Send only to those who don't have it.
	peersWhichHavePiece := peer.pieceManager.GetPeers(piece)
	haveMsg := HaveMsg{PieceIndex: piece}
	haveMsgBytes := SerializeMsg(Have, haveMsg)
	for n, c := range peer.liveConnections {
		GetLogger().Debug("Sending have message to %v\n", n)
		conn := c.Conn
		if !peersWhichHavePiece[conn.RemoteAddr().String()] {
			peer.writeConnectionsMap[conn.RemoteAddr().String()] <- haveMsgBytes
		}
	}
}

func (p *Peer) handleHaveMessage(data []byte, peer string) {
	GetLogger().Debug("Have message received\n")
	res := DeserializeMsg(Have, data).(HaveMsg)
	GetLogger().Debug("Have message = %v\n", res)
	p.pieceManager.UpdatePieceInfo(peer, res.PieceIndex)
}

func (p *Peer) findPeers(req PeerInfoManagerRequestMsg, trackerUrl string, peerCh chan *ConnectionData) {
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
			continue
		}
		GetLogger().Debug("Resp: %v, Error: %v\n", resp, err)

		json.NewDecoder(resp.Body).Decode(&result)

		if result.Err != "" {
			GetLogger().Debug("Error sent by tracker %v\n", result.Err)
			time.Sleep(time.Second * 1)
			GetLogger().Debug("Retrying again\n")
			continue
		} else {
			GetLogger().Debug("Got the peer list %v\n", result.Peers)
			break
		}
	}
	if retry < 0 {
		GetLogger().Debug("Tried connecting with tracker 6 times. Giving up now\n")
		return
	}
	GetLogger().Debug("Got the list of peers: %v, length: %v\n", result.Peers, len(result.Peers))

	if len(result.Peers) == 0 {
		// No peers to try
		// Retry after a while
		//time.Sleep(2 * time.Second)
		//return
	} else {
		// Check state. If seeder, Send seederpush else, send announce
		if req.State == Seeder {
			for _, peer := range result.Peers {
				// Send SeederPush
				spMsg := p.frameSeederPush(false)
				serlzMsg := SerializeMsg(SeederPush, *spMsg)
				GetLogger().Debug("Sending SeederPush: %v\n", serlzMsg)
				go connect(serlzMsg, peer+":2000", peerCh, "SeederPush")
			}
		} else {
			for _, peer := range result.Peers {
				announceMsg := p.frameAnnounce(peer)
				serlzMsg := SerializeMsg(Announce, *announceMsg)
				GetLogger().Debug("Sending announce: %v\n", serlzMsg)
				go connect(serlzMsg, peer+":2000", peerCh, "Announce")
			}
		}
	}
}

func connect(msg []byte, peerAddr string, peerCh chan *ConnectionData, msgType string) {
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
				GetLogger().Debug("Sent %v to %v\n", msgType, peerAddr)
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

// aggregatePieces writes all bytes to a file and dumps it
// Should be called once a given peer has all the pieces for a given file
func (p *Peer) aggregatePieces() error {
	f, err := os.Create(p.metaData.Name)
	if err != nil {
		return err
	}
	defer f.Close()

	var keys []int
	for k := range p.fileIndexBytes {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, k := range keys {
		_, err := f.Write(p.fileIndexBytes[k])
		if k == 0 {
			GetLogger().Debug("Dump Piece 0: %v\n", p.fileIndexBytes[k])
			GetLogger().Debug("Piece 0 string val: %v\n", string(p.fileIndexBytes[k]))
		}
		if err != nil {
			GetLogger().Debug("Error in aggregator\n")
			return err
		}
	}
	GetLogger().Debug("Aggregator: File successfully created\n")
	return nil
}

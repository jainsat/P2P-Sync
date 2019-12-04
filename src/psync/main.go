package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"lib"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
)

var (
	config     = flag.String("config", "", "name of the json config file that contains IP of all other peers")
	fileToSync = flag.String("file", "", "name of the file to sync across the peers mentione in the config file")

	completionChan chan int
	logger         = lib.GetCustomLogger("/var/log/psync.log")
	peers          []string

	httpServerPort = "10000"
	peerInfoManger *lib.PeerInfoManager
	once           sync.Once
)

// getLocalIP returns the local IPv4 Adress
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger.Debug("Failed to get interface addresses: %v\n", err)
		return ""
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

// createPieces returns total pieces and size of last piece in FileMetaInfo
func createPieces(file *os.File) (lib.FileMetaInfo, error) {
	// Read the given file
	fi, err := file.Stat()
	if err != nil {
		// Could not obtain stat, handle error
		return lib.FileMetaInfo{}, err
	}

	totalPieces := fi.Size() / int64(lib.PieceSize)
	lastPieceSize := fi.Size() % int64(lib.PieceSize)
	if lastPieceSize != 0 {
		totalPieces++
	}
	fmi := lib.FileMetaInfo{
		TotalPieces:   totalPieces,
		LastPieceSize: lastPieceSize,
	}
	logger.Debug("createPieces: Done calculating totalPieces: %v, lastPieceSize: %v", totalPieces, lastPieceSize)
	return fmi, nil
}

// sendSeederPush sends a Seeder Push message to the daemon running
// on the same machine
func sendSeederPush(file *os.File) error {
	// Dial on localhost
	conn, err := net.Dial("tcp", "127.0.0.1:2000")
	if err != nil {
		logger.Debug("Failed to dial connection to port 2000: %v\n", err)
	}

	// Write to a connection
	logger.Debug("Sending SeederPush to localhost\n")
	myIP := getLocalIP()

	// Need PeerInfoManager port as well!!
	if myIP == "" {
		// Cannot proceed. Flag error here.
		return errors.New("Did not find any local IP address")
	}
	myIP = myIP + ":" + httpServerPort
	url := strings.Join([]string{myIP, "announce"}, "/")
	url = "http://" + url
	logger.Debug("Sending PeerPushMessage with url: %v\n", url)

	// Generate the MetaInfo
	metaInfo, err := createPieces(file)
	if err != nil {
		logger.Debug("Failed to Create pieces: %v\n", err)
		return err
	}
	// Set the file name
	metaInfo.Name = file.Name()
	// Frame SeederPush message
	seederPush := lib.SeederPushMsg{
		TrackerURL: url,
		// TBD - Read meta data info file
		MetaData:   metaInfo,
		AmIStarter: true,
	}
	bytes := lib.SerializeMsg(lib.SeederPush, seederPush)
	logger.Debug("Bytes sending : %v\n", bytes)
	_, err = conn.Write(bytes)
	if err != nil {
		return err
	}
	// TBD - Do we need to read back from server here??
	return nil
}

// GetPeerInfoManager
func GetPeerInfoManager() *lib.PeerInfoManager {
	myIP := getLocalIP()
	if myIP == "" {
		fmt.Println("Error: No Local IP found")
		return nil
	}
	once.Do(func() {
		peerInfoManger = lib.NewPeerInfoManager(peers, completionChan, myIP)
		//syncLogger = createLogger("/Users/riteshsinha/git/SBU/cse534/P2P-Sync/sync.log")
	})
	return peerInfoManger
}

// runServer starts a HTTP server with /announce path for the peers to connect
// to the tracker
func runServer() {
	http.HandleFunc("/announce", AnnounceHandler)
	http.ListenAndServe(":"+httpServerPort, nil)
}

// AnnounceHandler is the handler function for /announce path
func AnnounceHandler(w http.ResponseWriter, r *http.Request) {
	// Get the request from Body
	decoder := json.NewDecoder(r.Body)
	var req lib.PeerInfoManagerRequestMsg
	err := decoder.Decode(&req)
	if err != nil {
		fmt.Println("Error while decoding json in the request at server: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logger.Debug("Request at server: %v from IP %v\n", req, r.RemoteAddr)
	logger.Debug("Header: %v\n", r.Header)
	req.IpAddress = strings.Split(r.RemoteAddr, ":")[0]

	logger.Debug("Request: %v\n", req)
	pim := GetPeerInfoManager()
	resp := pim.HandleRequest(req)

	logger.Debug("Response: %v\n", resp)
	bytesRepresentation, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	//Set Content-Type header so that clients will know how to read response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	//Write json response back to response
	w.Write(bytesRepresentation)
}

// parseConfig parses the config.json files and returns a list of IP addresses
func parseConfig() []string {
	file, err := os.Open(*config)
	if err != nil {
		logger.Debug("Could not open the config file: %v\n", err)
		return []string{}
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	configuration := lib.PeerConfig{}
	err = decoder.Decode(&configuration)
	if err != nil {
		logger.Debug("Could not decode the config into json: %v\n", err)
		return []string{}
	}
	return configuration.PeersList
}

// run species the main flow of psync utility. It does the following -
// i) Parse Config.
// ii) Spawn a tracker(Peer Info Manager)
// iii) Send SeederPush to myself.
// iv) Wait on channel.
func run(f *os.File) {
	fmt.Println("---------- P2P sync ----------")
	fmt.Println("Reading config file", *config)
	peers = parseConfig()
	fmt.Println("Peers List", peers)

	// TBD - Pass this to PeerInfoManager. Spawn a Routine for that.
	// Also pass a channel.
	fmt.Println("Starting PeerInfoManager")
	go runServer()

	// Send Seeder push
	err := sendSeederPush(f)
	if err != nil {
		logger.Debug("SeederPush failed: ", err)
	}
	fmt.Println("Starting transfer to other peers")

	//Wait on the channel
	<-completionChan
	// fmt.Println(fileToSync, "successfully synced to ", len(peers), "peers")
}
func main() {
	flag.Parse()

	// Checks for the flag being empty
	if *config == "" {
		fmt.Println("No config file name found in args. Please specify one. Exiting.")
		return
	}
	if *fileToSync == "" {
		fmt.Println("No file name found. Please specify one. Exiting")
		return
	}
	// Read file
	f, err := os.Open(*fileToSync)
	defer f.Close()
	if err != nil {
		fmt.Println("Error while opening file: ", err)
		return
	}
	// Trigger the run function
	run(f)
}

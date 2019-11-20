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
)

var (
	config     = flag.String("config", "", "name of the json config file that contains IP of all other peers")
	fileToSync = flag.String("file", "", "name of the file to sync across the peers mentione in the config file")

	completionChan chan int
	logger         = lib.GetLogger()
	peers          []string

	httpServerPort = "10000"
)

// getLocalIP returns the local IPv4 Adress
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger.Debug("Failed to get interface addresses: %v", err)
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

// sendSeederPush sends a Seeder Push message to the daemon running
// on the same machine
func sendSeederPush() error {
	// Dial on localhost
	conn, err := net.Dial("tcp", "127.0.0.1:2000")
	if err != nil {
		logger.Debug("Failed to dial connection to port 2000: %v", err)
	}

	// Write to a connection
	logger.Debug("Sending SeederPush to localhost")
	myIP := getLocalIP()

	// Need PeerInfoManager port as well!!
	if myIP == "" {
		// Cannot proceed. Flag error here.
		return errors.New("Did not find any local IP address")
	}
	myIP = myIP + ":" + httpServerPort
	url := strings.Join([]string{myIP, "announce"}, "/")
	url = "http://" + url
	logger.Debug("Sending PeerPushMessage with url: %v", url)
	// Frame SeederPush message
	seederPush := lib.SeederPushMsg{
		TrackerURL: url,
		// TBD - Read meta data info file
		MetaDataFile: []byte{},
		AmISeeder:    true,
	}

	_, err = conn.Write(lib.SerializeMsg(lib.SeederPush, seederPush))
	if err != nil {
		return err
	}
	// TBD - Do we need to read back from server here??
	return nil
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

	fmt.Println("Request at server: ", req, "from IP", r.RemoteAddr)
	fmt.Println("Header", r.Header)
	req.IpAddress = strings.Split(r.RemoteAddr, ":")[0]

	myIP := getLocalIP()
	if myIP == "" {
		fmt.Println("Error: No Local IP found")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	pim := lib.NewPeerInfoManager(peers, completionChan, myIP)
	logger.Debug("Request", req)
	resp := pim.HandleRequest(req)

	fmt.Println("Response: ", resp)
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
		logger.Debug("Could not open the config file: %v", err)
		return []string{}
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	configuration := lib.PeerConfig{}
	err = decoder.Decode(&configuration)
	if err != nil {
		logger.Debug("Could not decode the config into json: %v", err)
		return []string{}
	}
	return configuration.PeersList
}

// run species the main flow of psync utility. It does the following -
// i) Parse Config.
// ii) Spawn a tracker(Peer Info Manager)
// iii) Send SeederPush to myself.
// iv) Wait on channel.
func run() {
	fmt.Println("---------- P2P sync ----------")
	fmt.Println("Reading config file", *config)
	peers = parseConfig()
	fmt.Println("Peers List", peers)

	// TBD - Pass this to PeerInfoManager. Spawn a Routine for that.
	// Also pass a channel.
	fmt.Println("Starting PeerInfoManager")
	go runServer()

	// Send Seeder push
	err := sendSeederPush()
	if err != nil {
		logger.Debug("SeederPush failed: ", err)
	}
	fmt.Println("Starting transfer to other peers")

	// // Wait on the channel
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

	// Trigger the run function
	run()
}

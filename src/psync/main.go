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
	//flag.StringVar(&svar, "svar", "bar", "a string var")
	config     = flag.String("config", "", "name of the json config file that contains IP of all other peers")
	fileToSync = flag.String("file", "", "name of the file to sync across the peers mentione in the config file")

	completionChan chan int
	logger         = lib.GetLogger()
	peers          []string

	httpServerPort = "10000"
)

type PeerConfig struct {
	PeersList []string
}

// getLocalIP returns the local IPv4 Adress
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger.Debug("Failed to get interface addresses: ", err)
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

func sendSeederPush() error {
	conn, err := net.Dial("tcp", ":2000")
	if err != nil {
		logger.Debug("Failed to dial connection to port 2000: ", err)
	}
	// Write to a connection
	logger.Debug("Sending SeederPush to localhost")
	myIP := getLocalIP()

	// Need PeerInfoManager port as well!!
	// TBD - Do we keep this in the peerConfig.json file.
	if myIP == "" {
		// Cannot proceed. Flag error here.
		return errors.New("Did not find any local IP address")
	}
	url := strings.Join([]string{myIP, "announce"}, "/")
	// Frame SeederPush message
	seederPush := lib.SeederPushMsg{
		TrackerAddress: url,
		MetaDataFile:   []byte{},
		// TBD - Read meta data info file
	}

	_, err = conn.Write(lib.SerializeMsg(lib.SeederPush, seederPush))
	if err != nil {
		return err
	}
	// TBD - Do we need to read back from server here??
	return nil
}

func runServer() {
	http.HandleFunc("/announce", AnnounceHandler)
	http.ListenAndServe(":"+httpServerPort, nil)
}

func AnnounceHandler(w http.ResponseWriter, r *http.Request) {
	// Get the request from Body
	decoder := json.NewDecoder(r.Body)
	var req lib.PeerInfoManagerRequestMsg
	err := decoder.Decode(&req)
	if err != nil {
		fmt.Println("Error while decoding json in the request at server: ", err)
		w.WriteHeader(http.StatusBadRequest)
		// Write error in writer
		// TBD
	}

	fmt.Println("Request at server: ", req, "from IP", r.RemoteAddr)
	fmt.Println("Header", r.Header)
	req.IpAddress = strings.Split(r.RemoteAddr, ":")[0]

	myIP := getLocalIP()
	if myIP == "" {
		// BLUNDER
	}

	pim := lib.NewPeerInfoManager(peers, completionChan, myIP)
	logger.Debug("Request", req)
	resp := pim.HandleRequest(req)

	fmt.Println("Response: ", resp)
	bytesRepresentation, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
	}
	//Set Content-Type header so that clients will know how to read response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	//Write json response back to response
	w.Write(bytesRepresentation)
}

func parseConfig() []string {
	file, err := os.Open(*config)
	if err != nil {
		logger.Debug("Could not open the config file: ", err)
		return []string{}
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	configuration := PeerConfig{}
	err = decoder.Decode(&configuration)
	if err != nil {
		logger.Debug("Could not decode the config into json", err)
		return []string{}
	}
	return configuration.PeersList
}

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
	// fmt.Println("Starting transfer to other peers")
	// err := sendSeederPush()
	// if err != nil {
	// 	logger.Debug("SeederPush failed: ", err)
	// }

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

	// Trigger the run function to
	// i) Parse Config.
	// ii) Spawn a tracker(Peer Info Manager)
	// iii) Send SeederPush to myself.
	// iv) Wait on channel.
	run()
}

package lib

import "sync"

type PeerInfoManager struct {
	Seeder   *DCL
	Active   *DCL
	Inactive *DCL
	PsyncCh  chan int //Channel to notfiy psync
	mu       sync.Mutex
}

func NewPeerInfoManager(ips []string, ch chan int, starterIp string) *PeerInfoManager {
	seeder := NewDCL()
	active := NewDCL()
	// Appending Starter IP to active
	active.Append(starterIp)
	inactive := NewDCL()
	GetLogger().Debug("IPS %v\n", ips)
	for _, ip := range ips {
		// initially all are inactive
		inactive.Append(ip)
	}
	return &PeerInfoManager{Seeder: seeder, Active: active, Inactive: inactive, PsyncCh: ch}
}

func (pi *PeerInfoManager) handleSeeder(ipAddress string, numOfPeers int) PeerInfoManagerResponseMsg {
	// Put ip address in seeder list, remove it from active list
	// Give it the specified number of inactive peers.
	pi.mu.Lock()
	defer pi.mu.Unlock()
	GetLogger().Debug("inactive=%v, active=%v, seeder=%v\n", pi.Inactive, pi.Active, pi.Seeder)

	// This is to make sure all peer ips are unique, because our DCL will
	// keep rolling back.
	peerSet := NewSet()

	for i := 0; i < numOfPeers; i++ {
		peerSet.Add(pi.Inactive.Next())
	}
	peerSet.Remove("")
	peerSet.Remove(ipAddress)
	GetLogger().Debug("PeerSet: %v", peerSet)
	response := PeerInfoManagerResponseMsg{}
	if peerSet.Length() == 0 {
		GetLogger().Debug("No peers could be found for ip %v\n", ipAddress)
		response.Err = "No peers"
	} else {
		response.Peers = peerSet.List()
	}
	//GetLogger().Debug("Response: %v\n", ipAddress)
	pi.Active.Remove(ipAddress)
	pi.Seeder.Append(ipAddress)
	GetLogger().Debug("inactive=%v, active=%v, seeder=%v\n", pi.Inactive, pi.Active, pi.Seeder)
	// TBD - Check if we no longer have inactive peers and have all seeders.
	// If yes, notify psync that all peers are done transferring
	if pi.Inactive.Length() == 0 && pi.Active.Length() == 0 {
		GetLogger().Debug("All peers are now seeders\n")
		go func() {
			pi.PsyncCh <- 1
			GetLogger().Debug("Test log\n")
		}()
	}
	return response
}

func (pi *PeerInfoManager) handleActiveNode(ipAddress string, numOfPeers int, peers map[string]bool) PeerInfoManagerResponseMsg {
	// Put ip address in active list, remove it from inactive list
	// Give it 1 seeder and and numOfPeers -1 active peers
	//
	pi.mu.Lock()
	defer pi.mu.Unlock()
	GetLogger().Debug("Entry-inactive=%v, active=%v, seeder=%v\n", pi.Inactive, pi.Active, pi.Seeder)
	GetLogger().Debug("Peers which I already have %v\n", peers)

	peerSet := NewSet()

	//  Find active peers. If can't find, then find seeders.

	for i := 0; i < pi.Active.Length() && peerSet.Length() <= numOfPeers; i++ {
		s := pi.Active.Next()
		if !peers[s] {
			peerSet.Add(s)
		}
	}

	// Add a seeder here, if not in the list of peers
	for i := 0; i < pi.Seeder.Length() && peerSet.Length() <= numOfPeers; i++ {
		s := pi.Seeder.Next()
		if !peers[s] {
			peerSet.Add(s)
		}
	}
	peerSet.Remove("")
	peerSet.Remove(ipAddress)
	response := PeerInfoManagerResponseMsg{}
	if peerSet.Length() == 0 {
		GetLogger().Debug("No peers could be found for ip %v\n", ipAddress)
		response.Err = "No peers"
	} else {
		response.Peers = peerSet.List()
	}
	pi.Inactive.Remove(ipAddress)
	pi.Active.Append(ipAddress)
	GetLogger().Debug("Exit-inactive=%v, active=%v, seeder=%v\n", pi.Inactive, pi.Active, pi.Seeder)
	return response
}

func (pi *PeerInfoManager) HandleRequest(request PeerInfoManagerRequestMsg) PeerInfoManagerResponseMsg {

	switch request.State {
	case Seeder:
		// give back the list of inactive nodes
		return pi.handleSeeder(request.IpAddress, request.NumOfPeers)
	case Active:
		return pi.handleActiveNode(request.IpAddress, request.NumOfPeers, request.Peers)
	default:
		GetLogger().Debug("Invalid state %v\n", request.State)
	}
	return PeerInfoManagerResponseMsg{Err: "Invalid state " + string(request.State)}
}

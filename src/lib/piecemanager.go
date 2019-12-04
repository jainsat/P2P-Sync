package lib

import (
	"math"
	"sync"
)

const NoPiece = -1

type PieceManager struct {
	peerToPiece map[string]map[int]bool
	// peerToDonePiece - not sure if it is needed
	pieceToPeer      map[int]map[string]bool
	piecesInProgress map[string]map[int]bool
	myPieces         map[int]bool

	mu sync.Mutex
}

func NewPieceManager() *PieceManager {
	pieceManager := PieceManager{}
	pieceManager.peerToPiece = make(map[string]map[int]bool)
	pieceManager.pieceToPeer = make(map[int]map[string]bool)
	pieceManager.piecesInProgress = make(map[string]map[int]bool)
	pieceManager.myPieces = make(map[int]bool)
	return &pieceManager
}

func (pm *PieceManager) getPiece(peer string) int {
	// Find piece which is available to very few(min)peers.
	// Remove it from peerToPeer from all peers which have it so that it doesn't get chosen
	// Add this piece to in progress list.
	pm.mu.Lock()
	defer pm.mu.Unlock()
	GetLogger().Debug("Received get piece for peer %v\n", peer)
	_, ok := pm.peerToPiece[peer]

	// If this peer has no pieces
	if !ok {
		return NoPiece
	}

	// Iterate through peerToPiece[peer] and find out which piece is owned by minimum number
	// of peers.
	minPeers := math.MaxInt32
	pieceSelected := -1
	for piece := range pm.peerToPiece[peer] {
		if len(pm.pieceToPeer[piece]) < minPeers {
			minPeers = len(pm.pieceToPeer[piece])
			pieceSelected = piece
		}
	}

	if pieceSelected == -1 {
		// Thsi peer has no pieces
		return NoPiece
	}
	// Add piece to in progress list
	_, ok = pm.piecesInProgress[peer]
	if !ok {
		pm.piecesInProgress[peer] = make(map[int]bool)
	}
	pm.piecesInProgress[peer][pieceSelected] = true

	// Remove it from peerToPiece list for all peers
	peers := pm.pieceToPeer[pieceSelected]

	for peer := range peers {
		delete(pm.peerToPiece[peer], pieceSelected)
	}

	return pieceSelected

}

func (pm *PieceManager) updatePieceInfos(peer string, piece []int) {
	for _, p := range piece {
		pm.updatePieceInfo(peer, p)
	}
}

// Received a have message.
func (pm *PieceManager) updatePieceInfo(peer string, piece int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	GetLogger().Debug("Received update from peer %v for piece %v\n", peer, piece)

	// Check if I already have piece
	if pm.myPieces[piece] {
		GetLogger().Debug("Already have this piece %v\n", piece)
		return
	}
	_, ok := pm.pieceToPeer[piece]
	if !ok {
		pm.pieceToPeer[piece] = make(map[string]bool)
	}
	pm.pieceToPeer[piece][peer] = true

	_, ok = pm.peerToPiece[peer]

	if !ok {
		pm.peerToPiece[peer] = make(map[int]bool)
	}
	pm.peerToPiece[peer][piece] = true
}

func (pm *PieceManager) notify(success bool, peer string, piece int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	GetLogger().Debug("Notify success=%v, peer=%v, piece=%v\n", success, peer, piece)
	delete(pm.piecesInProgress[peer], piece)
	if success {
		// Piece has been successfully received.
		// Remove it from progess list
		// Add it to myPieces
		pm.myPieces[piece] = true
	} else {
		// Failed to receive
		// Remove it from progress list
		// Add it to peerToPiece for all peers which have it
		peers := pm.pieceToPeer[piece]

		for peer := range peers {
			pm.peerToPiece[peer][piece] = true
		}
	}
}

func (pm *PieceManager) getNumOfInProgressPieces(peer string) int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return len(pm.piecesInProgress[peer])
}

func (pm *PieceManager) havePiece(piece int) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.myPieces[piece]
}

func (pm *PieceManager) getPeers(piece int) []int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	peers := make([]int, 0)
	for p := range pm.pieceToPeer {
		peers = append(peers, p)
	}
	return peers

}

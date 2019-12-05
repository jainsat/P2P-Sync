package lib

import (
	"math"
	"sync"
)

const NoPiece = -1

type PieceManager struct {
	peerToPiece map[string]map[int]bool
	// peerToDonePiece - not sure if it is needed
	pieceToPeer        map[int]map[string]bool
	piecesInProgress   map[string]map[int]bool
	myPieces           map[int]bool
	allPiecesInProgess map[int]bool
	mu                 sync.Mutex
}

func NewPieceManager() *PieceManager {
	pieceManager := PieceManager{}
	pieceManager.peerToPiece = make(map[string]map[int]bool)
	pieceManager.pieceToPeer = make(map[int]map[string]bool)
	pieceManager.piecesInProgress = make(map[string]map[int]bool)
	pieceManager.myPieces = make(map[int]bool)
	pieceManager.allPiecesInProgess = make(map[int]bool)

	return &pieceManager
}

func (pm *PieceManager) GetTotalCurrentPieces() int64 {
	return int64(len(pm.myPieces))
}

func (pm *PieceManager) GetPiece(peer string) int {
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
	pm.allPiecesInProgess[pieceSelected] = true

	// Remove it from peerToPiece list for all peers
	peers := pm.pieceToPeer[pieceSelected]

	for peer := range peers {
		delete(pm.peerToPiece[peer], pieceSelected)
	}

	return pieceSelected

}

func (pm *PieceManager) UpdatePieceInfos(peer string, piece []int) {
	for _, p := range piece {
		pm.UpdatePieceInfo(peer, p)
	}
}

// Received a have message.
func (pm *PieceManager) UpdatePieceInfo(peer string, piece int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	GetLogger().Debug("Received update from peer %v for piece %v\n", peer, piece)

	// Check if I already have piece
	if pm.myPieces[piece] || pm.allPiecesInProgess[piece] {
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

func (pm *PieceManager) Notify(success bool, peer string, piece int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	GetLogger().Debug("Notify success=%v, peer=%v, piece=%v\n", success, peer, piece)
	delete(pm.piecesInProgress[peer], piece)
	delete(pm.allPiecesInProgess, piece)

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

func (pm *PieceManager) ReceivedAllPieces(totalPieces int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	GetLogger().Debug("Received all pieces.\n")
	for i := 0; i < totalPieces; i++ {
		pm.myPieces[i] = true
	}
}

func (pm *PieceManager) GetNumOfInProgressPieces(peer string) int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return len(pm.piecesInProgress[peer])
}

func (pm *PieceManager) HavePiece(piece int) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.myPieces[piece]
}

func (pm *PieceManager) GetPeers(piece int) map[string]bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.pieceToPeer[piece]
}

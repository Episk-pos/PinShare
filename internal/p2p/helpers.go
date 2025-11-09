package p2p

import (
	"fmt"
	"pinshare/internal/store"
)

// Global P2PManager instance
var globalP2PManager *PubSubManager

// SetGlobalP2PManager sets the global PubSubManager instance
func SetGlobalP2PManager(psm *PubSubManager) {
	globalP2PManager = psm
}

// PublishMetadataUpdate publishes a metadata update to the P2P network
// This is a convenience function for use by other packages
func PublishMetadataUpdate(metadata store.BaseMetadata) error {
	if globalP2PManager == nil {
		// P2P not initialized, silently skip
		fmt.Println("[WARNING] P2P not initialized, skipping metadata publish")
		return nil
	}

	return globalP2PManager.PublishMetadata(metadata)
}

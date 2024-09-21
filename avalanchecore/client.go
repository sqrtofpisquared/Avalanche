package avalanchecore

import (
	"github.com/google/uuid"
	"net"
)

type ClientCapability struct {
}

type remoteStream struct {
	// Entry in stream table
}

type avalancheClient struct {
	ClientID     uuid.UUID
	Destination  net.Addr
	Capabilities []ClientCapability
	ASPVersion   uint8
}

type remoteClient struct {
	avalancheClient
	LastSeenTimestamp uint64
	Quality           LinkQuality
}

type localClient struct {
	avalancheClient
	ClientTable map[uuid.UUID]remoteClient
	StreamTable map[uint16]remoteStream
}

func InitializeClient(cmnAddress string) {
	var c localClient

	cmn := cmnConnect(cmnAddress)

	if err := cmn.sendAnnouncement(c.avalancheClient); err != nil {
		// TODO handle announcement failure
	}

}

func (client localClient) InitStream() AvalancheStream {
	var s AvalancheStream

	// Communicate with CMN to agree on a new AvalancheStream ID

	// Reference client table to find destination client

	// Negotiate connection with destination client over CMN

	// Announce stream to CMN

	// Perform AvalancheStream handshake

	return s
}
